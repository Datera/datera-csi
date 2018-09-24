package driver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	grpc "google.golang.org/grpc"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
	udc "github.com/Datera/go-udc/pkg/udc"
)

const (
	driverName    = "io.daterainc.csi.dsp"
	vendorVersion = "0.1.0"

	// Environment Variables
	EnvVolPerNode       = "DAT_VOL_PER_NODE"
	EnvDisableMultipath = "DAT_DISABLE_MULTIPATH"
	EnvReplicaOverride  = "DAT_REPLICA_OVERRIDE"
)

type EnvVars struct {
	VolPerNode       int
	DisableMultipath bool
	ReplicaOverride  bool
}

func readEnvVars() *EnvVars {
	vpn, err := strconv.ParseInt(os.Getenv(EnvVolPerNode), 0, 0)
	if err != nil {
		vpn = int64(256)
	}
	var dm bool
	if d := os.Getenv(EnvDisableMultipath); d != "" {
		dm = true
	}
	var ro bool
	if d := os.Getenv(EnvReplicaOverride); d != "" {
		ro = true
	}
	return &EnvVars{
		VolPerNode:       int(vpn),
		DisableMultipath: dm,
		ReplicaOverride:  ro,
	}
}

// Driver is a single-binary implementation of:
//   * csi.ControllerServer
//   * csi.IdentityServer
//   * csi.NodeServer
type Driver struct {
	gs  *grpc.Server
	dc  *dc.DateraClient
	env *EnvVars
	nid string

	sock string
}

func NewDateraDriver(sock string, udc *udc.UDC) (*Driver, error) {
	client, err := dc.NewDateraClient(udc)
	if err != nil {
		return nil, err
	}
	env := readEnvVars()
	return &Driver{
		dc:   client,
		sock: sock,
		env:  env,
		nid:  co.GetHost(),
	}, nil
}

func (d *Driver) Run() error {
	ctxt := co.WithCtxt(context.Background(), "Run")
	co.Infof(ctxt, "Starting CSI driver\n")

	u, err := url.Parse(d.sock)
	if err != nil {
		return err
	}
	if u.Scheme != "unix" {
		return fmt.Errorf("Only unix sockets are supported by CSI")
	}
	addr := path.Join(u.Host, filepath.FromSlash(u.Path))
	if u.Host == "" {
		addr = filepath.FromSlash(u.Path)
	}
	co.Infof(ctxt, "Removing socket: %s\n", addr)
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		co.Errorf(ctxt, "Failed to remove unix domain socket file: %s", addr)
		return err
	}
	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		co.Errorf(ctxt, "Error starting listener for address: %s", addr)
		return err
	}
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(logServer))
	csi.RegisterControllerServer(d.gs, d)
	csi.RegisterIdentityServer(d.gs, d)
	csi.RegisterNodeServer(d.gs, d)
	co.Infof(ctxt, "Datera CSI Driver Serving On Socket: %s\n", addr)
	return d.gs.Serve(listener)
}

func (d *Driver) Stop() {
	ctxt := co.WithCtxt(context.Background(), "Stop")
	co.Info(ctxt, "Datera CSI driver stopped")
	d.gs.Stop()
}

func logServer(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctxt := co.WithCtxt(ctx, "rpc")
	co.Infof(ctxt, "GRPC -- request: %s -- %+v\n", info.FullMethod, req)
	resp, err := handler(ctx, req)
	co.Infof(ctxt, "GRPC -- response: %s -- %+v\n", info.FullMethod, resp)
	if err != nil {
		co.Errorf(ctxt, "GRPC -- error: %s -- %+v\n", info.FullMethod, err)
	}
	return resp, err
}

func (d *Driver) InitFunc(ctx context.Context, piece, funcName string, req interface{}) context.Context {
	ctxt := co.WithCtxt(ctx, fmt.Sprintf("%s.%s", piece, funcName))
	d.dc.WithContext(ctxt)
	co.Infof(ctxt, "%s service '%s' called\n", piece, funcName)
	co.Debugf(ctxt, "%s: %+v\n", funcName, req)
	return ctxt
}

func RegisterVolumeCapability(ctxt context.Context, md *dc.VolMetadata, vc *csi.VolumeCapability) {
	// Record req.VolumeCapabilities in metadata We don't actually do anything
	// with this information because it's all the same to us, but we should
	// keep it for future product filtering/aggregate operations
	if vc == nil {
		co.Warningf(ctxt, "VolumeCapability is nil")
		return
	}
	var (
		at string
		fs string
		mo string
	)
	if vc.GetAccessMode() != nil {
		mo = vc.GetAccessMode().Mode.String()
	}
	switch vc.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		at = "block"
	case *csi.VolumeCapability_Mount:
		at = "mount"
		fs = vc.GetMount().FsType + " " + strings.Join(vc.GetMount().MountFlags, "")
		co.Debugf(ctxt, "Registering Filesystem %s", fs)
	default:
		at = "unknown"
	}
	co.Debugf(ctxt, "Registering VolumeCapability %s", at)
	co.Debugf(ctxt, "Registering VolumeCapability %s", mo)
	(*md)["access-type"] = at
	(*md)["access-fs"] = fs
	(*md)["access-mode"] = mo
	co.Debugf(ctxt, "VolumeMetadata: %#v", *md)
}

func GetClientForTests(d *Driver) *dc.DateraClient {
	return d.dc
}

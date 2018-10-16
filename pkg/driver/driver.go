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
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	grpc "google.golang.org/grpc"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
	udc "github.com/Datera/go-udc/pkg/udc"
)

const (
	driverName = "io.daterainc.csi.dsp"

	// Environment Variables
	EnvSocket           = "DAT_SOCKET"
	EnvHeartbeat        = "DAT_HEARTBEAT"
	EnvType             = "DAT_TYPE"
	EnvVolPerNode       = "DAT_VOL_PER_NODE"
	EnvDisableMultipath = "DAT_DISABLE_MULTIPATH"
	EnvReplicaOverride  = "DAT_REPLICA_OVERRIDE"
	EnvMetadataDebug    = "DAT_METADATA_DEBUG"
	EnvDisableLogPush   = "DAT_DISABLE_LOGPUSH"
	EnvLogPushInterval  = "DAT_LOGPUSH_INTERVAL"

	IdentityType = iota + 1
	ControllerType
	NodeType
	NodeIdentityType
	ControllerIdentityType
	AllType
)

var (
	DefaultSocket = fmt.Sprintf("unix:///var/lib/kubelet/plugins/%s/csi.sock", driverName)
	StrToType     = map[string]int{
		"identity":   IdentityType,
		"controller": ControllerType,
		"node":       NodeType,
		"nodeident":  NodeIdentityType,
		"conident":   ControllerIdentityType,
		"all":        AllType,
	}
	Version = "No Version Provided"
	Githash = "No Githash Provided"
)

type EnvVars struct {
	Socket           string
	Type             int
	VolPerNode       int
	DisableMultipath bool
	ReplicaOverride  bool
	Heartbeat        int
	MetadataDebug    bool
	LogPush          bool
	LogPushInterval  int
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
	var so string
	if so = os.Getenv(EnvSocket); so == "" {
		so = DefaultSocket
	}
	hb64, err := strconv.ParseInt(os.Getenv(EnvHeartbeat), 0, 0)
	if err != nil {
		hb64 = int64(60)
	}
	var mdd bool
	if d := os.Getenv(EnvDisableMultipath); d != "" {
		mdd = true
	}
	lp := true
	if d := os.Getenv(EnvDisableLogPush); d != "" && d != "false" {
		lp = false
	}
	lpi, err := strconv.ParseInt(os.Getenv(EnvLogPushInterval), 0, 0)
	if err != nil {
		lpi = int64(time.Hour * 12)
	}
	return &EnvVars{
		VolPerNode:       int(vpn),
		DisableMultipath: dm,
		ReplicaOverride:  ro,
		Socket:           so,
		Type:             StrToType[os.Getenv(EnvType)],
		Heartbeat:        int(hb64),
		MetadataDebug:    mdd,
		LogPush:          lp,
		LogPushInterval:  int(lpi),
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

	sock    string
	version string
}

func NewDateraDriver(udc *udc.UDC) (*Driver, error) {
	env := readEnvVars()
	v := fmt.Sprintf("%s-%s", Version, Githash)
	client, err := dc.NewDateraClient(udc, false, v)
	if err != nil {
		return nil, err
	}
	dc.MetadataDebug = env.MetadataDebug
	return &Driver{
		dc:   client,
		sock: env.Socket,
		env:  env,
		nid:  co.GetHost(),
	}, nil
}

func (d *Driver) Run() error {
	ctxt := co.WithCtxt(context.Background(), "Run")
	co.Infof(ctxt, "Starting CSI driver\n")

	co.Infof(ctxt, "Parsing socket: %s\n", d.sock)
	u, err := url.Parse(d.sock)
	co.Debugf(ctxt, "Parsed socket: %#v\n", u)
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
	co.Debugf(ctxt, "Checking for file: %s\n", addr)
	if _, err := os.Stat(addr); os.IsNotExist(err) {
		co.Debugf(ctxt, "Creating directories: %s\n", addr)
		err = os.MkdirAll(addr, os.ModePerm)
		if err != nil {
			return err
		}
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
	if d.env.Type == ControllerType || d.env.Type == ControllerIdentityType || d.env.Type == AllType {
		co.Info(ctxt, "Starting 'controller' service\n")
		csi.RegisterControllerServer(d.gs, d)
	}
	if d.env.Type == IdentityType || d.env.Type == NodeIdentityType || d.env.Type == ControllerIdentityType || d.env.Type == AllType {
		co.Info(ctxt, "Starting 'identity' service\n")
		csi.RegisterIdentityServer(d.gs, d)
	}
	if d.env.Type == NodeType || d.env.Type == NodeIdentityType || d.env.Type == AllType {
		co.Info(ctxt, "Starting 'node' service\n")
		csi.RegisterNodeServer(d.gs, d)
	}
	co.Infof(ctxt, "Datera CSI Driver Serving On Socket: %s\n", addr)
	go d.Heartbeater()
	go d.LogPusher()
	return d.gs.Serve(listener)
}

func (d *Driver) Stop() {
	ctxt := co.WithCtxt(context.Background(), "Stop")
	co.Info(ctxt, "Datera CSI driver stopped")
	d.gs.Stop()
}

func (d *Driver) Heartbeater() {
	ctxt := co.WithCtxt(context.Background(), "Heartbeat")
	co.Infof(ctxt, "Starting heartbeat service. Interval: %d", d.env.Heartbeat)
	t := int(time.Second * time.Duration(d.env.Heartbeat))
	for {
		if err := d.dc.HealthCheck(); err != nil {
			co.Errorf(ctxt, "Heartbeat failure: %s\n", err)
		}
		Sleeper(t)
	}
}

func (d *Driver) LogPusher() {
	ctxt := co.WithCtxt(context.Background(), "LogPusher")
	co.Infof(ctxt, "Starting LogPusher service. Interval: %d", d.env.Heartbeat)
	t := int(time.Second * time.Duration(d.env.LogPushInterval))
	for {
		if err := d.dc.LogPush(ctxt, "/var/log/driver.log", "/var/log/driver.log.1"); err != nil {
			co.Errorf(ctxt, "LogPush failure: %s\n", err)
		}
		Sleeper(t)
	}
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
		at     string
		fs     string
		fsargs string
		mo     string
	)
	if vc.GetAccessMode() != nil {
		mo = vc.GetAccessMode().Mode.String()
	}
	switch vc.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		at = "block"
	case *csi.VolumeCapability_Mount:
		at = "mount"
		fs = vc.GetMount().FsType
		fsargs = strings.Join(vc.GetMount().MountFlags, "")
		co.Debugf(ctxt, "Registering Filesystem %s %s", fs, fsargs)
	default:
		at = "unknown"
	}
	co.Debugf(ctxt, "Registering VolumeCapability %s", at)
	co.Debugf(ctxt, "Registering VolumeCapability %s", mo)
	(*md)["access_type"] = at
	if fs != "" {
		(*md)["fs_type"] = fs
		(*md)["fs_args"] = fsargs
	}
	(*md)["access_mode"] = mo
	co.Debugf(ctxt, "VolumeMetadata: %#v", *md)
}

func GetClientForTests(d *Driver) *dc.DateraClient {
	return d.dc
}

// For finer grained sleeping, interval is specified in seconds
func Sleeper(interval int) {
	for {
		if interval <= 0 {
			break
		}
		time.Sleep(time.Second * 1)
		interval--
	}
}

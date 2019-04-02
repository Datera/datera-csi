package driver

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	grpc "google.golang.org/grpc"
	gmd "google.golang.org/grpc/metadata"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
	udc "github.com/Datera/go-udc/pkg/udc"
)

const (
	driverName = "dsp.csi.daterainc.io"

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
	EnvFormatTimeout    = "DAT_FORMAT_TIMEOUT"

	Ext4 = "ext4"
	Xfs  = "xfs"

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
	Version          = "No Version Provided"
	Githash          = "No Githash Provided"
	SdkVersion       = "No SdkVersion Provided"
	SupportedFsTypes = map[string]struct{}{
		Ext4: struct{}{},
		Xfs:  struct{}{},
	}
	DefaultFsArgs = map[string]string{
		Ext4: "-E lazy_itable_init=0,lazy_journal_init=0,nodiscard -F",
		Xfs:  "",
	}
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
	FormatTimeout    int
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
		// Default is to run logpusher every 2 hours
		lpi = int64((time.Hour * 2) / time.Second)
	}
	ft, err := strconv.ParseInt(os.Getenv(EnvFormatTimeout), 0, 0)
	if err != nil {
		ft = int64(60)
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
		FormatTimeout:    int(ft),
	}
}

func isSupportedFs(fs string) bool {
	_, ok := SupportedFsTypes[fs]
	return ok
}

func supportedFsTypes() []string {
	types := []string{}
	for k := range SupportedFsTypes {
		types = append(types, k)
	}
	return types
}

// Driver is a single-binary implementation of:
//   * csi.ControllerServer
//   * csi.IdentityServer
//   * csi.NodeServer
type Driver struct {
	gs            *grpc.Server
	dc            *dc.DateraClient
	env           *EnvVars
	nid           string
	healthy       bool
	vendorVersion string
	manifest      *dc.Manifest
	rpcStatus     map[string]struct{}

	sock    string
	version string
}

func NewDateraDriver(udc *udc.UDC) (*Driver, error) {
	env := readEnvVars()
	v := fmt.Sprintf("datera-csi-%s-%s-gosdk-%s", Version, Githash, SdkVersion)
	client, err := dc.NewDateraClient(udc, false, v)
	if err != nil {
		return nil, err
	}
	dc.MetadataDebug = env.MetadataDebug
	return &Driver{
		dc:        client,
		sock:      env.Socket,
		env:       env,
		nid:       co.GetHost(),
		version:   Version,
		rpcStatus: map[string]struct{}{},
	}, nil
}

func (d *Driver) Run() error {
	ctxt := co.WithCtxt(context.Background(), "Run", "")
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
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(logServerAndSetId))
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
	if d.env.LogPush {
		go d.LogPusher()
	}
	return d.gs.Serve(listener)
}

func (d *Driver) Stop() {
	ctxt := co.WithCtxt(context.Background(), "Stop", "")
	co.Info(ctxt, "Datera CSI driver stopped")
	d.gs.Stop()
}

func (d *Driver) Heartbeater() {
	ctxt := co.WithCtxt(context.Background(), "Heartbeat", "")
	co.Infof(ctxt, "Starting heartbeat service. Interval: %d", d.env.Heartbeat)
	t := d.env.Heartbeat
	for {
		if mf, err := d.dc.HealthCheck(ctxt); err != nil {
			d.healthy = false
			d.manifest = mf
			d.vendorVersion = mf.BuildVersion
			co.Errorf(ctxt, "Heartbeat failure: %s\n", err)
		} else {
			d.healthy = true
		}
		Sleeper(t)
	}
}

func (d *Driver) LogPusher() {
	ctxt := co.WithCtxt(context.Background(), "LogPusher", "")
	co.Infof(ctxt, "Starting LogPusher service. Interval: %d", d.env.LogPushInterval)
	t := d.env.LogPushInterval
	// Give the driver a chance to start before doing first log collect
	Sleeper(10)
	for {
		if err := d.dc.LogPush(ctxt, "/etc/logrotate.d/driver-logrotate", "/var/log/driver.log.1.gz"); err != nil {
			co.Errorf(ctxt, "LogPush failure: %s\n", err)
		}
		Sleeper(t)
	}
}

func logServerAndSetId(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	id := co.GenId()
	ctxt := co.WithCtxt(ctx, "rpc", id)
	ctxt = gmd.AppendToOutgoingContext(ctxt, "datera-request-id", id)
	co.Infof(ctxt, "GRPC -- request: %s -- %s -- %+v\n", info.FullMethod, id, req)
	ts1 := time.Now()
	resp, err := handler(ctxt, req)
	ts2 := time.Now()
	td := math.Round(float64(ts2.Sub(ts1) / time.Second))
	co.Infof(ctxt, "GRPC -- response: %s -- %s %ds -- %+v\n", info.FullMethod, id, td, resp)
	if err != nil {
		co.Errorf(ctxt, "GRPC -- error: %s -- %s -- %+v\n", info.FullMethod, id, err)
	}
	return resp, err
}

func (d *Driver) InitFunc(ctx context.Context, piece, funcName string, req interface{}) (context.Context, bool, func()) {
	id := ctx.Value(co.TraceId).(string)
	// Sets trace id in driver
	ctxt := co.WithCtxt(ctx, fmt.Sprintf("%s.%s", piece, funcName), id)
	// Sets trace id in client
	ctxt = d.dc.WithContext(ctxt)
	// We're not going to log the identity calls because they're really verbose with the
	// liveness probe sidecar
	if piece != "identity" {
		co.Infof(ctxt, "%s service '%s' called\n", piece, funcName)
		co.Debugf(ctxt, "%s: %+v\n", funcName, req)
	}
	key := strings.Join([]string{piece, funcName, fmt.Sprintf("%+v", req)}, "|")
	inProgress := false
	cleaner := func() {}
	if _, ok := d.rpcStatus[key]; ok {
		inProgress = true
	} else {
		d.rpcStatus[key] = struct{}{}
		cleaner = func() {
			delete(d.rpcStatus, key)
		}
	}
	return ctxt, inProgress, cleaner
}

func RegisterVolumeCapability(ctxt context.Context, md *dc.VolMetadata, vc *csi.VolumeCapability) error {
	// Record req.VolumeCapabilities in metadata We don't actually do anything
	// with this information because it's all the same to us, but we should
	// keep it for future product filtering/aggregate operations
	if vc == nil {
		err := fmt.Errorf("VolumeCapability is nil")
		co.Warning(ctxt, err)
		return err
	}
	var (
		at        string
		fs        string
		mountArgs string
		mo        string
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
		if !isSupportedFs(fs) {
			err := fmt.Errorf("Unsupported filesystem type: %s, supported types :re %s", fs, supportedFsTypes())
			co.Error(ctxt, err)
			return err
		}
		co.Debugf(ctxt, "Registering Filesystem %s", fs)
		mountArgs = strings.Join(vc.GetMount().MountFlags, "")
		co.Debugf(ctxt, "Registering MountFlags %s", mountArgs)
	default:
		return fmt.Errorf("Unsupported VolumeCapability: %s.  Supported capabilities are Mount and Block", fs)
	}
	co.Debugf(ctxt, "Registering VolumeCapability %s", at)
	co.Debugf(ctxt, "Registering VolumeCapability %s", mo)
	(*md)["access_type"] = at
	if fs != "" {
		(*md)["fs_type"] = fs
		(*md)["m_flags"] = mountArgs
	}
	(*md)["access_mode"] = mo
	co.Debugf(ctxt, "VolumeMetadata: %#v", *md)
	return nil
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

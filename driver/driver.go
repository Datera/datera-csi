package driver

import (
	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

const (
	driverName    = "io.daterainc.csi.dsp"
	vendorVersion = "0.1.0"
)

// Driver is a single-binary implementation of:
//   * csi.ControllerServer
//   * csi.IdentityServer
//   * csi.NodeServer
type Driver struct {
	gs *grpc.Server
	// dc *datera.Client
	dc  struct{}
	nid string

	Sock     string
	Url      string
	Username string
	Password string
}

func NewDateraDriver(sock, username, password, url string) (*Driver, error) {
	return &Driver{
		Sock:     sock,
		Username: username,
		Password: password,
		Url:      url,
	}, nil
}

func (d *Driver) Run() error {
	log.WithField("method", "driver.Run").Infof("Starting CSI driver")
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(logServer))
	csi.RegisterControllerServer(d.gs, d)
	csi.RegisterIdentityServer(d.gs, d)
	csi.RegisterNodeServer(d.gs, d)
	return nil
}

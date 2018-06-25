package driver

import (
	"context"

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
	errf := func(ctxt context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctxt, req)
		if err != nil {
			log.WithError(err).WithField("method", info.FullMethod).Error("method failed")
		}
		return resp, err
	}
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(errf))
	csi.RegisterControllerServer(d.gs, d)
	csi.RegisterIdentityServer(d.gs, d)
	csi.RegisterNodeServer(d.gs, d)
	return nil
}

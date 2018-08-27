package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"

	client "github.com/Datera/datera-csi/pkg/client"
	udc "github.com/Datera/go-udc/pkg/udc"
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
	gs  *grpc.Server
	dc  *client.DateraClient
	nid string

	Sock string
}

func NewDateraDriver(sock string, udc *udc.UDC) (*Driver, error) {
	client, err := client.NewDateraClient(udc)
	if err != nil {
		return nil, err
	}
	return &Driver{
		dc:   client,
		Sock: sock,
	}, nil
}

func (d *Driver) Run() error {
	log.WithField("method", "driver.Run").Infof("Starting CSI driver\n")
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(logServer))
	csi.RegisterControllerServer(d.gs, d)
	csi.RegisterIdentityServer(d.gs, d)
	csi.RegisterNodeServer(d.gs, d)
	return nil
}

func (d *Driver) Stop() {
	log.Info("Datera CSI driver stopped")
	d.gs.Stop()
}

func logServer(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.WithField("method", info.FullMethod).Infof("GRPC -- request: %+v", req)
	resp, err := handler(ctx, req)
	log.WithField("method", info.FullMethod).Infof("GRPC -- response: %+v", resp)
	if err != nil {
		log.WithField("method", info.FullMethod).Infof("GRPC -- error: %+v", err)
	}
	return resp, err
}

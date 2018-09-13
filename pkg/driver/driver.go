package driver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"

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

	sock string
}

func NewDateraDriver(sock string, udc *udc.UDC) (*Driver, error) {
	client, err := client.NewDateraClient(udc)
	if err != nil {
		return nil, err
	}
	return &Driver{
		dc:   client,
		sock: sock,
	}, nil
}

func (d *Driver) Run() error {
	log.WithField("method", "driver.Run").Infof("Starting CSI driver\n")

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
	log.Infof("Removing socket: %s\n", addr)
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		log.Errorf("Failed to remove unix domain socket file: %s", addr)
		return err
	}
	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		log.Errorf("Error starting listener for address: %s", addr)
		return err
	}
	d.gs = grpc.NewServer(grpc.UnaryInterceptor(logServer))
	csi.RegisterControllerServer(d.gs, d)
	csi.RegisterIdentityServer(d.gs, d)
	csi.RegisterNodeServer(d.gs, d)
	log.Infof("Serving socket: %s\n", addr)
	return d.gs.Serve(listener)
}

func (d *Driver) Stop() {
	log.Info("Datera CSI driver stopped")
	d.gs.Stop()
}

func logServer(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.WithField("method", info.FullMethod).Infof("GRPC -- request: %s -- %+v\n", info.FullMethod, req)
	resp, err := handler(ctx, req)
	log.WithField("method", info.FullMethod).Infof("GRPC -- response: %s -- %+v\n", info.FullMethod, resp)
	if err != nil {
		log.WithField("method", info.FullMethod).Infof("GRPC -- error: %s -- %+v\n", info.FullMethod, err)
	}
	return resp, err
}

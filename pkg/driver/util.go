package driver

import (
	"context"
	"strings"

	uuid "github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

func waitForAttach() {
}

func waitForOnline() {
}

func genVolName(name string) string {
	if name == "" {
		name = uuid.Must(uuid.NewRandom()).String()
	}
	return strings.Join([]string{"CSI", name}, "-")
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

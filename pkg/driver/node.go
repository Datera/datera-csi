package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	log.WithField("method", "node_stage_volume").Infof("Node server %s 'NodeStageVolume' called", d.nid)
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{
		NodeId: d.nid,
	}, nil
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	d.InitFunc(ctx, "node", "NodeGetInfo", *req)
	log.WithField("method", "node_get_info").Infof("Node server %s 'NodeGetInfo' called", d.nid)
	return &csi.NodeGetInfoResponse{
		NodeId:             d.nid,
		MaxVolumesPerNode:  10000,
		AccessibleTopology: nil,
	}, nil
}

func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	d.InitFunc(ctx, "node", "NodeGetVolumeStats", *req)
	v, err := d.dc.GetVolume(req.VolumeId, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	size, used, avail := v.GetUsage()
	return &csi.NodeGetVolumeStatsResponse{
		Usage: []*csi.VolumeUsage{
			&csi.VolumeUsage{
				Available: int64(avail * units.GiB),
				Total:     int64(size * units.GiB),
				Used:      int64(used * units.GiB),
				Unit:      csi.VolumeUsage_BYTES,
			},
		},
	}, nil
}

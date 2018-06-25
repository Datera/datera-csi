package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	log "github.com/sirupsen/logrus"
)

func (d *Driver) NodeStageVolume(ctxt context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	log.WithField("method", "node_stage_volume").Infof("Node server %s 'NodeStageVolume' called", d.nid)
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctxt context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctxt context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctxt context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetId(ctxt context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{
		NodeId: d.nid,
	}, nil
}

func (d *Driver) NodeGetCapabilities(ctxt context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			&csi.NodeServiceCapability{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) NodeGetInfo(ctxt context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	log.WithField("method", "node_get_info").Infof("Node server %s 'NodeGetInfo' called", d.nid)
	return &csi.NodeGetInfoResponse{
		NodeId:             d.nid,
		MaxVolumesPerNode:  10000,
		AccessibleTopology: nil,
	}, nil
}

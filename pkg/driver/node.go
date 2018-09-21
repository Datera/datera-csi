package driver

import (
	"context"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
)

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeStageVolume", *req)
	md := &dc.VolMetadata{}
	vc := req.VolumeCapability
	RegisterVolumeCapability(ctxt, md, vc)
	vid := req.VolumeId
	vol, err := d.dc.GetVolume(vid, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	// Setup ACL
	init, err := d.dc.CreateGetInitiator()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	if err = vol.RegisterAcl(init); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	// Login to target
	err = vol.Login(!d.env.DisableMultipath)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	// TODO: Add support for Block capability when CSI officially suports it
	if fs, ok := (*md)["access-fs"]; ok {
		// Mount Device
		parts := strings.Split(fs, " ")
		fsType, fsArgs := parts[0], parts[1:]
		err = vol.Format(fsType, fsArgs)
		if err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
		err = vol.Mount(req.StagingTargetPath, []string{})
		if err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeUnstageVolume", *req)
	vid := req.VolumeId
	vol, err := d.dc.GetVolume(vid, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	// Don't return an error for failures to unmount or logout (fail gracefully)
	// We log the errors so if something did go wrong we can track it down without bringing
	// everything to a halt
	err = vol.Unmount()
	if err != nil {
		co.Warning(ctxt, err)
	}
	err = vol.Logout()
	if err != nil {
		co.Warning(ctxt, err)
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	d.InitFunc(ctx, "node", "NodeGetId", *req)
	return &csi.NodeGetIdResponse{
		NodeId: d.nid,
	}, nil
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	d.InitFunc(ctx, "node", "NodeGetVolumeStats", *req)
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
		MaxVolumesPerNode:  int64(d.env.VolPerNode),
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

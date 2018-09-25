package driver

import (
	"context"
	"os"
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
	if vc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	RegisterVolumeCapability(ctxt, md, vc)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "StagingTargetPath cannot be empty")
	}
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
		vol.Formatted = true
		(*md)["formatted"] = "true"
		err = vol.Mount(req.StagingTargetPath, []string{})
		if err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
		(*md)["mount"] = vol.MountPath
		if _, err = vol.SetMetadata(md); err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeUnstageVolume", *req)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	vol, err := d.dc.GetVolume(vid, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	vol.MountPath = (*md)["mount"]
	// Don't return an error for failures to unmount or logout (fail gracefully)
	// We log the errors so if something did go wrong we can track it down without bringing
	// everything to a halt
	err = vol.Unmount()
	if err != nil {
		co.Warning(ctxt, err)
	}
	(*md)["mount"] = ""
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	err = vol.Logout()
	if err != nil {
		co.Warning(ctxt, err)
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodePublishVolume", *req)
	if _, e := os.Stat(req.StagingTargetPath); req.StagingTargetPath == "" || os.IsNotExist(e) {
		return nil, status.Errorf(codes.NotFound, "StagingTargetPath does not exist on this host: %s", req.StagingTargetPath)
	}
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "StagingTargetPath cannot be empty")
	}
	if req.TargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "TargetPath cannot be empty")
	}
	vol, err := d.dc.GetVolume(vid, false)
	vc := req.VolumeCapability
	if vc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	md, err := vol.GetMetadata()
	RegisterVolumeCapability(ctxt, md, vc)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	for _, bm := range strings.Split((*md)["bind-mount"], ",") {
		vol.BindMountPaths.Add(bm)
	}
	if err = vol.BindMount(req.TargetPath); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	(*md)["bind-mount"] = strings.Join(vol.BindMountPaths.List(), ",")
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	err = vol.UnBindMount(req.TargetPath)
	if err != nil {
		co.Warning(ctxt, err)
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	d.InitFunc(ctx, "node", "NodeUnpublishVolume", *req)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	vol, err := d.dc.GetVolume(vid, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	for _, bm := range strings.Split((*md)["bind-mount"], ",") {
		vol.BindMountPaths.Add(bm)
	}
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
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

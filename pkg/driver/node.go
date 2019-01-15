package driver

import (
	"context"
	"fmt"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	units "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
)

func (d *Driver) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeStageVolume", *req)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "StagingTargetPath cannot be empty")
	}
	vc := req.VolumeCapability
	if vc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	vol, err := d.dc.GetVolume(vid, false, true)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	RegisterVolumeCapability(ctxt, md, vc)
	// Setup ACL
	init, err := d.dc.CreateGetInitiator()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	if err = vol.RegisterAcl(init); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}

	// Setup IpPool
	if ipp, err := d.dc.GetIpPoolFromName((*md)["ip_pool"]); err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	} else {
		if err = vol.RegisterIpPool(ipp); err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
	}
	// Login to target
	if err = vol.Login(!d.env.DisableMultipath, (*md)["round_robin"] == "true"); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	(*md)["device_path"] = vol.DevicePath
	switch vc.GetAccessType().(type) {

	case *csi.VolumeCapability_Mount:
		co.Infof(ctxt, "Handling NodeStageVolume VolumeCapability_Mount")
		if fs := (*md)["fs_type"]; fs != "" {
			// Mount Device
			fsType, fsArgs := fs, strings.Split((*md)["fs_args"], " ")
			err = vol.Format(fsType, fsArgs, d.env.FormatTimeout)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, err.Error())
			}
			vol.Formatted = true
			(*md)["formatted"] = "true"
			mountArgs := strings.Split((*md)["m_args"], " ")
			err = vol.Mount(req.StagingTargetPath, mountArgs)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, err.Error())
			}
			(*md)["mount_path"] = vol.MountPath
		}
	case *csi.VolumeCapability_Block:
		// No formatting is needed since this is raw block
		co.Infof(ctxt, "Handling NodeStageVolume VolumeCapability_Block")
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Unknown volume capability: %#v", vc))
	}
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeUnstageVolume", *req)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "StagingTargetPath cannot be empty")
	}
	vol, err := d.dc.GetVolume(vid, false, true)
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
	md, err := vol.GetMetadata()
	if err != nil {
		co.Warning(ctxt, err)
	}
	if md == nil {
		md = &dc.VolMetadata{}
	}
	(*md)["mount_path"] = ""
	if _, err = vol.SetMetadata(md); err != nil {
		co.Warning(ctxt, err)
	}
	err = vol.Logout()
	if err != nil {
		co.Warning(ctxt, err)
	}
	if (*md)["delete_on_unmount"] == "true" {
		co.Infof(ctxt, "Auto-deleting %s on unmount", vol.Name)
		if err = vol.Delete(false); err != nil {
			co.Warning(ctxt, err)
		}
	}
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodePublishVolume", *req)
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
	vol, err := d.dc.GetVolume(vid, false, true)
	vc := req.VolumeCapability
	if vc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	RegisterVolumeCapability(ctxt, md, vc)
	for _, bm := range strings.Split((*md)["bind_mount"], ",") {
		vol.BindMountPaths.Add(bm)
	}
	if err = vol.BindMount(req.TargetPath); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	(*md)["bind_mount"] = strings.Join(vol.BindMountPaths.List(), ",")
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "node", "NodeUnpublishVolume", *req)
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.TargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "TargetPath cannot be empty")
	}
	vol, err := d.dc.GetVolume(vid, false, true)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	for _, bm := range strings.Split((*md)["bind_mount"], ",") {
		vol.BindMountPaths.Delete(bm)
	}
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	err = vol.UnBindMount(req.TargetPath)
	if err != nil {
		co.Warning(ctxt, err)
	}
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	d.InitFunc(ctx, "node", "NodeGetVolumeStats", *req)
	resp := &csi.NodeGetCapabilitiesResponse{Capabilities: []*csi.NodeServiceCapability{}}
	addCap := func(t csi.NodeServiceCapability_RPC_Type) {
		resp.Capabilities = append(resp.Capabilities, &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: t,
				},
			},
		})
	}
	for _, t := range []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
		csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
	} {
		addCap(t)
	}
	return resp, nil
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
	v, err := d.dc.GetVolume(req.VolumeId, false, false)
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

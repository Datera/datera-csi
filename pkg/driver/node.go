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

	chapParams := map[string]string{}
	chapParams = co.StripSecretsAndGetChapParams(req)

	ctxt, ip, clean := d.InitFunc(ctx, "node", "NodeStageVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	vid := req.VolumeId
	if vid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.StagingTargetPath == "" {
		return nil, status.Errorf(codes.InvalidArgument, "StagingTargetPath cannot be empty. Kubelet on the worker node is responsible for creating this StagingTargetPath directory.")
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
	if err := RegisterVolumeCapability(ctxt, md, vc); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	// Setup ACL
	init, err := d.dc.CreateGetInitiator()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	if err = vol.RegisterAcl(init); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}

	// This has been moved to volume creation time to satisfy silly requirements
	// // Setup IpPool
	// if vol.Template == "" {
	// 	co.Debugf(ctxt, "Registering IP Pool: %s", (*md)["ip_pool"])
	// 	if ipp, err := d.dc.GetIpPoolFromName((*md)["ip_pool"]); err != nil {
	// 		return nil, status.Errorf(codes.NotFound, err.Error())
	// 	} else {
	// 		if err = vol.RegisterIpPool(ipp); err != nil {
	// 			return nil, status.Errorf(codes.Unknown, err.Error())
	// 		}
	// 	}
	// } else {
	// 	co.Debug(ctxt, "Skipping IP Pool registration due to Template")
	// }
	// Online AI (to ensure targets are accessible)
	if err = vol.Online(); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	// Login to target
	if err = vol.Login(!d.env.DisableMultipath, (*md)["round_robin"] == "true", chapParams); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	(*md)["device_path"] = vol.DevicePath
	switch vc.GetAccessType().(type) {

	case *csi.VolumeCapability_Mount:
		co.Infof(ctxt, "Handling NodeStageVolume VolumeCapability_Mount")
		fsType := (*md)["fs_type"]
		// Mount Device
		if fsType == "" {
			fsType = co.Ext4
		}
		fsArgs := strings.Split((*md)["fs_args"], " ")
		if len(fsArgs) == 0 {
			fsArgs = DefaultFsArgs[fsType]
		}
		if !vol.Formatted && (*md)["formatted"] != "true" {
			err = vol.Format(fsType, fsArgs, d.env.FormatTimeout)
			if err != nil {
				return nil, status.Errorf(codes.Unknown, err.Error())
			}
			vol.Formatted = true
			(*md)["formatted"] = "true"
		}
		mountArgs := strings.Split((*md)["m_args"], " ")
		err = vol.Mount(req.StagingTargetPath, mountArgs, fsType)
		if err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
		(*md)["mount_path"] = vol.MountPath
	case *csi.VolumeCapability_Block:
		// No formatting is needed since this is raw block
		co.Infof(ctxt, "Handling NodeStageVolume VolumeCapability_Block")
		(*md)["mount_path"] = vol.DevicePath
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Unknown volume capability: %#v", vc))
	}
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "node", "NodeUnstageVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
	init, err := d.dc.CreateGetInitiator()
	if err != nil {
		co.Warning(ctxt, err)
	}
	err = vol.UnregisterAcl(init)
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
	ctxt, ip, clean := d.InitFunc(ctx, "node", "NodePublishVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
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
	vol, err := d.dc.GetVolume(vid, false, true)
	vc := req.VolumeCapability
	if vc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	md, err := vol.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	if err := RegisterVolumeCapability(ctxt, md, vc); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	for _, bm := range strings.Split((*md)["bind_mount"], ",") {
		vol.BindMountPaths.Add(bm)
	}
	fsType := (*md)["fs_type"]
	if err = vol.BindMount(req.TargetPath, fsType); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	(*md)["bind_mount"] = strings.Join(vol.BindMountPaths.List(), ",")
        // Add Pod level details into App Instance metadata, if they exist in this call
        if  req.VolumeContext != nil {
                podName := req.VolumeContext["csi.storage.k8s.io/pod.name"]
                podNamespace := req.VolumeContext["csi.storage.k8s.io/pod.namespace"]
                podUid := req.VolumeContext["csi.storage.k8s.io/pod.uid"]
                svcAcct := req.VolumeContext["csi.storage.k8s.io/serviceAccount.name"]
                if podName != "" {
                        (*md)["pod_name"] = podName
                }
                if podNamespace != "" {
                        (*md)["pod_namespace"] = podNamespace
                }
                if podUid != "" {
                        (*md)["pod_uid"] = podUid
                }
                if svcAcct != "" {
                        (*md)["k8s_service_account"] = svcAcct
                }
        }
	if _, err = vol.SetMetadata(md); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "node", "NodeUnpublishVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
	// Remove Pod level details from the Volume Metadata
        (*md)["pod_name"] = ""
        (*md)["pod_namespace"] = ""
        (*md)["pod_uid"] = ""
        (*md)["k8s_service_account"] = ""

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
	_, ip, clean := d.InitFunc(ctx, "node", "NodeGetVolumeStats", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
                csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	} {
		addCap(t)
	}
	return resp, nil
}

func (d *Driver) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "node", "NodeGetInfo", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	log.WithField("method", "node_get_info").Infof("Node server %s 'NodeGetInfo' called", d.nid)
	return &csi.NodeGetInfoResponse{
		NodeId:             d.nid,
		MaxVolumesPerNode:  int64(d.env.VolPerNode),
		AccessibleTopology: nil,
	}, nil
}

func (d *Driver) NodeGetVolumeStats(ctx context.Context, req *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "node", "NodeGetVolumeStats", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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

func (d *Driver) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "node", "NodeExpandVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	v, err := d.dc.GetVolume(req.VolumeId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	md, err := v.GetMetadata()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	cr := req.CapacityRange
	if cr != nil && cr.LimitBytes == 0 {
		cr.LimitBytes = cr.RequiredBytes
	}
	size := int(cr.RequiredBytes / units.GiB)
	if err := v.ExpandFs(req.VolumePath, (*md)["fs_type"], int64(size)); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, err.Error())
	}
	resp := &csi.NodeExpandVolumeResponse{
		CapacityBytes: req.GetCapacityRange().RequiredBytes,
	}
	return resp, nil
}

package driver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	codes "google.golang.org/grpc/codes"
	gmd "google.golang.org/grpc/metadata"
	status "google.golang.org/grpc/status"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
)

const (
	DefaultSize = 16
)

func parseVolParams(ctxt context.Context, params map[string]string) (*dc.VolOpts, error) {
	//Golang makes something that should be simple, repetative and gross
	dparams := make(map[string]string, 13)
	vo := &dc.VolOpts{}
	var err error
	for k := range params {
		if strings.HasPrefix(k, "DF:") {
			nk := strings.Replace(k, "DF:", "", 1)
			dparams[nk] = params[k]
		}
	}
	co.Debugf(ctxt, "Filtered Params: %s", dparams)
	// set defaults
	if _, ok := dparams["iops_per_gb"]; !ok {
		dparams["iops_per_gb"] = "0"
	}
	if _, ok := dparams["bandwidth_per_gb"]; !ok {
		dparams["bandwidth_per_gb"] = "0"
	}
	if _, ok := dparams["placement_mode"]; !ok {
		dparams["placement_mode"] = "hybrid"
	}
	if _, ok := dparams["round_robin"]; !ok {
		dparams["round_robin"] = "false"
	}
	if _, ok := dparams["replica_count"]; !ok {
		dparams["replica_count"] = "3"
	}
	if _, ok := dparams["ip_pool"]; !ok {
		dparams["ip_pool"] = "default"
	}
	if _, ok := dparams["template"]; !ok {
		dparams["template"] = ""
	}
	if _, ok := dparams["read_iops_max"]; !ok {
		dparams["read_iops_max"] = "0"
	}
	if _, ok := dparams["write_iops_max"]; !ok {
		dparams["write_iops_max"] = "0"
	}
	if _, ok := dparams["total_iops_max"]; !ok {
		dparams["total_iops_max"] = "0"
	}
	if _, ok := dparams["read_bandwidth_max"]; !ok {
		dparams["read_bandwidth_max"] = "0"
	}
	if _, ok := dparams["write_bandwidth_max"]; !ok {
		dparams["write_bandwidth_max"] = "0"
	}
	if _, ok := dparams["total_bandwidth_max"]; !ok {
		dparams["total_bandwidth_max"] = "0"
	}
	if _, ok := dparams["fs_type"]; !ok {
		dparams["fs_type"] = "ext4"
	}
	if _, ok := dparams["fs_args"]; !ok {
		dparams["fs_args"] = "-E lazy_itable_init=0,lazy_journal_init=0,nodiscard -F"
	}

	val, err := strconv.ParseInt(dparams["iops_per_gb"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.IopsPerGb = int(val)
	val, err = strconv.ParseInt(dparams["bandwidth_per_gb"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.BandwidthPerGb = int(val)
	vo.PlacementMode = dparams["placement_mode"]
	b, err := strconv.ParseBool(dparams["round_robin"])
	if err != nil {
		return nil, err
	}
	vo.RoundRobin = b
	val, err = strconv.ParseInt(dparams["replica_count"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.Replica = int(val)
	vo.IpPool = dparams["ip_pool"]
	vo.Template = dparams["template"]
	val, err = strconv.ParseInt(dparams["read_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.ReadIopsMax = int(val)
	val, err = strconv.ParseInt(dparams["write_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.WriteIopsMax = int(val)
	val, err = strconv.ParseInt(dparams["total_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.TotalIopsMax = int(val)
	val, err = strconv.ParseInt(dparams["read_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.ReadBandwidthMax = int(val)
	val, err = strconv.ParseInt(dparams["write_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.WriteBandwidthMax = int(val)
	val, err = strconv.ParseInt(dparams["total_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.TotalBandwidthMax = int(val)
	vo.FsType = dparams["fs_type"]
	vo.FsArgs = strings.Split(dparams["fs_args"], " ")
	return vo, nil
}

func validateSnapId(snapId string) error {
	const example = "/app_instances/my-app/storage_instances/storage-1/volumes/volume-1/snapshots/1536262088.285952448"
	parts := strings.Split(strings.TrimLeft(snapId, "/"), "/")
	if len(parts) != 8 {
		return fmt.Errorf("Snapshot ID invalid.  Example: %s", example)
	}
	return nil
}

func handleTopologyRequirement(tr *csi.TopologyRequirement) error {
	if tr == nil {
		return nil
	}
	if len(tr.Requisite) > 0 || len(tr.Preferred) > 0 {
		return fmt.Errorf("TopologyRequirements and Preferred Topologies are currently unsupported")
	}
	return nil
}

func registerMdFromCtxt(ctxt context.Context, md *dc.VolMetadata) error {
	gmdata, ok := gmd.FromIncomingContext(ctxt)
	co.Debugf(ctxt, "Recieved Metadata: %s", gmdata)
	if !ok {
		return fmt.Errorf("Error retrieving metadata from RPC")
	}
	for k, v := range gmdata {
		(*md)[k] = strings.Join(v, ",")
	}
	return nil
}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "CreateVolume", *req)
	// Handle req.Name
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Name must be provided (currently empty string)")
	}
	id := co.GenName(req.Name)

	cr := req.CapacityRange
	if cr != nil && cr.LimitBytes == 0 {
		cr.LimitBytes = cr.RequiredBytes
	}

	// Check to see if a volume already exists with this name
	if vol, err := d.dc.GetVolume(id, false); err == nil {
		size := int64(vol.Size * units.GiB)
		if cr != nil && (cr.LimitBytes < size || cr.RequiredBytes != size) {
			return nil, status.Errorf(codes.InvalidArgument, "Requested volume exists, but has a different size")
		}
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes: size,
				Id:            vol.Name,
				Attributes:    map[string]string{},
			},
		}, nil
	}

	// Handle req.AccessibilityRequirements.  Currently we just error out if a topology requirement exists
	// TODO: Digest this beast: https://github.com/container-storage-interface/spec/blob/master/lib/go/csi/v0/csi.pb.go#L1431
	if err := handleTopologyRequirement(req.AccessibilityRequirements); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	md := &dc.VolMetadata{}
	(*md)["display-name"] = req.Name
	registerMdFromCtxt(ctxt, md)

	vcs := req.VolumeCapabilities
	if vcs == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapabilities cannot be empty")
	}
	for _, vc := range vcs {
		RegisterVolumeCapability(ctxt, md, vc)
	}
	co.Debugf(ctxt, "Metadata after registering VolumeCapabilities: %#v", *md)
	// Handle req.Parameters
	params, err := parseVolParams(ctxt, req.Parameters)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Needed for testing on single-node systems
	if d.env.ReplicaOverride {
		params.Replica = 1
	}

	// Handle req.VolumeContentSource
	cs := req.VolumeContentSource
	if snap := cs.GetSnapshot(); snap != nil {
		if err = validateSnapId(snap.Id); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		params.CloneSnapSrc = snap.Id
	}

	// Handle req.CapacityRange
	if cr != nil && cr.RequiredBytes > cr.LimitBytes {
		return &csi.CreateVolumeResponse{}, fmt.Errorf("RequiredBytes must be less than or equal to LimitBytes: [%d, %d]", cr.RequiredBytes, cr.LimitBytes)
	}
	var size int
	if cr != nil {
		// Default to LimitBytes since we've verified that it's larger than RequiredBytes
		size = int(cr.LimitBytes / units.GiB)
		// If we haven't been passed any capacity, default to 16 GiB
		if size == 0 {
			size = DefaultSize
		} else if size < 1 {
			size = 1
		}
	} else {
		size = DefaultSize
	}
	params.Size = size
	// Create AppInstance/StorageInstance/Volume
	vol, err := d.dc.CreateVolume(id, params, true)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}

	// Handle req.ControllerCreateSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// handleVolSecrets(req.ControllerCreateSecrets)

	//Set metadata, fail gracefully
	if md, err = vol.SetMetadata(md); err != nil {
		co.Error(ctxt, err)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: int64(size * units.GiB),
			Id:            vol.Name,
			Attributes:    map[string]string{},
			ContentSource: nil,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "DeleteVolume", *req)
	vid := req.VolumeId
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	// Handle req.ControllerDeleteSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// sec := req.ControllerDeleteSecrets
	if err := d.dc.DeleteVolume(req.VolumeId, true); err != nil {
		co.Errorf(ctxt, "Error deleting volume: %s.  err: %s", vid, err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	d.InitFunc(ctx, "controller", "ControllerPublishVolume", *req)
	return nil, status.Errorf(codes.Unimplemented, "ControllerPublishVolume Not Implemented")
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.VolumeCapability == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapability cannot be nil")
	}
	if req.NodeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "NodeId cannot be empty")
	}
	am := req.VolumeCapability.GetAccessMode()
	if am != nil {
		mo := am.Mode.String()
		if strings.Contains(mo, "WRITER") && req.Readonly {
			return nil, status.Errorf(codes.AlreadyExists, fmt.Sprintf("Volume cannot be publshed as ReadOnly with AccessMode %s simultaneously", mo))
		}
	}
	_, err := d.dc.GetVolume(req.VolumeId, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	h, err := os.Hostname()
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &csi.ControllerPublishVolumeResponse{
		PublishInfo: map[string]string{
			"controller_host": h,
		},
	}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	d.InitFunc(ctx, "controller", "ControllerUnpublishVolume", *req)
	return nil, status.Errorf(codes.Unimplemented, "ControllerUnPublishVolume Not Implemented")
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.VolumeCapabilities == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapabilities cannot be nil")
	}
	if _, err := d.dc.GetVolume(req.VolumeId, false); err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Supported: true,
		Message:   "",
	}, nil
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "ListVolumes", *req)
	var err error
	st := int64(0)
	if req.StartingToken != "" {
		st, err = strconv.ParseInt(req.StartingToken, 0, 0)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	vols, err := d.dc.ListVolumes(int(req.MaxEntries), int(st))
	if err != nil {
		co.Error(ctxt, err)
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	rvols := []*csi.ListVolumesResponse_Entry{}
	for _, vol := range vols {
		rvols = append(rvols, &csi.ListVolumesResponse_Entry{
			Volume: &csi.Volume{
				CapacityBytes: int64(vol.Size * units.GiB),
				Id:            vol.Name,
				Attributes:    map[string]string{},
				ContentSource: nil,
			},
		})
	}
	return &csi.ListVolumesResponse{
		Entries: rvols,
	}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "GetCapacity", *req)
	params, err := parseVolParams(ctxt, req.Parameters)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	cap, err := d.dc.GetCapacity()
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	acap := int64(cap.Total)
	if params.PlacementMode == "all_flash" {
		acap = int64(cap.FlashTotal)
	}
	acap = int64(acap / int64(params.Replica))
	return &csi.GetCapacityResponse{
		AvailableCapacity: acap,
	}, nil
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	d.InitFunc(ctx, "controller", "ControllerGetCapabilities", *req)
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
					},
				},
			},
			// &csi.ControllerServiceCapability{
			// 	Type: &csi.ControllerServiceCapability_Rpc{
			// 		Rpc: &csi.ControllerServiceCapability_RPC{
			// 			Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			// 		},
			// 	},
			// },
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_GET_CAPACITY,
					},
				},
			},
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
					},
				},
			},
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	d.InitFunc(ctx, "controller", "CreateSnapshot", *req)
	if req.SourceVolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "SourceVolumeId cannot be empty")
	}
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Name field cannot be empty")
	}
	vol, err := d.dc.GetVolume(req.SourceVolumeId, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	snap, err := vol.CreateSnapshot(req.Name)
	if err != nil && strings.Contains(err.Error(), "use a different UUID") {
		return nil, status.Errorf(codes.AlreadyExists, err.Error())
	} else if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	ts, err := strconv.ParseFloat(snap.Id, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	//TODO Implement snapshot polling before returning
	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			// We set the id to "<volume-id>:<snapshot-id>" since during delete requests
			// we are not given the parent volume id
			Id:             co.MkSnapId(vol.Name, snap.Id),
			SourceVolumeId: vol.Name,
			SizeBytes:      int64(vol.Size * units.GiB),
			CreatedAt:      int64(ts),
			Status: &csi.SnapshotStatus{
				Type:    csi.SnapshotStatus_READY,
				Details: snap.Status,
			},
		},
	}, nil
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "DeleteSnapshot", *req)
	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "SnapshotId is invalid (empty string)")
	}
	vid, sid := co.ParseSnapId(req.SnapshotId)
	if vid == "" || sid == "" {
		co.Warningf(ctxt, "SnapshotId is invalid (Not of the form app_instance_id:snapshot_id): %s", req.SnapshotId)
		return &csi.DeleteSnapshotResponse{}, nil
	}
	vol, err := d.dc.GetVolume(vid, false)
	if err != nil {
		co.Warningf(ctxt, "VolumeId is invalid: %s", vid)
		return &csi.DeleteSnapshotResponse{}, nil
	}
	if err = vol.DeleteSnapshot(sid); err != nil {
		co.Warning(ctxt, err)
		return &csi.DeleteSnapshotResponse{}, nil
	}
	return &csi.DeleteSnapshotResponse{}, nil
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	ctxt := d.InitFunc(ctx, "controller", "ListSnapshots", *req)
	rsnaps := []*csi.ListSnapshotsResponse_Entry{}
	var err error
	st := int64(0)
	if req.StartingToken != "" {
		st, err = strconv.ParseInt(req.StartingToken, 0, 0)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	snaps, err := d.dc.ListSnapshots(req.SnapshotId, req.SourceVolumeId, int(req.MaxEntries), int(st))
	if err != nil && req.SourceVolumeId != "" && strings.Contains(err.Error(), "NotFound") {
		return &csi.ListSnapshotsResponse{
			Entries: []*csi.ListSnapshotsResponse_Entry{},
		}, nil
	} else if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	co.Debugf(ctxt, "Recieved snapshots: %#v", snaps)
	for _, snap := range snaps {
		ts, err := strconv.ParseFloat(snap.Id, 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		rsnaps = append(rsnaps, &csi.ListSnapshotsResponse_Entry{
			Snapshot: &csi.Snapshot{
				Id:             co.MkSnapId(snap.Vol.Name, snap.Id),
				SizeBytes:      int64(snap.Vol.Size * units.GiB),
				SourceVolumeId: snap.Vol.Name,
				CreatedAt:      int64(ts),
			},
		})
	}
	co.Debugf(ctxt, "Returning snapshots: %#v", rsnaps)
	return &csi.ListSnapshotsResponse{
		Entries: rsnaps,
	}, nil
}

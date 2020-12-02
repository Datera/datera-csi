package driver

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	units "github.com/docker/go-units"
	ptypes "github.com/golang/protobuf/ptypes"
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
	if params == nil {
		params = make(map[string]string, 16)
	}
	vo := &dc.VolOpts{}
	var err error
	co.Debugf(ctxt, "Volume Params: %s", params)
	// set defaults
	if _, ok := params["iops_per_gb"]; !ok {
		params["iops_per_gb"] = "0"
	}
	if _, ok := params["bandwidth_per_gb"]; !ok {
		params["bandwidth_per_gb"] = "0"
	}
	if _, ok := params["placement_mode"]; !ok {
		params["placement_mode"] = "hybrid"
	}
	if _, ok := params["placement_policy"]; !ok {
		params["placement_policy"] = "default"
	}
	if _, ok := params["round_robin"]; !ok {
		params["round_robin"] = "false"
	}
	if _, ok := params["replica_count"]; !ok {
		params["replica_count"] = "3"
	}
	if _, ok := params["ip_pool"]; !ok {
		params["ip_pool"] = "default"
	}
	if _, ok := params["template"]; !ok {
		params["template"] = ""
	}
	if _, ok := params["disable_template_override"]; !ok {
		params["disable_template_override"] = "false"
	}
	if _, ok := params["read_iops_max"]; !ok {
		params["read_iops_max"] = "0"
	}
	if _, ok := params["write_iops_max"]; !ok {
		params["write_iops_max"] = "0"
	}
	if _, ok := params["total_iops_max"]; !ok {
		params["total_iops_max"] = "0"
	}
	if _, ok := params["read_bandwidth_max"]; !ok {
		params["read_bandwidth_max"] = "0"
	}
	if _, ok := params["write_bandwidth_max"]; !ok {
		params["write_bandwidth_max"] = "0"
	}
	if _, ok := params["total_bandwidth_max"]; !ok {
		params["total_bandwidth_max"] = "0"
	}
	if _, ok := params["delete_on_unmount"]; !ok {
		params["delete_on_unmount"] = "false"
	}

	val, err := strconv.ParseInt(params["iops_per_gb"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.IopsPerGb = int(val)
	val, err = strconv.ParseInt(params["bandwidth_per_gb"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.BandwidthPerGb = int(val)
	vo.PlacementMode = params["placement_mode"]
        vo.PlacementPolicy = params["placement_policy"]
	b, err := strconv.ParseBool(params["round_robin"])
	if err != nil {
		return nil, err
	}
	vo.RoundRobin = b
	val, err = strconv.ParseInt(params["replica_count"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.Replica = int(val)
	vo.IpPool = params["ip_pool"]
	vo.Template = params["template"]
	b, err = strconv.ParseBool(params["disable_template_override"])
	if err != nil {
		return nil, err
	}
	vo.DisableTemplateOverride = b
	val, err = strconv.ParseInt(params["read_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.ReadIopsMax = int(val)
	val, err = strconv.ParseInt(params["write_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.WriteIopsMax = int(val)
	val, err = strconv.ParseInt(params["total_iops_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.TotalIopsMax = int(val)
	val, err = strconv.ParseInt(params["read_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.ReadBandwidthMax = int(val)
	val, err = strconv.ParseInt(params["write_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.WriteBandwidthMax = int(val)
	val, err = strconv.ParseInt(params["total_bandwidth_max"], 10, 0)
	if err != nil {
		return nil, err
	}
	vo.TotalBandwidthMax = int(val)
	b, err = strconv.ParseBool(params["delete_on_unmount"])
	if err != nil {
		return nil, err
	}
	vo.DeleteOnUnmount = b
	return vo, nil
}

func parseSnapParams(ctxt context.Context, params map[string]string) (*dc.SnapOpts, error) {
	//Golang makes something that should be simple, repetative and gross
	if params == nil {
		params = make(map[string]string, 16)
	}
	so := &dc.SnapOpts{}
	// var err error
	co.Debugf(ctxt, "Snapshot Params: %s", params)
	// set defaults
	if _, ok := params["remote_provider_uuid"]; !ok {
		params["remote_provider_uuid"] = ""
	}
	if _, ok := params["type"]; !ok {
		params["type"] = "local"
	}
	so.RemoteProviderUuid = params["remote_provider_uuid"]
	so.Type = params["type"]
	return so, nil
}

func validateSnapId(snapId string) error {
	const example = "CSI-pvc-2071cca0-3259-11e9-aba5-003048f5d94a:1550370547.151396819"
	parts := strings.Split(snapId, ":")
	if len(parts) != 2 {
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

	chapParams := map[string]string{}
	chapParams = co.StripSecretsAndGetChapParams(req)

	ctxt, ip, clean := d.InitFunc(ctx, "controller", "CreateVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
	if vol, err := d.dc.GetVolume(id, false, false); err == nil {
		size := int64(vol.Size * units.GiB)
		if cr != nil && (cr.LimitBytes < size || cr.RequiredBytes != size) {
			return nil, status.Errorf(codes.AlreadyExists, "Requested volume exists, but has a different size")
		}
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes: size,
				VolumeId:      vol.Name,
				VolumeContext: map[string]string{},
			},
		}, nil
	}

	// Handle req.AccessibilityRequirements.  Currently we just error out if a topology requirement exists
	// TODO: Implement real topology handling.
	if err := handleTopologyRequirement(req.AccessibilityRequirements); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	md := &dc.VolMetadata{}
	// Limit name size so we don't overflow metadata
	if len(req.Name) > 100 {
		req.Name = req.Name[:100]
		co.Warningf(ctxt, "Limiting display-name to 100 characters: %s", req.Name)
	}
	(*md)["display_name"] = req.Name
	registerMdFromCtxt(ctxt, md)

	vcs := req.VolumeCapabilities
	if vcs == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapabilities cannot be empty")
	}
	for _, vc := range vcs {
		if err := RegisterVolumeCapability(ctxt, md, vc); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	co.Debugf(ctxt, "Metadata after registering VolumeCapabilities: %#v", *md)
	// Handle req.Parameters
	params, err := parseVolParams(ctxt, req.Parameters)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Add parameters to metadata for storage
	for k, v := range params.ToMap() {
		(*md)[k] = v
	}

	// Needed for testing on single-node systems
	if d.env.ReplicaOverride {
		params.Replica = 1
	}

	// Handle req.VolumeContentSource
	cs := req.VolumeContentSource
	if snap := cs.GetSnapshot(); snap != nil {
		if err = validateSnapId(snap.SnapshotId); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		src, err := d.dc.SnapshotPathFromCsiId(snap.SnapshotId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		params.CloneSnapSrc = src
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
	// Fix for CET-312. QoS params sent along with volume creation call
	// No need to update the performance_policy again
	// Fix for CET-491. CHAP params are obtained from K8S 
	// and sent to Datera backend for Auth configuration
	// Get the CHAP params passed from Kubernetes StorageClass
	// Strip the credentials and get it as chapParams

	vol, err := d.dc.CreateVolume(id, params, false, chapParams)
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
			VolumeId:      vol.Name,
			VolumeContext: map[string]string{},
			ContentSource: nil,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {

	// Just strip the secrets from the GRPC request.
	// Discard the returned chapParams since DeleteVolume doesn't need them.
	_ = co.StripSecretsAndGetChapParams(req)
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "DeleteVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	vid := req.VolumeId
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	// Handle req.ControllerDeleteSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// sec := req.ControllerDeleteSecrets
	if err := d.dc.DeleteVolume(req.VolumeId, true); err != nil {
		co.Errorf(ctxt, "Error deleting volume: %s.  err: %s", vid, err)
		if strings.Contains(err.Error(), "it has snapshots") {
			return nil, status.Errorf(codes.FailedPrecondition, "Volumes with snapshots cannot be deleted.  Delete snapshots first")
		}
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "controller", "ControllerPublishVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	return nil, status.Errorf(codes.Unimplemented, "ControllerPublishVolume Not Implemented")
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "controller", "ControllerUnpublishVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	return nil, status.Errorf(codes.Unimplemented, "ControllerUnPublishVolume Not Implemented")
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	_, ip, clean := d.InitFunc(ctx, "controller", "ValidateVolumeCapabilities", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	if req.VolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeId cannot be empty")
	}
	if req.VolumeCapabilities == nil {
		return nil, status.Errorf(codes.InvalidArgument, "VolumeCapabilities cannot be nil")
	}
	if _, err := d.dc.GetVolume(req.VolumeId, false, false); err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	return &csi.ValidateVolumeCapabilitiesResponse{
		Confirmed: &csi.ValidateVolumeCapabilitiesResponse_Confirmed{
			VolumeContext: map[string]string{},
			VolumeCapabilities: []*csi.VolumeCapability{
				{
					AccessType: &csi.VolumeCapability_Mount{
						Mount: &csi.VolumeCapability_MountVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
				{
					AccessType: &csi.VolumeCapability_Block{
						Block: &csi.VolumeCapability_BlockVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
					},
				},
				{
					AccessType: &csi.VolumeCapability_Block{
						Block: &csi.VolumeCapability_BlockVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
					},
				},
				{
					AccessType: &csi.VolumeCapability_Block{
						Block: &csi.VolumeCapability_BlockVolume{},
					},
					AccessMode: &csi.VolumeCapability_AccessMode{
						Mode: csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
					},
				},
			},
		},
		Message: "",
	}, nil
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "ListVolumes", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
				VolumeId:      vol.Name,
				VolumeContext: map[string]string{},
				ContentSource: nil,
			},
		})
	}
	return &csi.ListVolumesResponse{
		Entries: rvols,
	}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "GetCapacity", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
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
	_, ip, clean := d.InitFunc(ctx, "controller", "ControllerGetCapabilities", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	resp := &csi.ControllerGetCapabilitiesResponse{Capabilities: []*csi.ControllerServiceCapability{}}
	addCap := func(t csi.ControllerServiceCapability_RPC_Type) {
		resp.Capabilities = append(resp.Capabilities, &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: t,
				},
			},
		})
	}
	for _, t := range []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_GET_CAPACITY,
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
		csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
		csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	} {
		addCap(t)
	}
	return resp, nil
}

func (d *Driver) CreateSnapshot(ctx context.Context, req *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "CreateSnapshot", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	if req.SourceVolumeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "SourceVolumeId cannot be empty")
	}
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Name field cannot be empty")
	}
	vol, err := d.dc.GetVolume(req.SourceVolumeId, false, false)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, err.Error())
	}
	params, err := parseSnapParams(ctxt, req.Parameters)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	snap, err := vol.CreateSnapshot(req.Name, params)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	ts, err := strconv.ParseFloat(snap.Id, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sec, dec := math.Modf(ts)
	pts, err := ptypes.TimestampProto(time.Unix(int64(sec), int64(dec)))
	if err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	//TODO Implement snapshot polling before returning
	return &csi.CreateSnapshotResponse{
		Snapshot: &csi.Snapshot{
			// We set the id to "<volume-id>:<snapshot-id>" since during delete requests
			// we are not given the parent volume id
			SnapshotId:     co.MkSnapId(vol.Name, snap.Id),
			SourceVolumeId: vol.Name,
			SizeBytes:      int64(vol.Size * units.GiB),
			CreationTime:   pts,
			ReadyToUse:     true,
		},
	}, nil
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "DeleteSnapshot", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "SnapshotId is invalid (empty string)")
	}
	vid, sid := co.ParseSnapId(req.SnapshotId)
	if vid == "" || sid == "" {
		co.Warningf(ctxt, "SnapshotId is invalid (Not of the form app_instance_id:snapshot_id): %s", req.SnapshotId)
		return &csi.DeleteSnapshotResponse{}, nil
	}
	vol, err := d.dc.GetVolume(vid, false, false)
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
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "ListSnapshots", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	rsnaps := []*csi.ListSnapshotsResponse_Entry{}
	var err error
	st := int64(0)
	if req.StartingToken != "" {
		st, err = strconv.ParseInt(req.StartingToken, 0, 0)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
	}
	snaps, nextToken, err := d.dc.ListSnapshots(req.SnapshotId, req.SourceVolumeId, int(req.MaxEntries), int(st))
	if err != nil && req.SourceVolumeId != "" && strings.Contains(err.Error(), "NotFound") {
		return &csi.ListSnapshotsResponse{
			Entries: []*csi.ListSnapshotsResponse_Entry{},
		}, nil
	} else if err != nil && strings.Contains(err.Error(), "must be of format") {
		return &csi.ListSnapshotsResponse{
			Entries: rsnaps,
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
		sec, dec := math.Modf(ts)
		pts, err := ptypes.TimestampProto(time.Unix(int64(sec), int64(dec)))
		if err != nil {
			return nil, status.Errorf(codes.Unknown, err.Error())
		}
		rsnaps = append(rsnaps, &csi.ListSnapshotsResponse_Entry{
			Snapshot: &csi.Snapshot{
				SnapshotId:     co.MkSnapId(snap.Vol.Name, snap.Id),
				SizeBytes:      int64(snap.Vol.Size * units.GiB),
				SourceVolumeId: snap.Vol.Name,
				CreationTime:   pts,
			},
		})
	}
	nt := ""
	if nextToken != 0 {
		nt = strconv.FormatInt(int64(nextToken), 10)
	}
	co.Debugf(ctxt, "Returning snapshots: %#v", rsnaps)
	return &csi.ListSnapshotsResponse{
		Entries:   rsnaps,
		NextToken: nt,
	}, nil
}

func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	ctxt, ip, clean := d.InitFunc(ctx, "controller", "ControllerExpandVolume", *req)
	defer clean()
	if ip {
		return nil, status.Errorf(codes.Aborted, "Operation is still in progress")
	}
	cr := req.CapacityRange
	if cr != nil && cr.LimitBytes == 0 {
		cr.LimitBytes = cr.RequiredBytes
	}
	vol, err := d.dc.GetVolume(req.VolumeId, false, false)
	if err != nil {
		co.Warningf(ctxt, "VolumeId is invalid: %s", req.VolumeId)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := vol.Resize(int(cr.RequiredBytes / units.GiB)); err != nil {
		return nil, status.Errorf(codes.Unknown, err.Error())
	}
	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         cr.RequiredBytes,
		NodeExpansionRequired: true,
	}, nil
}

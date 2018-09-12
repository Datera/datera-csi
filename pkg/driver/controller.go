package driver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	gmd "google.golang.org/grpc/metadata"

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

func registerVolumeCapability(ctxt context.Context, md *dc.VolMetadata, vc *csi.VolumeCapability) {
	// Record req.VolumeCapabilities in metadata We don't actually do anything
	// with this information because it's all the same to us, but we should
	// keep it for future product filtering/aggregate operations
	var (
		at string
		fs string
	)
	mo := string(vc.GetAccessMode().Mode)
	switch vc.GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		at = "block"
	case *csi.VolumeCapability_Mount:
		at = "mount"
		fs = vc.GetMount().FsType + " " + strings.Join(vc.GetMount().MountFlags, "")
		co.Debugf(ctxt, "Registering Filesystem %s", fs)
	default:
		at = "unknown"
	}
	co.Debugf(ctxt, "Registering VolumeCapability %s", at)
	co.Debugf(ctxt, "Registering VolumeCapability %s", mo)
	(*md)["access-type"] = at
	(*md)["access-fs"] = fs
	(*md)["access-mode"] = mo
}

func handleControllerPublishVolume(vid, nid string, capabiltity *csi.VolumeCapability, readOnly bool, secrets, attrs map[string]string) {
}

func (d *Driver) initFunc(ctx context.Context, piece, funcName string, req interface{}) context.Context {
	ctxt := co.WithCtxt(ctx, fmt.Sprintf("%s.%s", piece, funcName))
	d.dc.WithContext(ctxt)
	co.Infof(ctxt, "Controller server '%s' called\n", funcName)
	co.Debugf(ctxt, "%s: %+v", funcName, req)
	return ctxt
}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	ctxt := d.initFunc(ctx, "controller", "CreateVolume", *req)
	// Handle req.Name
	id := co.GenName(req.Name)

	// Check to see if a volume already exists with this name
	if vol, err := d.dc.GetVolume(id, false); err == nil {
		return &csi.CreateVolumeResponse{
			Volume: &csi.Volume{
				CapacityBytes: int64(vol.Size),
				Id:            vol.Name,
				Attributes:    map[string]string{},
			},
		}, nil
	}

	// Handle req.AccessibilityRequirements.  Currently we just error out if a topology requirement exists
	// TODO: Digest this beast: https://github.com/container-storage-interface/spec/blob/master/lib/go/csi/v0/csi.pb.go#L1431
	if err := handleTopologyRequirement(req.AccessibilityRequirements); err != nil {
		return nil, err
	}

	md := &dc.VolMetadata{}
	(*md)["display-name"] = req.Name
	registerMdFromCtxt(ctxt, md)

	vcs := req.VolumeCapabilities
	for _, vc := range vcs {
		registerVolumeCapability(ctxt, md, vc)
	}
	// Handle req.Parameters
	params, err := parseVolParams(ctxt, req.Parameters)
	if err != nil {
		return nil, err
	}

	// Handle req.VolumeContentSource
	cs := req.VolumeContentSource
	if snap := cs.GetSnapshot(); snap != nil {
		if err = validateSnapId(snap.Id); err != nil {
			return nil, err
		}
		params.CloneSnapSrc = snap.Id
	}

	// Handle req.CapacityRange
	cr := req.CapacityRange
	if cr != nil && cr.RequiredBytes >= cr.LimitBytes {
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
		return nil, err
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
			CapacityBytes: int64(size),
			Id:            vol.Name,
			Attributes:    map[string]string{},
			ContentSource: nil,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	ctxt := d.initFunc(ctx, "controller", "DeleteVolume", *req)
	vid := req.VolumeId
	// Handle req.ControllerDeleteSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// sec := req.ControllerDeleteSecrets
	if err := d.dc.DeleteVolume(req.VolumeId, true); err != nil {
		co.Errorf(ctxt, "Error deleting volume: %s.  err: %s", vid, err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	d.initFunc(ctx, "controller", "ControllerPublishVolume", *req)
	h, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return &csi.ControllerPublishVolumeResponse{
		PublishInfo: map[string]string{
			"controller_host": h,
		},
	}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (d *Driver) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{
		Supported: true,
		Message:   "",
	}, nil
}

func (d *Driver) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	ctxt := d.initFunc(ctx, "controller", "ListVolumes", *req)
	var err error
	st := int64(0)
	if req.StartingToken != "" {
		st, err = strconv.ParseInt(req.StartingToken, 0, 0)
		if err != nil {
			return nil, err
		}
	}
	vols, err := d.dc.ListVolumes(int(req.MaxEntries), int(st))
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
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
	ctxt := d.initFunc(ctx, "controller", "GetCapacity", *req)
	cap, err := d.dc.GetCapacity()
	if err != nil {
		return nil, err
	}
	params, err := parseVolParams(ctxt, req.Parameters)
	if err != nil {
		return nil, err
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
	d.initFunc(ctx, "controller", "ControllerGetCapabilities", *req)
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
					},
				},
			},
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
					},
				},
			},
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
	return &csi.CreateSnapshotResponse{}, nil
}

func (d *Driver) DeleteSnapshot(ctx context.Context, req *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return &csi.DeleteSnapshotResponse{}, nil
}

func (d *Driver) ListSnapshots(ctx context.Context, req *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return &csi.ListSnapshotsResponse{}, nil
}

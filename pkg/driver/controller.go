package driver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"

	dc "github.com/Datera/datera-csi/pkg/client"
	co "github.com/Datera/datera-csi/pkg/common"
)

const (
	DefaultSize = 16
)

type VolMetadata map[string]string

func parseVolParams(params map[string]string) (*dc.VolOpts, error) {
	//Golang makes something that should be simple, repetative and gross
	dparams := make(map[string]string, 13)
	vo := &dc.VolOpts{}
	var err error
	for k := range params {
		if strings.HasPrefix("DF:", k) {
			nk := strings.TrimLeft("DF:", k)
			dparams[nk] = params[k]
		}
	}
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
	if len(tr.Requisite) > 0 || len(tr.Preferred) > 0 {
		return fmt.Errorf("TopologyRequirements and Preferred Topologies are currently unsupported")
	}
	return nil
}

func handleControllerPublishVolume(vid, nid string, capabiltity *csi.VolumeCapability, readOnly bool, secrets, attrs map[string]string) {
}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	logp := log.WithField("method", "create_volume")
	logp.Info("Controller server 'CreateVolume' called")
	logp.Debugf("CreateVolumeRequest: %+v", *req)

	// Handle req.AccessibilityRequirements.  Currently we just error out if a topology requirement exists
	// TODO: Digest this beast: https://github.com/container-storage-interface/spec/blob/master/lib/go/csi/v0/csi.pb.go#L1431
	if err := handleTopologyRequirement(req.AccessibilityRequirements); err != nil {
		return nil, err
	}

	md := make(VolMetadata)

	// Handle req.Name
	id := co.GenName(req.Name)
	md["display-name"] = req.Name

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
	// Record req.VolumeCapabilities in metadata
	vcs := req.VolumeCapabilities
	for i, vc := range vcs {
		s := string(i)
		var at string
		switch vc.GetAccessType().(type) {
		case *csi.VolumeCapability_Block:
			at = "block"
		case *csi.VolumeCapability_Mount:
			at = "mount"
		default:
			at = ""
		}
		md["access-type-"+s] = at
		md["access-mode-"+s] = string(vc.GetAccessMode().Mode)
	}
	// Handle req.Parameters
	params, err := parseVolParams(req.Parameters)
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

	// Create AppInstance/StorageInstance/Volume
	vol, err := d.dc.CreateVolume(id, params, true)
	if err != nil {
		return nil, err
	}

	// Handle req.ControllerCreateSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// handleVolSecrets(req.ControllerCreateSecrets)

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
	logp := log.WithField("method", "delete_volume")
	logp.Info("Controller server 'DeleteVolume' called")
	logp.Debugf("DeleteVolumeRequest: %+v", *req)
	vid := req.VolumeId
	// Handle req.ControllerDeleteSecrets
	// TODO: Figure out what we want to do with secrets (software encryption maybe?)
	// sec := req.ControllerDeleteSecrets
	if err := d.dc.DeleteVolume(req.VolumeId, true); err != nil {
		logp.Errorf("Error deleting volume: %s.  err: %s", vid, err)
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	logp := log.WithField("method", "controller_publish_volume")
	logp.Info("Controller server 'ControllerPublishVolume' called")
	logp.Debugf("ControllerPublishVolumeRequest: %+v", *req)
	vid := req.VolumeId
	nid := req.NodeId
	vc := req.VolumeCapability
	ro := req.Readonly
	cps := req.ControllerPublishSecrets
	va := req.VolumeAttributes
	handleControllerPublishVolume(vid, nid, vc, ro, cps, va)
	return &csi.ControllerPublishVolumeResponse{
		PublishInfo: map[string]string{},
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
	return &csi.ListVolumesResponse{
		Entries: []*csi.ListVolumesResponse_Entry{},
	}, nil
}

func (d *Driver) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return &csi.GetCapacityResponse{}, nil
}

func (d *Driver) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{},
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

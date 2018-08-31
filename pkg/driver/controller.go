package driver

import (
	"context"
	"fmt"
	"strings"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	units "github.com/docker/go-units"
	log "github.com/sirupsen/logrus"

	co "github.com/Datera/datera-csi/pkg/common"
)

type VolMetadata map[string]string

func handleVolParams(params map[string]string) map[string]string {
	// pull out everything prefixed with 'DF:'
	dparams := make(map[string]string, 10)
	for k := range params {
		if strings.HasPrefix("DF:", k) {
			nk := strings.TrimLeft("DF:", k)
			dparams[nk] = params[k]
		}
	}
	return dparams
}

func handleVolSecrets(secrets map[string]string) {
}

func handleSnap(id string) {
}

func handleTopologyRequirement(tr *csi.TopologyRequirement) {
}

func handleVolumeDelete(vid string, secrets map[string]string) error {
	return nil
}

func handleControllerPublishVolume(vid, nid string, capabiltity *csi.VolumeCapability, readOnly bool, secrets, attrs map[string]string) {
}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	logp := log.WithField("method", "create_volume")
	logp.Info("Controller server 'CreateVolume' called")
	logp.Debugf("CreateVolumeRequest: %+v", *req)

	md := make(VolMetadata)

	// Handle req.Name
	id := co.GenName(req.Name)

	// Handle req.CapacityRange
	cr := req.CapacityRange
	if cr.RequiredBytes >= cr.LimitBytes {
		return &csi.CreateVolumeResponse{}, fmt.Errorf("RequiredBytes must be less than or equal to LimitBytes: [%d, %d]", cr.RequiredBytes, cr.LimitBytes)
	}
	// Default to LimitBytes since we've verified that it's larger than RequiredBytes
	size := cr.LimitBytes / units.GiB
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
	params := handleVolParams(req.Parameters)
	println(params)

	// Handle req.ControllerCreateSecrets
	handleVolSecrets(req.ControllerCreateSecrets)

	// Handle req.VolumeContentSource
	cs := req.VolumeContentSource
	if snap := cs.GetSnapshot(); snap != nil {
		handleSnap(snap.Id)
	}

	// Handle req.AccessibilityRequirements
	handleTopologyRequirement(req.AccessibilityRequirements)

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: size,
			Id:            id,
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
	sec := req.ControllerDeleteSecrets
	err := handleVolumeDelete(vid, sec)
	// Always handle errors gracefully for delete.  Log them and return
	if err != nil {
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

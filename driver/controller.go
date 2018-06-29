package driver

import (
	"context"
	"fmt"

	csi "github.com/container-storage-interface/spec/lib/go/csi/v0"
	log "github.com/sirupsen/logrus"
)

type MockVolume struct {
	Id   string
	Size int64
}

type VolMetadata map[string]string

func handleVolParams(params map[string]string) {
}

func handleVolSecrets(secrets map[string]string) {
}

func handleSnap(id string) {
}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	logp := log.WithField("method", "create_volume")
	logp.Info("Controller server 'CreateVolume' called")
	logp.Debugf("CreateVolumeRequest: %+v", *req)

	md := make(VolMetadata)

	// Handle req.Name
	var volName string
	if req.Name != "" {
		volName = req.Name
	} else {
		volName = genVolName()
	}

	// Handle req.CapacityRange
	cr := req.CapacityRange
	if cr.RequiredBytes >= cr.LimitBytes {
		return &csi.CreateVolumeResponse{}, fmt.Errorf("RequiredBytes must be less than or equal to LimitBytes: [%d, %d]", cr.RequiredBytes, cr.LimitBytes)
	}

	// Handle req.VolumeCapabilities
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
	handleVolParams(req.Parameters)

	// Handle req.ControllerCreateSecrets
	handleVolSecrets(req.ControllerCreateSecrets)

	// Handle req.VolumeContentSource
	cs := req.VolumeContentSource
	if snap := cs.GetSnapshot(); snap != nil {
		handleSnap(snap.Id)
	}

	// Handle req.AccessibilityRequirements

	vol := MockVolume{Id: volName}
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes: vol.Size,
			Id:            vol.Id,
			Attributes:    map[string]string{},
			ContentSource: nil,
		},
	}, nil
}

func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
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

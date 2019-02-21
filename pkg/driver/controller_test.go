package driver

import (
	"context"
	"fmt"
	"testing"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
)

func getDriverController(t *testing.T) *Driver {
	conf, err := udc.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDateraDriver(conf)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func createVolume(t *testing.T, d *Driver) (string, *csi.Volume, func()) {
	if resp, err := d.CreateVolume(context.Background(), &csi.CreateVolumeRequest{
		Name: "csi-controller-test-" + dsdk.RandString(5),
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 10737418240,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			&csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{
						FsType: "ext4",
					},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{
			"replica_count": "1",
		},
	}); err != nil {
		t.Fatal(err)
	} else {
		id := resp.Volume.VolumeId
		cleanf := func() {
			if _, err := d.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{
				VolumeId: id,
			}); err != nil {
				t.Fatal(err)
			}

		}
		return id, resp.Volume, cleanf
	}
	return "", nil, func() {}
}

func createVolumeWithSnapshot(t *testing.T, d *Driver) (string, *csi.Volume, *csi.Snapshot, func()) {
	id, vol, cleanf := createVolume(t, d)
	if resp, err := d.CreateSnapshot(context.Background(), &csi.CreateSnapshotRequest{
		SourceVolumeId: id,
		Name:           "csi-controller-snapshot-test-" + dsdk.RandString(5),
	}); err != nil {
		t.Fatal(err)
	} else {
		snapid := resp.Snapshot.SnapshotId
		cleanf2 := func() {
			if _, err := d.DeleteSnapshot(context.Background(), &csi.DeleteSnapshotRequest{
				SnapshotId: snapid,
			}); err != nil {
				t.Fatal(err)
			}
			cleanf()
		}
		return snapid, vol, resp.Snapshot, cleanf2
	}
	return "", nil, nil, func() {}
}

func TestControllerCreateVolumeDeleteVolume(t *testing.T) {
	d := getDriverController(t)
	var id string
	if resp, err := d.CreateVolume(context.Background(), &csi.CreateVolumeRequest{
		Name: "csi-controller-test-" + dsdk.RandString(5),
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 10737418240,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			&csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{
						FsType: "ext4",
					},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{
			"replica_count": "1",
		},
	}); err != nil {
		t.Fatal(err)
	} else {
		id = resp.Volume.VolumeId
	}

	if _, err := d.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{
		VolumeId: id,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerCreateDeleteSnapshot(t *testing.T) {
	d := getDriverController(t)
	var snapid string
	id, _, cleanf := createVolume(t, d)
	defer cleanf()
	if resp, err := d.CreateSnapshot(context.Background(), &csi.CreateSnapshotRequest{
		SourceVolumeId: id,
		Name:           "csi-controller-snapshot-test-" + dsdk.RandString(5),
	}); err != nil {
		t.Fatal(err)
	} else {
		snapid = resp.Snapshot.SnapshotId
	}

	if _, err := d.DeleteSnapshot(context.Background(), &csi.DeleteSnapshotRequest{
		SnapshotId: snapid,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerCreateVolSnapshotVolumeSource(t *testing.T) {
	d := getDriverController(t)
	snapid, _, _, cleanf := createVolumeWithSnapshot(t, d)
	defer cleanf()
	var volid string
	if resp, err := d.CreateVolume(context.Background(), &csi.CreateVolumeRequest{
		Name: "csi-controller-test-" + dsdk.RandString(5),
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: 10737418240,
		},
		VolumeCapabilities: []*csi.VolumeCapability{
			&csi.VolumeCapability{
				AccessType: &csi.VolumeCapability_Mount{
					Mount: &csi.VolumeCapability_MountVolume{
						FsType: "ext4",
					},
				},
				AccessMode: &csi.VolumeCapability_AccessMode{
					Mode: csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
				},
			},
		},
		Parameters: map[string]string{
			"replica_count": "1",
		},
		VolumeContentSource: &csi.VolumeContentSource{
			Type: &csi.VolumeContentSource_Snapshot{
				Snapshot: &csi.VolumeContentSource_SnapshotSource{
					SnapshotId: snapid,
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	} else {
		volid = resp.Volume.VolumeId
	}

	if _, err := d.DeleteVolume(context.Background(), &csi.DeleteVolumeRequest{
		VolumeId: volid,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerGetCapacity(t *testing.T) {
	d := getDriverController(t)
	if resp, err := d.GetCapacity(context.Background(), &csi.GetCapacityRequest{}); err != nil {
		t.Fatal(err)
	} else {
		if resp.AvailableCapacity <= 0 {
			t.Fatal(fmt.Errorf("Available capacity is lower than expected: %d", resp.AvailableCapacity))
		}
	}
}

func TestControllerListVolumes(t *testing.T) {
	d := getDriverController(t)
	_, _, cleanf := createVolume(t, d)
	defer cleanf()
	if resp, err := d.ListVolumes(context.Background(), &csi.ListVolumesRequest{
		MaxEntries: 1,
	}); err != nil {
		t.Fatal(err)
	} else {
		if len(resp.Entries) != 1 {
			t.Fatal(fmt.Errorf("Volumes list did not return expected number of volumes. Expected 1, Found %d", len(resp.Entries)))
		}
	}
}

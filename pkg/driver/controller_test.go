package driver

import (
	"context"
	"fmt"
	"testing"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	units "github.com/docker/go-units"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

func getCtxt() context.Context {
	return context.WithValue(context.Background(), co.TraceId, co.GenId())
}

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
	if resp, err := d.CreateVolume(getCtxt(), &csi.CreateVolumeRequest{
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
			if _, err := d.DeleteVolume(getCtxt(), &csi.DeleteVolumeRequest{
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
	if resp, err := d.CreateSnapshot(getCtxt(), &csi.CreateSnapshotRequest{
		SourceVolumeId: id,
		Name:           "csi-controller-snapshot-test-" + dsdk.RandString(5),
	}); err != nil {
		t.Fatal(err)
	} else {
		snapid := resp.Snapshot.SnapshotId
		cleanf2 := func() {
			if _, err := d.DeleteSnapshot(getCtxt(), &csi.DeleteSnapshotRequest{
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
	if resp, err := d.CreateVolume(getCtxt(), &csi.CreateVolumeRequest{
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

	if _, err := d.DeleteVolume(getCtxt(), &csi.DeleteVolumeRequest{
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
	if resp, err := d.CreateSnapshot(getCtxt(), &csi.CreateSnapshotRequest{
		SourceVolumeId: id,
		Name:           "csi-controller-snapshot-test-" + dsdk.RandString(5),
	}); err != nil {
		t.Fatal(err)
	} else {
		snapid = resp.Snapshot.SnapshotId
	}

	if _, err := d.DeleteSnapshot(getCtxt(), &csi.DeleteSnapshotRequest{
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
	if resp, err := d.CreateVolume(getCtxt(), &csi.CreateVolumeRequest{
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

	if _, err := d.DeleteVolume(getCtxt(), &csi.DeleteVolumeRequest{
		VolumeId: volid,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerGetCapacity(t *testing.T) {
	d := getDriverController(t)
	if resp, err := d.GetCapacity(getCtxt(), &csi.GetCapacityRequest{}); err != nil {
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
	if resp, err := d.ListVolumes(getCtxt(), &csi.ListVolumesRequest{
		MaxEntries: 1,
	}); err != nil {
		t.Fatal(err)
	} else {
		if len(resp.Entries) != 1 {
			t.Fatal(fmt.Errorf("Volumes list did not return expected number of volumes. Expected 1, Found %d", len(resp.Entries)))
		}
	}
}

func TestControllerExpandVolumes(t *testing.T) {
	d := getDriverController(t)
	vid, vol, cleanf := createVolume(t, d)
	defer cleanf()
	if resp, err := d.ControllerExpandVolume(getCtxt(), &csi.ControllerExpandVolumeRequest{
		VolumeId: vid,
		CapacityRange: &csi.CapacityRange{
			RequiredBytes: vol.CapacityBytes + 1*units.GiB,
		},
	}); err != nil {
		t.Fatal(err)
	} else {
		vol, err := d.dc.GetVolume(vid, false, false)
		if err != nil {
			t.Fatal(err)
		}
		vsize := int64(vol.Size * units.GiB)
		if resp.CapacityBytes != vsize {
			t.Fatalf("CapacityBytes did not match volume size: [%d != %d]", resp.CapacityBytes, vsize)
		}
	}
}

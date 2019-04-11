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

func getDriverNode(t *testing.T) *Driver {
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

func TestNodeStageVolumeUnstageVolume(t *testing.T) {
	c := getDriverController(t)
	n := getDriverNode(t)
}

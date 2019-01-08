package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func (d *Driver) getManifestData() (map[string]string, error) {
	//TODO(_alastor_): Populate manifest with Datera DSP information
	mf, err := d.dc.GetManifest()
	if err != nil {
		return map[string]string{}, err
	}
	manifest := map[string]string{
		"build_version":       mf.BuildVersion,
		"callhome_enabled":    mf.CallhomeEnabled,
		"compression_enabled": mf.CompressionEnabled,
		"health":              mf.Health,
		"l3_enabled":          mf.L3Enabled,
		"name":                mf.Name,
		"op_state":            mf.OpState,
		"sw_version":          mf.SwVersion,
		"timezone":            mf.Timezone,
		"uuid":                mf.Uuid,
	}
	return manifest, nil
}

func (d *Driver) GetPluginInfo(ctx context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	d.InitFunc(ctx, "identity", "GetPluginInfo", *req)
	manifest, err := d.getManifestData()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, err.Error())
	}
	vv, err := d.dc.VendorVersion()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, err.Error())
	}
	return &csi.GetPluginInfoResponse{
		Name:          driverName,
		VendorVersion: vv,
		Manifest:      manifest,
	}, nil
}

func (d *Driver) GetPluginCapabilities(ctx context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	d.InitFunc(ctx, "identity", "GetPluginCapabilities", *req)
	return &csi.GetPluginCapabilitiesResponse{
		Capabilities: []*csi.PluginCapability{
			{
				Type: &csi.PluginCapability_Service_{
					Service: &csi.PluginCapability_Service{
						Type: csi.PluginCapability_Service_CONTROLLER_SERVICE,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) Probe(ctx context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	d.InitFunc(ctx, "identity", "Probe", *req)
	if err := d.dc.HealthCheck(); err != nil {
		return nil, status.Errorf(codes.Unavailable, err.Error())
	}
	return &csi.ProbeResponse{
		Ready: &wrappers.BoolValue{Value: true},
	}, nil
}

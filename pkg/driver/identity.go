package driver

import (
	"context"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	log "github.com/sirupsen/logrus"
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

func (d *Driver) GetPluginInfo(ctxt context.Context, req *csi.GetPluginInfoRequest) (*csi.GetPluginInfoResponse, error) {
	log.WithField("method", "get_plugin_info").Info("Identity server 'GetPluginInfo' called")
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

func (d *Driver) GetPluginCapabilities(ctxt context.Context, req *csi.GetPluginCapabilitiesRequest) (*csi.GetPluginCapabilitiesResponse, error) {
	log.WithField("method", "get_plugin_capabilities").Info("Identity server 'GetPluginCapabilities' called")
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

func (d *Driver) Probe(ctxt context.Context, req *csi.ProbeRequest) (*csi.ProbeResponse, error) {
	log.WithField("method", "probe").Info("Identity server 'Probe' called")
	if err := d.dc.HealthCheck(); err != nil {
		return nil, status.Errorf(codes.Unavailable, err.Error())
	}
	return &csi.ProbeResponse{
		Ready: &wrappers.BoolValue{Value: true},
	}, nil
}

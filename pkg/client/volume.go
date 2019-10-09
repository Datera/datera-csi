package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"encoding/json"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type VolOpts struct {
	Size                    int      `json:"size,omitempty"`
	Replica                 int      `json:"replica,omitempty"`
	Template                string   `json:"template,omitempty"`
	RemoteProvider          string   `json:"remote_provider,omitempty"`
	FsType                  string   `json:"fs_type,omitempty"`
	FsArgs                  []string `json:"fs_args,omitempty"`
	PlacementMode           string   `json:"placement,omitempty"`
	PlacementPolicy         string   `json:"placement_policy,omitempty"`
	CloneSrc                string   `json:"clone_src,omitempty"`
	CloneVolSrc             string   `json:"clone_vol_src,omitempty"`
	CloneSnapSrc            string   `json:"clone_snap_src,omitempty"`
	IpPool                  string   `json:"ip_pool,omitempty"`
	RoundRobin              bool     `json:"round_robin,omitempty"`
	DeleteOnUnmount         bool     `json:"delete_on_unmount,omitempty"`
	DisableTemplateOverride bool     `json:"disable_template_override,omitempty"`

	// QoS IOPS
	WriteIopsMax int `json:"write_iops_max,omitempty"`
	ReadIopsMax  int `json:"read_iops_max,omitempty"`
	TotalIopsMax int `json:"total_iops_max,omitempty"`

	// QoS Bandwidth
	WriteBandwidthMax int `json:"write_bandwidth_max,omitempty"`
	ReadBandwidthMax  int `json:"read_bandwidth_max,omitempty"`
	TotalBandwidthMax int `json:"total_bandwidth_max,omitempty"`

	// Dynamic QoS
	IopsPerGb      int `json:"iops_per_gb,omitempty"`
	BandwidthPerGb int `json:"bandwidth_per_gb,omitempty"`
}

type Volume struct {
	ctxt           context.Context
	dc             *DateraClient
	Ai             *dsdk.AppInstance
	Name           string
	AdminState     string
	RepairPriority string
	Template       string

	TargetOpState string
	Ips           []string
	Iqn           string
	Initiators    []string

	Replicas        int
	PlacementMode   string
	PlacementPolicy string
	Size            int

	// QoS in map form, mostly for logging
	QoS map[string]int

	// Direct Access QoS Iops
	WriteIopsMax int
	ReadIopsMax  int
	TotalIopsMax int
	// Direct Access QoS Iops
	WriteBandwidthMax int
	ReadBandwidthMax  int
	TotalBandwidthMax int

	DevicePath     string
	MountPath      string
	BindMountPaths *dsdk.StringSet
	FsType         string
	FsArgs         []string
	Formatted      bool
}

type VolMetadata map[string]string

// This is used to force expensive checking behavior to ensure we're not
// sending more metadata than can be processed (2048 characters)
var MetadataDebug = false

func (v VolOpts) ToMap() map[string]string {
	return map[string]string{
		"size":                      strconv.FormatInt(int64(v.Size), 10),
		"replica":                   strconv.FormatInt(int64(v.Replica), 10),
		"template":                  v.Template,
		"fs_type":                   v.FsType,
		"fs_args":                   strings.Join(v.FsArgs, " "),
		"placement":                 v.PlacementMode,
		"clone_src":                 v.CloneSrc,
		"clone_vol_src":             v.CloneVolSrc,
		"clone_snap_src":            v.CloneSnapSrc,
		"ip_pool":                   v.IpPool,
		"round_robin":               strconv.FormatBool(v.RoundRobin),
		"delete_on_unmount":         strconv.FormatBool(v.DeleteOnUnmount),
		"disable_template_override": strconv.FormatBool(v.DisableTemplateOverride),

		// QoS IOPS
		"write_iops_max": strconv.FormatInt(int64(v.WriteIopsMax), 10),
		"read_iops_max":  strconv.FormatInt(int64(v.ReadIopsMax), 10),
		"total_iops_max": strconv.FormatInt(int64(v.TotalIopsMax), 10),

		// QoS Bandwidth
		"write_bandwidth_max": strconv.FormatInt(int64(v.WriteBandwidthMax), 10),
		"read_bandwidth_max":  strconv.FormatInt(int64(v.ReadBandwidthMax), 10),
		"total_bandwidth_max": strconv.FormatInt(int64(v.TotalBandwidthMax), 10),

		// Dynamic QoS
		"iops_per_gb":      strconv.FormatInt(int64(v.IopsPerGb), 10),
		"bandwidth_per_gb": strconv.FormatInt(int64(v.BandwidthPerGb), 10),
	}
}

func aiToClientVol(ctx context.Context, ai *dsdk.AppInstance, qos, metadata bool, client *DateraClient) (*Volume, error) {
	ctxt := context.WithValue(ctx, co.ReqName, "aiToClientVol")
	if ai == nil {
		return nil, fmt.Errorf("Cannot construct a Client Volume from a nil AppInstance")
	}
	si := ai.StorageInstances[0]
	v := si.Volumes[0]
	inits := []string{}
	for _, init := range si.AclPolicy.Initiators {
		inits = append(inits, init.Name)
	}
	var pp map[string]int
	if qos && client != nil {
		resp, apierr, err := v.PerformancePolicy.Get(&dsdk.PerformancePolicyGetRequest{
			Ctxt: ctxt,
		})
		if err != nil {
			co.Error(ctxt, err)
			return nil, err
		} else if apierr != nil {
			co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
			return nil, co.ErrTranslator(apierr)
		}
		pp = map[string]int{"read_iops_max": resp.ReadIopsMax,
			"write_iops_max":      resp.WriteIopsMax,
			"total_iops_max":      resp.TotalIopsMax,
			"read_bandwidth_max":  resp.ReadBandwidthMax,
			"write_bandwidth_max": resp.WriteBandwidthMax,
			"total_bandwidth_max": resp.TotalBandwidthMax,
		}
	}

	vol := &Volume{
		ctxt:           ctxt,
		dc:             client,
		Ai:             ai,
		Name:           ai.Name,
		AdminState:     ai.AdminState,
		RepairPriority: ai.RepairPriority,
		Template:       ai.AppTemplate.Path,

		TargetOpState: si.OpState,
		Ips:           si.Access.Ips,
		Iqn:           si.Access.Iqn,
		Initiators:    inits,

		Replicas:      v.ReplicaCount,
		PlacementMode: v.PlacementMode,
		Size:          v.Size,

		QoS:               pp,
		ReadIopsMax:       pp["read_iops_max"],
		WriteIopsMax:      pp["write_iops_max"],
		TotalIopsMax:      pp["total_iops_max"],
		ReadBandwidthMax:  pp["read_bandwidth_max"],
		WriteBandwidthMax: pp["write_bandwidth_max"],
		TotalBandwidthMax: pp["total_bandwidth_max"],
	}

	if metadata {
		md, err := vol.GetMetadata()
		if err != nil {
			return nil, err
		}
		var fm bool
		fmj, ok := (*md)["formatted"]
		if !ok || fmj == "false" {
			fm = false
		} else {
			fm = true
		}
		fsType, fsArgs := (*md)["fs_type"], strings.Split((*md)["fs_args"], " ")
		vol.DevicePath = (*md)["device_path"]
		vol.MountPath = (*md)["mount_path"]
		vol.BindMountPaths = dsdk.NewStringSet(10, strings.Split((*md)["bind-mount-paths"], " ")...)
		vol.FsType = fsType
		vol.FsArgs = fsArgs
		vol.Formatted = fm
	}

	co.Debugf(ctxt, "Instantiated Client Volume: %#v", vol)
	return vol, nil
}

func (r *DateraClient) GetVolume(name string, qos, metadata bool) (*Volume, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetVolume")
	co.Debugf(ctxt, "GetVolume invoked for %s", name)
	if name == "" {
		return nil, fmt.Errorf("Volume name cannot be an empty string")
	}
	newAi, apierr, err := r.sdk.AppInstances.Get(&dsdk.AppInstancesGetRequest{
		Ctxt: ctxt,
		Id:   name,
	})
	if err != nil {
		return nil, err
	}
	if apierr != nil {
		return nil, co.ErrTranslator(apierr)
	}
	v, err := aiToClientVol(ctxt, newAi, qos, metadata, r)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *DateraClient) CreateVolume(name string, volOpts *VolOpts, qos bool) (*Volume, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateVolume")
	co.Debugf(ctxt, "CreateVolume invoked for %s, volOpts: %#v", name, volOpts)
	var ai dsdk.AppInstancesCreateRequest
        var mode string = "kubernetes"
	if volOpts.Template != "" {
		// From Template
		template := strings.Trim(volOpts.Template, "/")
		co.Debugf(ctxt, "Creating AppInstance with template: %s", template)
		at := &dsdk.AppInstanceAppTemplate{
			Path: "/app_templates/" + template,
		}
		ai = dsdk.AppInstancesCreateRequest{
			Ctxt:        ctxt,
			Name:        name,
                        CreateMode:  mode,
			AppTemplate: at,
		}
		if !volOpts.DisableTemplateOverride {
			ai.TemplateOverride = map[string]interface{}{
				"storage_instances": map[string]interface{}{
					"storage-1": map[string]interface{}{
						"volumes": map[string]interface{}{
							"volume-1": map[string]interface{}{
								"size": strconv.FormatInt(int64(volOpts.Size), 10),
							},
						},
					},
				},
			}
		}
	} else if volOpts.CloneVolSrc != "" {
		// Clone Volume
		c := &dsdk.Volume{Path: volOpts.CloneVolSrc}
		co.Debugf(ctxt, "Creating AppInstance from Volume clone: %s", volOpts.CloneVolSrc)
		ai = dsdk.AppInstancesCreateRequest{
			Ctxt:           ctxt,
			Name:           name,
                        CreateMode:     mode,
			CloneVolumeSrc: c,
		}
	} else if volOpts.CloneSnapSrc != "" {
		// Clone Snapshot
		c := &dsdk.Snapshot{Path: volOpts.CloneSnapSrc}
		co.Debugf(ctxt, "Creating AppInstance from Snapshot clone: %s", volOpts.CloneSrc)
		ai = dsdk.AppInstancesCreateRequest{
			Ctxt:             ctxt,
			Name:             name,
                        CreateMode:       mode,
			CloneSnapshotSrc: c,
		}
	} else {
		// Vanilla Volume Create
		var vol *dsdk.Volume
		if yes, err := co.DatVersionGte(r.vendorVersion, "3.3.0.0"); err != nil && yes {
			vol = &dsdk.Volume{
				Name:          "volume-1",
				Size:          int(volOpts.Size),
				PlacementMode: volOpts.PlacementMode,
				PlacementPolicy: &dsdk.PlacementPolicy{
					Path: volOpts.PlacementPolicy,
				},
				ReplicaCount: int(volOpts.Replica),
			}
		} else if err != nil {
			co.Errorf(ctxt, "Could not determine vendor version: %s", r.vendorVersion)
			return nil, err
		} else {
			vol = &dsdk.Volume{
				Name:          "volume-1",
				Size:          int(volOpts.Size),
				PlacementMode: volOpts.PlacementMode,
				ReplicaCount:  int(volOpts.Replica),
			}
		}
		si := &dsdk.StorageInstance{
			Name: "storage-1",
			IpPool: &dsdk.AccessNetworkIpPool{
				Path: fmt.Sprintf("/access_network_ip_pools/%s", volOpts.IpPool),
			},
			Volumes: []*dsdk.Volume{vol},
		}
		ai = dsdk.AppInstancesCreateRequest{
			Ctxt:             ctxt,
			Name:             name,
                        CreateMode:       mode,
			StorageInstances: []*dsdk.StorageInstance{si},
		}
	}
	newAi, apierr, err := r.sdk.AppInstances.Create(&ai)
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	v, err := aiToClientVol(ctxt, newAi, false, false, r)
	v.Formatted = false
	if qos && volOpts.Template == "" {
		if err = v.SetPerformancePolicy(volOpts); err != nil {
			return nil, err
		}
	}
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	return v, nil
}

func (r *DateraClient) DeleteVolume(name string, force bool) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "DeleteVolume")
	co.Debugf(ctxt, "DeleteVolume invoked for %s", name)
	ai, apierr, err := r.sdk.AppInstances.Get(&dsdk.AppInstancesGetRequest{
		Ctxt: ctxt,
		Id:   name,
	})
	v, err := aiToClientVol(ctxt, ai, false, false, r)
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	// Kube doesn't perform this check for us, so we need to stop any deletion
	// of a volume currently possessing snapshots to avoid unintentional data loss.
	snaps, err := v.HasSnapshots()
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if snaps {
		err = fmt.Errorf("Volume %s cannot be deleted because it has snapshots", v.Name)
		co.Error(ctxt, err)
		return err
	}
	return v.Delete(force)
}

func (r *Volume) Delete(force bool) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "Delete")
	co.Debugf(ctxt, "Volume Delete invoked for %s", r.Name)
	_, apierr, err := r.Ai.Set(&dsdk.AppInstanceSetRequest{
		Ctxt:       ctxt,
		AdminState: "offline",
		Force:      force,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	_, apierr, err = r.Ai.Delete(&dsdk.AppInstanceDeleteRequest{
		Ctxt:  ctxt,
		Force: force,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	return nil
}

func (r *DateraClient) ListVolumes(maxEntries int, startToken int) ([]*Volume, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListVolumes")
	co.Debug(ctxt, "ListVolumes invoked\n")
	params := dsdk.ListParams{
		Limit:  maxEntries,
		Offset: startToken,
	}
	resp, apierr, err := r.sdk.AppInstances.List(&dsdk.AppInstancesListRequest{
		Ctxt:   ctxt,
		Params: params,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	vols := []*Volume{}
	for _, ai := range resp {
		v, err := aiToClientVol(ctxt, ai, false, false, r)
		if err != nil {
			co.Error(ctxt, err)
			continue
		}
		vols = append(vols, v)
	}
	return vols, nil
}

func (r *Volume) SetPerformancePolicy(volOpts *VolOpts) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "SetPerformancePolicy")
	co.Debugf(ctxt, "SetPerformancePolicy invoked for %s, volOpts: %#v", r.Name, volOpts)
	ai := r.Ai
	im := volOpts.TotalIopsMax
	bm := volOpts.TotalBandwidthMax
	if volOpts.IopsPerGb != 0 {
		ipg := volOpts.IopsPerGb * volOpts.Size
		// Not using zero, because zero means unlimited
		if ipg < im {
			im = ipg
		}
	}
	if volOpts.BandwidthPerGb != 0 {
		bpg := volOpts.BandwidthPerGb * volOpts.Size
		// Not using zero, because zero means unlimited
		if bpg < bm {
			bm = bpg
		}
	}
	pp := dsdk.PerformancePolicyCreateRequest{
		Ctxt:              ctxt,
		ReadIopsMax:       int(volOpts.ReadIopsMax),
		WriteIopsMax:      int(volOpts.WriteIopsMax),
		TotalIopsMax:      int(im),
		ReadBandwidthMax:  int(volOpts.ReadBandwidthMax),
		WriteBandwidthMax: int(volOpts.WriteBandwidthMax),
		TotalBandwidthMax: int(bm),
	}
	resp, apierr, err := ai.StorageInstances[0].Volumes[0].PerformancePolicy.Create(&pp)
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	r.QoS = map[string]int{
		"read_iops_max":       resp.ReadIopsMax,
		"write_iops_max":      resp.WriteIopsMax,
		"total_iops_max":      resp.TotalIopsMax,
		"read_bandwidth_max":  resp.ReadBandwidthMax,
		"write_bandwidth_max": resp.WriteBandwidthMax,
		"total_bandwidth_max": resp.TotalBandwidthMax,
	}
	r.ReadIopsMax = resp.ReadIopsMax
	r.WriteIopsMax = resp.WriteIopsMax
	r.TotalIopsMax = resp.TotalIopsMax
	r.ReadBandwidthMax = resp.ReadBandwidthMax
	r.WriteBandwidthMax = resp.WriteBandwidthMax
	r.TotalBandwidthMax = resp.TotalBandwidthMax
	return nil
}

func (r *Volume) GetMetadata() (*VolMetadata, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetMetadata")
	co.Debugf(ctxt, "GetMetadata invoked for %s", r.Name)
	resp, apierr, err := r.Ai.GetMetadata(&dsdk.AppInstanceMetadataGetRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	result := VolMetadata(*resp)
	return &result, nil
}

func (r *Volume) SetMetadata(metadata *VolMetadata) (*VolMetadata, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "SetMetadata")
	co.Debugf(ctxt, "SetMetadata invoked for %s", r.Name)
	if MetadataDebug {
		co.Debugf(ctxt, "Running size check on metadata")
		tmd, err := r.GetMetadata()
		if err != nil {
			co.Error(ctxt, err)
			return nil, err
		}
		for k, v := range *metadata {
			(*tmd)[k] = v
		}
		b, err := json.Marshal(tmd)
		if err != nil {
			co.Error(ctxt, err)
			return nil, err
		}
		co.Debugf(ctxt, "Size of Metadata in bytes: %d", len(b))
		co.Debugf(ctxt, "Size of Metadata in runes: %d", len([]rune(string(b))))
	}
	resp, apierr, err := r.Ai.SetMetadata(&dsdk.AppInstanceMetadataSetRequest{
		Ctxt:     ctxt,
		Metadata: *metadata,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	result := VolMetadata(*resp)
	return &result, nil
}

func (r *Volume) GetUsage() (int, int, int) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetUsage")
	co.Debugf(ctxt, "GetUsage invoked for %s", r.Name)
	v := r.Ai.StorageInstances[0].Volumes[0]
	size := v.Size
	used := v.CapacityInUse
	avail := size - used
	return size, used, avail
}

func (r *Volume) Reload(qos, metadata bool) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "Volume Reload")
	co.Debugf(ctxt, "Volume Reload invoked: %s", r.Name)
	newAi, apierr, err := r.Ai.Reload(&dsdk.AppInstanceReloadRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	v, err := aiToClientVol(ctxt, newAi, qos, metadata, r.dc)
	// Update reciever
	*r = *v
	return nil
}

func (r *Volume) Resize(newSize int) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "Volume Resize")
	co.Debugf(ctxt, "Volume Resize invoked: %s", r.Name)

	v := r.Ai.StorageInstances[0].Volumes[0]
	_, apierr, err := v.Set(&dsdk.VolumeSetRequest{
		Ctxt: ctxt,
		Size: newSize,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	return r.Reload(false, false)
}

func (r *Volume) Online() error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "Volume Reload")
	co.Debugf(ctxt, "Volume Reload invoked: %s", r.Name)
	_, apierr, err := r.Ai.Set(&dsdk.AppInstanceSetRequest{
		Ctxt:       ctxt,
		AdminState: "online",
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	return r.Reload(false, false)
}

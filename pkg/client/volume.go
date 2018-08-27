package client

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type VolOpts struct {
	Size            int
	Replica         int
	Template        string
	FsType          string
	FsArgs          string
	PlacementMode   string
	CloneSrc        string
	RoundRobin      bool
	DeleteOnUnmount bool

	// QoS IOPS
	WriteIopsMax int
	ReadIopsMax  int
	TotalIopsMax int

	// QoS Bandwidth
	WriteBandwidthMax int
	ReadBandwidthMax  int
	TotalBandwidthMax int

	// Dynamic QoS
	IopsPerGb      int
	BandwidthPerGb int
}

func (r DateraClient) CreateVolume(ctxt context.Context, name string, volOpts *VolOpts) error {
	log.Debugf("CreateVolume invoked for %s, volOpts: %#v", name, volOpts)
	var ai dsdk.AppInstancesCreateRequest
	if volOpts.Template != "" {
		template := strings.Trim(volOpts.Template, "/")
		log.Debugf("Creating AppInstance with template: %s", template)
		at := dsdk.AppTemplate{
			Path: "/app_templates/" + template,
		}
		ai = dsdk.AppInstancesCreateRequest{
			Name:        name,
			AppTemplate: at,
		}
	} else if volOpts.CloneSrc != "" {
		c := dsdk.AppInstance{Path: "/app_instances/" + volOpts.CloneSrc}
		log.Debugf("Creating AppInstance from clone: %s", volOpts.CloneSrc)
		ai = dsdk.AppInstancesCreateRequest{
			Name:     name,
			CloneSrc: c,
		}
	} else {
		vol := dsdk.Volume{
			Name:          "volume-1",
			Size:          int(volOpts.Size),
			PlacementMode: volOpts.PlacementMode,
			ReplicaCount:  int(volOpts.Replica),
		}
		si := dsdk.StorageInstance{
			Name:    "storage-1",
			Volumes: []dsdk.Volume{vol},
		}
		ai = dsdk.AppInstancesCreateRequest{
			Name:             name,
			StorageInstances: []dsdk.StorageInstance{si},
		}
	}
	resp, err := r.sdk.AppInstances.Create(&ai)
	if err != nil {
		log.Error(err)
		return err
	}
	newAi := dsdk.AppInstance(*resp)
	// Handle QoS values
	pp := dsdk.PerformancePolicyCreateRequest{
		ReadIopsMax:       int(volOpts.ReadIopsMax),
		WriteIopsMax:      int(volOpts.WriteIopsMax),
		TotalIopsMax:      int(volOpts.TotalIopsMax),
		ReadBandwidthMax:  int(volOpts.ReadBandwidthMax),
		WriteBandwidthMax: int(volOpts.WriteBandwidthMax),
		TotalBandwidthMax: int(volOpts.TotalBandwidthMax),
	}
	_, err = newAi.StorageInstances[0].Volumes[0].PerformancePolicyEp.Create(&pp)
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (r DateraClient) DeleteVolume(ctxt context.Context, name string, volOpts *VolOpts) error {
	resp, err := r.sdk.AppInstances.Get(&dsdk.AppInstancesGetRequest{
		Id: name,
	})
	if err != nil {
		log.Error(err)
		return err
	}
	ai := dsdk.AppInstance(*resp)
	_, err = ai.Delete(&dsdk.AppInstanceDeleteRequest{})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

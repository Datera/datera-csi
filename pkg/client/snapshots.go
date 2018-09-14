package client

import (
	"context"
	"fmt"
	"sort"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type Snapshot struct {
	ctxt   context.Context
	Snap   *dsdk.Snapshot
	Vol    *dsdk.Volume
	Id     string
	Path   string
	Status string
}

func (r DateraClient) ListSnapshots(sourceVol string, maxEntries, startToken int) ([]*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debugf(ctxt, "ListSnapshots invoked for %s", sourceVol)
	var err error
	vols, snaps := []*Volume{}, []*Snapshot{}
	if sourceVol == "" {
		vols, err = r.ListVolumes(0, 0)
		if err != nil {
			return nil, err
		}
	} else {
		vol, err := r.GetVolume(sourceVol, false)
		if err != nil {
			return nil, err
		}
		vols = append(vols, vol)
	}
	sres := make(chan *Snapshot, 100)
	for _, vol := range vols {
		go func() {
			psnaps, err := vol.ListSnapshots("", 0, 0)
			if err != nil {
				co.Error(ctxt, err)
				return
			}
			for _, snap := range psnaps {
				sres <- snap
			}
		}()

	}
	for {
		select {
		case snap := <-sres:
			snaps = append(snaps, snap)
		default:
			break
		}
	}
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Id < snaps[j].Id
	})
	return snaps[startToken : startToken+maxEntries], nil
}

func (r *Volume) CreateSnapshot() (*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateSnapshot")
	co.Debugf(ctxt, "CreateSnapshot invoked for %s", r.Name)
	snap, apierr, err := r.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.Create(&dsdk.SnapshotsCreateRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, fmt.Errorf("ApiError: %#v", *apierr)
	}
	return &Snapshot{
		ctxt:   r.ctxt,
		Snap:   snap,
		Vol:    r.Ai.StorageInstances[0].Volumes[0],
		Id:     snap.UtcTs,
		Path:   snap.Path,
		Status: snap.OpState,
	}, nil
}

func (r *Volume) DeleteSnapshot(id string) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "DeleteSnapshot")
	co.Debugf(ctxt, "DeleteSnapshot invoked for %s", r.Name)
	var found *dsdk.Snapshot
	for _, snap := range r.Ai.StorageInstances[0].Volumes[0].Snapshots {
		if snap.Uuid == id || snap.UtcTs == id {
			found = snap
		}
	}
	if found == nil {
		// Fail gracefully
		co.Warningf(ctxt, "No Snapshot found with Id or UtcTs matching %s", id)
		return nil
	}
	_, apierr, err := found.Delete(&dsdk.SnapshotDeleteRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return fmt.Errorf("ApiError: %#v", *apierr)
	}
	return nil
}

func (r *Volume) ListSnapshots(snapId string, maxEntries int, startToken int) ([]*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debug(ctxt, "ListSnapshots invoked")
	snaps := []*Snapshot{}
	params := dsdk.ListParams{
		Limit:  maxEntries,
		Offset: startToken,
	}
	v := r.Ai.StorageInstances[0].Volumes[0]
	rsnaps, apierr, err := v.SnapshotsEp.List(&dsdk.SnapshotsListRequest{
		Ctxt:   ctxt,
		Params: params,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, fmt.Errorf("ApiError: %#v", *apierr)
	}
	for _, s := range rsnaps {
		if snapId == "" || snapId == s.UtcTs {
			snaps = append(snaps, &Snapshot{
				ctxt:   r.ctxt,
				Snap:   s,
				Vol:    r.Ai.StorageInstances[0].Volumes[0],
				Id:     s.UtcTs,
				Path:   s.Path,
				Status: s.OpState,
			})
		}
	}
	return snaps, nil
}

func (s *Snapshot) Reload() error {
	ctxt := context.WithValue(s.ctxt, co.ReqName, "Snapshot Reload")
	co.Debug(ctxt, "Snapshot Reload invoked")
	snap, apierr, err := s.Vol.SnapshotsEp.Get(&dsdk.SnapshotsGetRequest{
		Ctxt:      ctxt,
		Timestamp: s.Id,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return fmt.Errorf("ApiError: %#v", *apierr)
	}
	s.Snap = snap
	s.Status = snap.OpState
	return nil
}

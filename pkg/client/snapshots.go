package client

import (
	"context"
	"fmt"
	"sort"
	"sync"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type Snapshot struct {
	ctxt   context.Context
	Snap   *dsdk.Snapshot
	Vol    *Volume
	Id     string
	Path   string
	Status string
}

func (r DateraClient) ListSnapshots(snapId, sourceVol string, maxEntries, startToken int) ([]*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debugf(ctxt, "ListSnapshots invoked for %s\n", sourceVol)
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
	wg := sync.WaitGroup{}
	addL := sync.Mutex{}
	addSnaps := func(psnaps []*Snapshot) {
		addL.Lock()
		defer addL.Unlock()
		snaps = append(snaps, psnaps...)
	}
	for _, vol := range vols {
		wg.Add(1)
		go func(v *Volume) {
			psnaps, err := v.ListSnapshots(snapId, 0, 0)
			if err != nil {
				co.Error(ctxt, err)
				wg.Done()
				return
			}
			addSnaps(psnaps)
			wg.Done()
		}(vol)

	}
	co.Debug(ctxt, "Waiting")
	wg.Wait()
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Id < snaps[j].Id
	})
	end := startToken + maxEntries
	if end == 0 {
		end = len(snaps)
	}
	co.Debugf(ctxt, "Returning snapshots: %#v", snaps[startToken:end])
	return snaps[startToken:end], nil
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
	v, err := AiToClientVol(ctxt, r.Ai, false, nil)
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	return &Snapshot{
		ctxt:   r.ctxt,
		Snap:   snap,
		Vol:    v,
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
	co.Debugf(ctxt, "Volume %s ListSnapshots invoked\n", r.Name)
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
			v, err := AiToClientVol(ctxt, r.Ai, false, nil)
			if err != nil {
				co.Error(ctxt, err)
				return nil, err
			}
			snaps = append(snaps, &Snapshot{
				ctxt:   r.ctxt,
				Snap:   s,
				Vol:    v,
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
	snap, apierr, err := s.Vol.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.Get(&dsdk.SnapshotsGetRequest{
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

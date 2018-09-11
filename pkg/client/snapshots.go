package client

import (
	"context"
	"fmt"
	// "strings"
	"strconv"

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

func (r *Volume) ListSnapshots(snapId string, maxEntries int, startToken string) ([]*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debug(ctxt, "ListSnapshots invoked")
	snaps := []*Snapshot{}
	params := map[string]string{
		"limit":  strconv.FormatInt(int64(maxEntries), 10),
		"offset": startToken,
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
		snaps = append(snaps, &Snapshot{
			ctxt:   r.ctxt,
			Snap:   s,
			Vol:    r.Ai.StorageInstances[0].Volumes[0],
			Id:     s.UtcTs,
			Path:   s.Path,
			Status: s.OpState,
		})
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

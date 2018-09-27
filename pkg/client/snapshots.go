package client

import (
	"context"
	"fmt"
	"sort"
	"sync"

	uuid "github.com/google/uuid"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

const SnapDomainStr = "7079EAEC-2660-4A35-9A48-9C47204C01A9"

var SnapDomain *uuid.UUID

type Snapshot struct {
	ctxt   context.Context
	Snap   *dsdk.Snapshot
	Vol    *Volume
	Id     string
	Path   string
	Status string
}

func initSnapDomain() *uuid.UUID {
	sd, err := uuid.Parse(SnapDomainStr)
	if err != nil {
		panic(err)
	}
	return &sd
}

func snapIdFromName(ctxt context.Context, name string) *uuid.UUID {
	sid := uuid.NewSHA1(*SnapDomain, []byte(name))
	co.Debugf(ctxt, "Generating Snapshot Id %s from name %s", sid.String(), name)
	return &sid
}

func (r DateraClient) ListSnapshots(snapId, sourceVol string, maxEntries, startToken int) ([]*Snapshot, int, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debugf(ctxt, "ListSnapshots invoked.  snapId = %s, sourceVol = %s, maxEntries = %d, startToken = %d\n", snapId, sourceVol, maxEntries, startToken)
	var (
		err   error
		vid   string
		sid   string
		vols  = []*Volume{}
		snaps = []*Snapshot{}
	)
	if snapId != "" {
		vid, sid = co.ParseSnapId(snapId)
		if vid == "" || sid == "" {
			return []*Snapshot{}, 0, fmt.Errorf("SnapshotId must be of format app_instance_name:snapshot_timestamp")
		}
		if sourceVol == "" {
			sourceVol = vid
		}
	}

	if vid != "" && sid != "" {
		vol, err := r.GetVolume(vid, false)
		if err != nil {
			return nil, 0, err
		}
		snaps, err = vol.ListSnapshots(sid, 0, 0)
	} else {
		if sourceVol == "" {
			vols, err = r.ListVolumes(0, 0)
			if err != nil {
				return nil, 0, err
			}
		} else {
			vol, err := r.GetVolume(sourceVol, false)
			if err != nil {
				return nil, 0, err
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
				psnaps, err := v.ListSnapshots(sid, 0, 0)
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
	}
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Id < snaps[j].Id
	})
	end := startToken + maxEntries
	if end == 0 {
		end = len(snaps)
	}
	co.Debugf(ctxt, "Returning snapshots: %#v", snaps[startToken:end])
	return snaps[startToken:end], end, nil
}

func (r *Volume) GetSnapshotByUuid(id *uuid.UUID) (*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetSnapshotByUuid")
	co.Debugf(ctxt, "GetSnapshotByUuid invoked for %s", r.Name)
	snaps, apierr, err := r.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.List(&dsdk.SnapshotsListRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, fmt.Errorf("ApiError: %#v", *apierr)
	}
	for _, snap := range snaps {
		if snap.Uuid == id.String() {
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
	}
	return nil, fmt.Errorf("No snapshot found with UUID %s", id.String())

}

func (r *Volume) CreateSnapshot(name string) (*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateSnapshot")
	co.Debugf(ctxt, "CreateSnapshot invoked for %s", r.Name)
	sid := snapIdFromName(ctxt, name)
	snap, apierr, err := r.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.Create(&dsdk.SnapshotsCreateRequest{
		Ctxt: ctxt,
		Uuid: sid.String(),
	})
	if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		// Duplicate found
		if apierr.Name == "InvalidRequestError" && apierr.Code == 15 {
			return r.GetSnapshotByUuid(sid)
		}
		return nil, fmt.Errorf("ApiError: %#v", *apierr)
	} else if err != nil {
		co.Error(ctxt, err)
		return nil, err
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

func init() {
	SnapDomain = initSnapDomain()
}

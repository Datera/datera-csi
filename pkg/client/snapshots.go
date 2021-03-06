package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	uuid "github.com/google/uuid"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

// NEVER CHANGE THIS AFTER v1.0 release
const SnapDomainStr = "7079EAEC-2660-4A35-9A48-9C47204C01A9"

var SnapDomain *uuid.UUID

type SnapOpts struct {
	RemoteProviderUuid string
	Type               string
}

type Snapshot struct {
	ctxt   context.Context
	dc     *DateraClient
	Snap   *dsdk.Snapshot
	Vol    *Volume
	Id     string
	Path   string
	Status string
}

// This is a workaround to allow for encoding the name of a CSI snapshot into a
// normal UUID on the Datera side.  The domain of the uuids is hardcoded above
// and should NEVER be changed, otherwise we lose our references to customer
// snapshots between CSI plugin versions
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

func (r *DateraClient) SnapshotPathFromCsiId(csiId string) (string, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "SnapshotPathFromCsiId")
	co.Debugf(ctxt, "SnapshotPathFromCsiId invoked.  csiId: %s", csiId)
	parts := strings.Split(csiId, ":")
	vid := parts[0]
	snapTs := parts[1]
	co.Debugf(ctxt, "Snapshot parts: %s, %s", vid, snapTs)
	vol, err := r.GetVolume(vid, false, false)
	if err != nil {
		co.Errorf(ctxt, "Could not find volume from provided csi snapshot ID: %s, err: %s", csiId, err.Error())
		return "", err
	}
	snaps, err := vol.ListSnapshots(snapTs)
	if len(snaps) != 1 {
		err = fmt.Errorf("Unexpected number of snapshots found for csi snapshot ID: %s, expected 1 found %d", csiId, len(snaps))
		co.Error(ctxt, err)
		return "", err
	}
	return snaps[0].Path, nil
}

func (r *DateraClient) ListSnapshots(snapId, sourceVol string, maxEntries, startToken int) ([]*Snapshot, int, error) {
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
		vol, err := r.GetVolume(vid, false, false)
		if err != nil {
			return nil, 0, err
		}
		snaps, err = vol.ListSnapshots(sid)
	} else {
		// TODO: When the new Snapshots API is available, bypass this slow path
		if sourceVol == "" {
			vols, err = r.ListVolumes(0, 0)
			if err != nil {
				return nil, 0, err
			}
		} else {
			vol, err := r.GetVolume(sourceVol, false, false)
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
				psnaps, err := v.ListSnapshots(sid)
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
	if len(snaps) == 0 || startToken > len(snaps) {
		return snaps, 0, nil
	}
	snapEnd := len(snaps)
	if maxEntries == 0 {
		maxEntries = snapEnd - startToken
	}
	end := startToken + maxEntries
	if end == 0 || end > snapEnd {
		end = snapEnd
	}
	// startToken = 0, maxEntries = len(snap) --> end = 6
	//
	//
	// [0, 1, 2, 3, 4, 5]
	//  |              |
	//  st             end
	//

	// startToken = 3, maxEntries = 0 -- > end = 6
	//
	// [0, 1, 2, 3, 4, 5]
	//           |     |
	//           st    end
	//

	// startToken = 2, maxEntries = 2 -- > end = 4
	//
	// [0, 1, 2, 3, 4, 5]
	//        |  |
	//        st end
	//
	co.Debugf(ctxt, "startToken = %d, maxEntries = %d, end = %d", startToken, maxEntries, end)
	co.Debugf(ctxt, "Found snapshots: %#v", snaps)
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
		return nil, co.ErrTranslator(apierr)
	}
	for _, snap := range snaps {
		if snap.Uuid == id.String() {
			v, err := aiToClientVol(ctxt, r.Ai, false, false, nil)
			if err != nil {
				co.Error(ctxt, err)
				return nil, err
			}
			return &Snapshot{
				ctxt:   r.ctxt,
				dc:     r.dc,
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

func (r *Volume) CreateSnapshot(name string, snapOpts *SnapOpts) (*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateSnapshot")
	co.Debugf(ctxt, "CreateSnapshot invoked for %s", r.Name)
	sid := snapIdFromName(ctxt, name)
	var (
		snap   *dsdk.Snapshot
		apierr *dsdk.ApiErrorResponse
		err    error
	)
	if snapOpts.RemoteProviderUuid != "" {
		snap, apierr, err = r.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.Create(&dsdk.SnapshotsCreateRequest{
			Ctxt:               ctxt,
			Uuid:               sid.String(),
			RemoteProviderUuid: snapOpts.RemoteProviderUuid,
			Type:               snapOpts.Type,
		})
	} else {
		snap, apierr, err = r.Ai.StorageInstances[0].Volumes[0].SnapshotsEp.Create(&dsdk.SnapshotsCreateRequest{
			Ctxt: ctxt,
			Uuid: sid.String(),
		})
	}
	if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		// Duplicate found
		if apierr.Name == "InvalidRequestError" && apierr.Code == 15 {
			return r.GetSnapshotByUuid(sid)
		}
		return nil, co.ErrTranslator(apierr)
	} else if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	v, err := aiToClientVol(ctxt, r.Ai, false, false, nil)
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	csnap := &Snapshot{
		ctxt:   r.ctxt,
		dc:     r.dc,
		Snap:   snap,
		Vol:    v,
		Id:     snap.UtcTs,
		Path:   snap.Path,
		Status: snap.OpState,
	}
	// Poll for availability
	timeout := 30
	for {
		err = csnap.Reload()
		if err != nil {
			return csnap, err
		}
		if csnap.Status == "available" {
			return csnap, nil
		}
		co.Debugf(ctxt, "Snapshot %s is not available yet", csnap.Id)
		time.Sleep(time.Second * 1)
		timeout--
		if timeout <= 0 {
			err := fmt.Errorf("Snapshot %s did not become available before timeout", csnap.Id)
			return csnap, err
		}
	}
}

func (r *Volume) DeleteSnapshot(id string) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "DeleteSnapshot")
	co.Debugf(ctxt, "DeleteSnapshot invoked for %s", r.Name)
	var found *dsdk.Snapshot
	err := r.Reload(false, false)
	if err != nil {
		co.Warning(ctxt, err)
		return nil
	}
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
		return co.ErrTranslator(apierr)
	}
	return nil
}

func (r *Volume) HasSnapshots() (bool, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "HasSnapshots")
	co.Debugf(ctxt, "Volume %s HasSnapshots invoked\n", r.Name)
	snaps, err := r.ListSnapshots("")
	if err != nil {
		return false, err
	}
	return len(snaps) > 0, nil
}

func (r *Volume) ListSnapshots(snapId string) ([]*Snapshot, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "ListSnapshots")
	co.Debugf(ctxt, "Volume %s ListSnapshots invoked. snapId: %s", r.Name, snapId)
	snaps := []*Snapshot{}
	// Reload volume (app_instance) to ensure data is valid
	err := r.Reload(false, false)
	if err != nil {
		co.Warning(ctxt, err)
		return snaps, nil
	}
	v := r.Ai.StorageInstances[0].Volumes[0]
	rsnaps, apierr, err := v.SnapshotsEp.List(&dsdk.SnapshotsListRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	for _, s := range rsnaps {
		if snapId == "" || snapId == s.UtcTs {
			v, err := aiToClientVol(ctxt, r.Ai, false, false, nil)
			if err != nil {
				co.Error(ctxt, err)
				return nil, err
			}
			snaps = append(snaps, &Snapshot{
				ctxt:   r.ctxt,
				dc:     r.dc,
				Snap:   s,
				Vol:    v,
				Id:     s.UtcTs,
				Path:   s.Path,
				Status: s.OpState,
			})
		}
	}
	co.Debugf(ctxt, "Returning Snapshots: %#v", snaps)
	return snaps, nil
}

func (s *Snapshot) Reload() error {
	ctxt := context.WithValue(s.ctxt, co.ReqName, "Snapshot Reload")
	co.Debugf(ctxt, "Snapshot Reload invoked: %s", s.Id)
	snap, apierr, err := s.Snap.Reload(&dsdk.SnapshotReloadRequest{
		Ctxt: ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	s.Snap = snap
	s.Status = snap.OpState
	return nil
}

func init() {
	SnapDomain = initSnapDomain()
}

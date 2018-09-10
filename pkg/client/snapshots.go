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
	Snap *dsdk.Snapshot
	Id   string
}

func (r Volume) CreateSnapshot() (*Snapshot, error) {
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
		Snap: snap,
		Id:   snap.Uuid,
	}, nil
}

func (r Volume) DeleteSnapshot(id string) error {
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
	_, apierr, err := v.SnapshotsEp.List(&dsdk.SnapshotsListRequest{
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
	return snaps, nil
}

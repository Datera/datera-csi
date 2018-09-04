package client

import (
	"context"
	"io/ioutil"
	"strings"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

var (
	initiatorFile = "/etc/iscsi/initiatorname.iscsi"
)

type Initiator struct {
	Name string
	Path string
	Iqn  string
}

// Gets an Initiator path based on IQN.  If that initiator does not exist it creates the Initiator
// then returns the path to the newly created Initiator
func (r DateraClient) CreateGetInitiator() (*Initiator, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateGetInitiator")
	co.Debugf(ctxt, "CreateGetInitiator invoked")
	iqn, err := getClientIqn(ctxt)
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	co.Debugf(ctxt, "CreateGetInitiator invoked for %s", iqn)
	init, apierr, err := r.sdk.Initiators.Get(&dsdk.InitiatorsGetRequest{
		Ctxt: ctxt,
		Id:   iqn,
	})
	if err != nil {
		if apierr.Name != "NotFoundError" {
			co.Error(ctxt, err)
			return nil, err
		}
		init, apierr, err = r.sdk.Initiators.Create(&dsdk.InitiatorsCreateRequest{
			Ctxt: ctxt,
			Name: co.GenName(""),
			Id:   iqn,
		})
		if err != nil || apierr != nil {
			co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
			return nil, err
		}

	}
	return &Initiator{
		Name: init.Name,
		Path: init.Path,
		Iqn:  init.Id,
	}, nil
}

func (r *Volume) RegisterAcl(cinit *Initiator) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateGetInitiator")
	co.Debugf(ctxt, "CreateACL invoked for %s with initiator %s", r.Name, cinit.Name)
	myInit := &dsdk.Initiator{
		Path: cinit.Path,
	}
	// Update existing AclPolicy if it exists
	si := r.Ai.StorageInstances[0]
	acl, apierr, err := si.AclPolicy.Get(&dsdk.AclPolicyGetRequest{Ctxt: ctxt})
	if err != nil || apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return err
	}
	acl.Initiators = append(acl.Initiators, myInit)
	if _, apierr, err = acl.Set(&dsdk.AclPolicySetRequest{
		Ctxt:       ctxt,
		Initiators: acl.Initiators,
	}); err != nil || apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return err
	}
	return nil
}

func getClientIqn(ctxt context.Context) (string, error) {
	// Parse InitiatorName
	dat, err := ioutil.ReadFile(initiatorFile)
	if err != nil {
		co.Debugf(ctxt, "Could not read file %s", initiatorFile)
		return "", err
	}
	iqn := strings.Split(strings.TrimSpace(string(dat)), "=")[1]
	co.Debugf(ctxt, "Obtained client iqn: %s", iqn)

	return iqn, nil
}

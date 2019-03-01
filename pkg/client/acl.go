package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

var (
	initiatorFile = "/etc/iscsi/initiatorname.iscsi"
)

type Initiator struct {
	ctxt context.Context
	Init *dsdk.Initiator
	Name string
	Path string
	Iqn  string
}

// Gets an Initiator path based on IQN.  If that initiator does not exist it creates the Initiator
// then returns the path to the newly created Initiator
func (r DateraClient) CreateGetInitiator() (*Initiator, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "CreateGetInitiator")
	co.Debugf(ctxt, "CreateGetInitiator invoked")
	iqn, err := GetClientIqn(ctxt)
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	co.Debugf(ctxt, "CreateGetInitiator invoked for %s", iqn)
	init, apierr, err := r.sdk.Initiators.Get(&dsdk.InitiatorsGetRequest{
		Ctxt: ctxt,
		Id:   iqn,
	})
	if apierr != nil {
		if apierr.Name != "NotFoundError" {
			co.Error(ctxt, err)
			return nil, err
		}
		init, apierr, err = r.sdk.Initiators.Create(&dsdk.InitiatorsCreateRequest{
			Ctxt:  ctxt,
			Name:  co.GenName(""),
			Id:    iqn,
			Force: true,
		})
		if err != nil {
			co.Error(ctxt, err)
			return nil, err
		} else if apierr != nil {
			co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
			return nil, co.ErrTranslator(apierr)
		}

	}
	return &Initiator{
		ctxt: ctxt,
		Init: init,
		Name: init.Name,
		Path: init.Path,
		Iqn:  init.Id,
	}, nil
}

func (r *Initiator) Delete(quiet bool) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "Initiator Delete")
	co.Debugf(ctxt, "Initiator Delete invoked")
	_, apierr, err := r.Init.Delete(&dsdk.InitiatorDeleteRequest{
		Ctxt: ctxt,
		Id:   r.Iqn,
	})
	if err != nil {
		co.Error(ctxt, err)
		if !quiet {
			return err
		}
	}
	if apierr != nil {
		err = fmt.Errorf(dsdk.Pretty(apierr))
		co.Error(ctxt, err)
		if !quiet {
			return co.ErrTranslator(apierr)
		}
	}
	return nil
}

func (r *Volume) RegisterAcl(cinit *Initiator) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "RegisterAcl")
	co.Debugf(ctxt, "RegisterAcl invoked for %s with initiator %s", r.Name, cinit.Name)
	// Update existing AclPolicy if it exists
	si := r.Ai.StorageInstances[0]
	acl, apierr, err := si.AclPolicy.Get(&dsdk.AclPolicyGetRequest{Ctxt: ctxt})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	acl.Initiators = append(acl.Initiators, &dsdk.Initiator{
		Path: cinit.Path,
	})
	if _, apierr, err = acl.Set(&dsdk.AclPolicySetRequest{
		Ctxt:       ctxt,
		Initiators: acl.Initiators,
	}); err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	return nil
}

func (r *Volume) UnregisterAcl(cinit *Initiator) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "UnregisterAcl")
	co.Debugf(ctxt, "UnregisterAcl invoked for %s with initiator %s", r.Name, cinit.Name)
	// Update existing AclPolicy if it exists
	si := r.Ai.StorageInstances[0]
	acl, apierr, err := si.AclPolicy.Get(&dsdk.AclPolicyGetRequest{Ctxt: ctxt})
	if err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}

	// Remove the matching initiator from the initiators list
	newInits := []*dsdk.Initiator{}
	for _, init := range acl.Initiators {
		if init.Path != cinit.Path {
			newInits = append(newInits, &dsdk.Initiator{
				Path: cinit.Path,
			})
		}
	}
	acl.Initiators = newInits

	if _, apierr, err = acl.Set(&dsdk.AclPolicySetRequest{
		Ctxt:       ctxt,
		Initiators: acl.Initiators,
	}); err != nil {
		co.Error(ctxt, err)
		return err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return co.ErrTranslator(apierr)
	}
	return nil
}

func GetClientIqn(ctxt context.Context) (string, error) {
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

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

// Gets an Initiator path based on IQN.  If that initiator does not exist it creates the Initiator
// then returns the path to the newly created Initiator
func (r DateraClient) CreateGetInitiator(ctxt context.Context) (string, error) {
	iqn, err := getClientIqn(ctxt)
	if err != nil {
		co.Error(ctxt, err)
		return "", err
	}
	co.Debugf(ctxt, "CreateGetInitiator invoked for %s", iqn)
	resp, err := r.sdk.Initiators.Get(&dsdk.InitiatorsGetRequest{
		Id: iqn,
	})
	var init dsdk.Initiator
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			co.Error(ctxt, err)
			return "", err
		}
		cresp, err := r.sdk.Initiators.Create(&dsdk.InitiatorsCreateRequest{
			Name: co.GenName(""),
			Id:   iqn,
		})
		if err != nil {
			co.Error(ctxt, err)
			return "", err
		}
		init = dsdk.Initiator(*cresp)

	} else {
		init = dsdk.Initiator(*resp)
	}
	return init.Path, nil
}

func (r DateraClient) CreateACL(ctxt context.Context, name string) error {
	co.Debugf(ctxt, "CreateACL invoked for %s", name)
	initPath, err := r.CreateGetInitiator(ctxt)
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	myInit := &dsdk.Initiator{
		Path: initPath,
	}
	// Update existing AclPolicy if it exists
	resp, err := r.sdk.AppInstances.Get(&dsdk.AppInstancesGetRequest{
		Id: name,
	})
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	si := dsdk.AppInstance(*resp).StorageInstances[0]
	aclResp, err := si.AclPolicy.Get(&dsdk.AclPolicyGetRequest{})
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	acl := dsdk.AclPolicy(*aclResp)
	acl.Initiators = append(acl.Initiators, myInit)
	if _, err = acl.Set(&dsdk.AclPolicySetRequest{
		Initiators: acl.Initiators,
	}); err != nil {
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

package client

import (
	"context"

	co "github.com/Datera/datera-csi/pkg/common"
	// dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

func (r DateraClient) CreateACL(ctxt context.Context, name string) error {
	co.Debugf(ctxt, "CreateACL invoked for %s", name)
	// name = co.getName(name)
	// initiator, err := getInitName(ctxt)
	// if err != nil {
	// 	co.Error(ctxt, err)
	// 	return err
	// }

	// iep := r.Api.GetEp("initiators")

	// // Check if initiator exists
	// init, err := iep.GetEp(initiator).Get(ctxt)

	// var path string
	// if err != nil {
	// 	// Create the initiator
	// 	iname, _ := dsdk.NewUUID()
	// 	iname = Prefix + iname
	// 	_, err = iep.Create(ctxt, fmt.Sprintf("name=%s", iname), fmt.Sprintf("id=%s", initiator))
	// 	path = fmt.Sprintf("/initiators/%s", initiator)
	// } else {
	// 	path = init.GetM()["path"].(string)
	// }

	// // Register initiator with storage instance
	// myInit := dsdk.Initiator{
	// 	Path: path,
	// }

	// // Update existing AclPolicy if it exists
	// aclep := r.Api.GetEp("app_instances").GetEp(name).GetEp("storage_instances").GetEp(StorageName).GetEp("acl_policy")
	// eacl, err := aclep.Get(ctxt)
	// if err != nil {
	// 	co.Error(ctxt, err)
	// 	return err
	// }
	// acl, err := dsdk.NewAclPolicy(eacl.GetB())
	// if err != nil {
	// 	co.Error(ctxt, err)
	// 	return err
	// }
	// // Add the new initiator to the initiator list
	// newit := append(*acl.Initiators, myInit)
	// acl.Initiators = &newit
	// acl.Path = ""

	// _, err = aclep.Set(ctxt, acl)
	// if err != nil {
	// 	return err
	// }
	return nil
}

package client

import (
	"context"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type IpPool struct {
	ctxt   context.Context
	IpPool *dsdk.AccessNetworkIpPool
	Name   string
	Path   string
}

func (r DateraClient) GetIpPoolFromName(name string) (*IpPool, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetIpPoolFromName")
	co.Debugf(ctxt, "GetIpPoolFromName invoked. Name: %s", name)
	ipp, apierr, err := r.sdk.AccessNetworkIpPools.Get(&dsdk.AccessNetworkIpPoolsGetRequest{
		Ctxt: ctxt,
		Name: name,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	} else if apierr != nil {
		co.Errorf(ctxt, "%s, %s", dsdk.Pretty(apierr), err)
		return nil, co.ErrTranslator(apierr)
	}
	return &IpPool{
		ctxt:   ctxt,
		IpPool: ipp,
		Name:   ipp.Name,
		Path:   ipp.Path,
	}, nil
}

func (r *Volume) RegisterIpPool(ipPool *IpPool) error {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "RegisterIpPool")
	co.Debugf(ctxt, "RegisterIpPool invoked for %s with ipPool %s", r.Name, ipPool)
	si := r.Ai.StorageInstances[0]
	_, apierr, err := si.Set(&dsdk.StorageInstanceSetRequest{
		Ctxt: ctxt,
		IpPool: &dsdk.AccessNetworkIpPool{
			Path: ipPool.Path,
		},
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

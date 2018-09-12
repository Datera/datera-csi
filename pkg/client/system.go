package client

import (
	"context"
	"fmt"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

type Capacity struct {
	Total             int
	Provisioned       int
	FlashTotal        int
	FlashProvisioned  int
	HybridTotal       int
	HybridProvisioned int
}

func (r DateraClient) GetCapacity() (*Capacity, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetCapacity")
	co.Debugf(ctxt, "GetCapacity invoked")
	sys, apierr, err := r.sdk.System.Get(&dsdk.SystemGetRequest{
		Ctxt: r.ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
	}
	if apierr != nil {
		err = fmt.Errorf(dsdk.Pretty(apierr))
		co.Error(ctxt, err)
	}
	return &Capacity{
		Total:             sys.TotalCapacity,
		Provisioned:       sys.TotalProvisionedCapacity,
		FlashTotal:        sys.AllFlashTotalCapacity,
		FlashProvisioned:  sys.AllFlashProvisionedCapacity,
		HybridTotal:       sys.HybridTotalCapacity,
		HybridProvisioned: sys.HybridProvisionedCapacity,
	}, nil
}

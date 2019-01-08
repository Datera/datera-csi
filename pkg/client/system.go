package client

import (
	"context"
	"fmt"
	"strconv"

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

type Manifest struct {
	BuildVersion       string
	CallhomeEnabled    string
	CompressionEnabled string
	Health             string
	L3Enabled          string
	Name               string
	OpState            string
	SwVersion          string
	Timezone           string
	Uuid               string
}

func (r DateraClient) GetCapacity() (*Capacity, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetCapacity")
	co.Debugf(ctxt, "GetCapacity invoked")
	sys, apierr, err := r.sdk.System.Get(&dsdk.SystemGetRequest{
		Ctxt: r.ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	if apierr != nil {
		err = fmt.Errorf(dsdk.Pretty(apierr))
		co.Error(ctxt, err)
		return nil, co.ErrTranslator(apierr)
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

func (r DateraClient) VendorVersion() (string, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "VendorVersion")
	co.Debugf(ctxt, "VendorVersion invoked")
	sys, apierr, err := r.sdk.System.Get(&dsdk.SystemGetRequest{
		Ctxt: r.ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return "", err
	}
	if apierr != nil {
		err = fmt.Errorf(dsdk.Pretty(apierr))
		co.Error(ctxt, err)
		return "", co.ErrTranslator(apierr)
	}
	return sys.BuildVersion, nil
}

func (r DateraClient) GetManifest() (*Manifest, error) {
	ctxt := context.WithValue(r.ctxt, co.ReqName, "GetManifest")
	co.Debugf(ctxt, "GetManifest invoked")
	sys, apierr, err := r.sdk.System.Get(&dsdk.SystemGetRequest{
		Ctxt: r.ctxt,
	})
	if err != nil {
		co.Error(ctxt, err)
		return nil, err
	}
	if apierr != nil {
		err = fmt.Errorf(dsdk.Pretty(apierr))
		co.Error(ctxt, err)
		return nil, co.ErrTranslator(apierr)
	}
	mf := &Manifest{
		BuildVersion:       sys.BuildVersion,
		CallhomeEnabled:    strconv.FormatBool(sys.CallhomeEnabled),
		CompressionEnabled: strconv.FormatBool(sys.CompressionEnabled),
		Health:             sys.Health,
		L3Enabled:          strconv.FormatBool(sys.L3Enabled),
		Name:               sys.Name,
		OpState:            sys.OpState,
		SwVersion:          sys.SwVersion,
		Timezone:           sys.Timezone,
		Uuid:               sys.Uuid,
	}
	return mf, nil
}

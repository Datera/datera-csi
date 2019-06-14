package client

import (
	"context"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

type DateraClient struct {
	sdk           *dsdk.SDK
	udc           *udc.UDC
	ctxt          context.Context
	vendorVersion string
}

func NewDateraClient(udc *udc.UDC, healthcheck bool, driver string) (*DateraClient, error) {
	sdk, err := dsdk.NewSDK(udc, true)
	if err != nil {
		return nil, err
	}
	sdk.SetDriver(driver)
	if healthcheck {
		if err = sdk.HealthCheck(); err != nil {
			return nil, err
		}
	}
	return &DateraClient{
		sdk: sdk,
		udc: udc,
	}, nil
}

func (r *DateraClient) NewContext() context.Context {
	r.ctxt = r.sdk.NewContext()
	return r.ctxt
}

func (r *DateraClient) WithContext(ctxt context.Context) context.Context {
	r.ctxt = r.sdk.WithContext(ctxt)
	return r.ctxt
}

func (r *DateraClient) HealthCheck(ctxt context.Context) (*Manifest, error) {
	r.ctxt = r.WithContext(ctxt)
	return r.GetManifest()
}

func (r *DateraClient) LogPush(ctxt context.Context, rule, rotated string) error {
	ctxt = r.WithContext(ctxt)
	return r.sdk.LogsUpload.RotateUploadRemove(ctxt, rule, rotated)
}

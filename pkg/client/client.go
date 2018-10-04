package client

import (
	"context"

	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

type DateraClient struct {
	sdk  *dsdk.SDK
	udc  *udc.UDC
	ctxt context.Context
}

func NewDateraClient(udc *udc.UDC, healthcheck bool) (*DateraClient, error) {
	sdk, err := dsdk.NewSDK(udc, true)
	if err != nil {
		return nil, err
	}
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

func (r *DateraClient) HealthCheck() error {
	return r.sdk.HealthCheck()
}

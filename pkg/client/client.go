package client

import (
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

type DateraClient struct {
	sdk *dsdk.SDK
	udc *udc.UDC
}

func NewDateraClient(udc *udc.UDC) (*DateraClient, error) {
	sdk, err := dsdk.NewSDK(udc, true)
	if err != nil {
		return nil, err
	}
	if err = sdk.HealthCheck(); err != nil {
		return nil, err
	}
	return &DateraClient{
		sdk: sdk,
		udc: udc,
	}, nil
}

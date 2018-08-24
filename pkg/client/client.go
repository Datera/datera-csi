package client

import (
	sdk "github.com/Datera/go-sdk/pkg/sdk"
	udc "github.com/Datera/go-udc/pkg/udc"
)

type DateraClient struct {
	sdk *sdk.SDK
}

func NewDateraClient(udc *udc.UDC) *DateraClient {
	sdk, err := dsdk.NewSDK(udc, true)
	if err != nil {
		return nil, err
	}
	if err = sdk.HealthCheck(); err != nil {
		return nil, err
	}
	return &DateraClient{}
}

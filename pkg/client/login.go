package client

import (
	"context"

	iscsi "github.com/j-griffith/csi-connectors/iscsi"

	co "github.com/Datera/datera-csi/pkg/common"
)

func (v *Volume) Login(multipath bool) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Login")
	co.Debugf(ctxt, "Login invoked for %s", v.Name)
	c := iscsi.Connector{
		TargetIqn:     v.Iqn,
		TargetPortals: v.Ips,
		Lun:           0,
		Multipath:     multipath,
	}
	path, err := iscsi.Connect(c)
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.DevicePath = path
	return nil
}

func (v *Volume) Logout() error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Logout")
	co.Debugf(ctxt, "Logout invoked for %s", v.Name)
	err := iscsi.Disconnect(v.Iqn, v.Ips)
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.DevicePath = ""
	return nil
}

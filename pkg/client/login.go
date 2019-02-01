package client

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	iscsi "github.com/kubernetes-csi/csi-lib-iscsi/iscsi"

	co "github.com/Datera/datera-csi/pkg/common"
)

func robin() int {
	// Gen new source each time
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Intn(2)
}

func (v *Volume) Login(multipath, round_robin bool) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Login")
	co.Debugf(ctxt, "Login invoked for %s.  Multipath: %t", v.Name, multipath)
	var ips []string
	if multipath {
		if round_robin {
			ips = []string{v.Ips[robin()]}
		} else {
			ips = v.Ips
		}
	} else {
		if round_robin {
			co.Warningf(ctxt, "round_robin not supported on non-multipath environments")
		}
		ips = []string{v.Ips[0]}
	}
	c := iscsi.Connector{
		TargetIqn:     v.Iqn,
		TargetPortals: ips,
		Port:          "3260",
		Lun:           0,
		Multipath:     multipath,
	}
	co.Debugf(ctxt, "ISCSI Connector: %#v", c)
	path, err := iscsi.Connect(c)
	if err != nil {
		co.Error(ctxt, err)
		return err
	}
	if len(path) < 1 {
		err = fmt.Errorf("Recieved no paths from ISCSI connector: %s", path)
		co.Error(ctxt, err)
		return err
	}
	v.DevicePath = path
	co.Debugf(ctxt, "DevicePath for volume %s: %s", v.Name, v.DevicePath)
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

func init() {
	iscsi.EnableDebugLogging(os.Stdout)
}

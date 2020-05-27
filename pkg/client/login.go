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

func (v *Volume) Login(multipath, round_robin bool, chapParams map[string]string) error {
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
	var targets []iscsi.TargetInfo
	for _, Ip := range ips {
		targets = append(targets, iscsi.TargetInfo{v.Iqn, Ip, "3260"})
	}

	secrets := iscsi.Secrets{}
	c := iscsi.Connector{}

	c.Targets = targets
	c.Lun = 0
	c.Multipath = multipath
	c.RetryCount = 3

	if len(chapParams) != 0 {
		secrets.SecretsType = "chap"
		if value, exists := chapParams["node.session.auth.username"]; exists {
			secrets.UserName = value
		}
		if value, exists := chapParams["node.session.auth.password"]; exists {
			secrets.Password = value
		}
		if value, exists := chapParams["node.session.auth.username_in"]; exists {
			secrets.UserNameIn = value
		}
		if value, exists := chapParams["node.session.auth.password_in"]; exists {
			secrets.PasswordIn = value
		}
		c.AuthType = "chap"
		c.SessionSecrets = secrets
		c.DoCHAPDiscovery = true
	} else {
		c.DoDiscovery = true
	}

	iscsi_conn := c
	if secrets.UserName != "" {
		iscsi_conn.SessionSecrets.UserName = "***stripped***"
	}
	if secrets.Password != "" {
		iscsi_conn.SessionSecrets.Password = "***stripped***"
	}
	if secrets.UserNameIn != "" {
		iscsi_conn.SessionSecrets.UserNameIn = "***stripped***"
	}
	if secrets.PasswordIn != "" {
		iscsi_conn.SessionSecrets.PasswordIn = "***stripped***"
	}

	co.Debugf(ctxt, "ISCSI Connector: %#v", iscsi_conn)
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

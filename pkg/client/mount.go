package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
)

func (v *Volume) Format(fsType string, fsArgs []string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Format")
	co.Debugf(ctxt, "Format invoked for %s", v.Name)
	if v.Formatted {
		co.Warningf(ctxt, "Volume %s already formatted: %s, %s", v.Name, v.FsType, v.FsArgs)
		return nil
	}
	if err := format(ctxt, v.DevicePath, fsType, fsArgs); err != nil {
		return err
	}
	v.FsType = fsType
	v.FsArgs = fsArgs
	return nil
}

func format(ctxt context.Context, device, fsType string, fsArgs []string) error {
	cmd := append([]string{fmt.Sprintf("mkfs.%s", fsType), device}, fsArgs...)
	timeout := 10
	for {
		if _, err := co.RunCmd(ctxt, cmd...); err != nil {
			co.Info(ctxt, err)
			if timeout < 0 {
				co.Errorf(ctxt, "Could not format device %s, before timeout reached: %s", device, err.Error())
				return err
			}
			timeout--
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	return nil
}

func (v *Volume) Mount(dest string, options []string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Mount")
	co.Debugf(ctxt, "Mount invoked for %s", v.Name)
	if v.DevicePath == "" {
		return fmt.Errorf("No device path found for volume %s.  Is the volume logged in?", v.Name)
	} else if v.MountPath != "" {
		return fmt.Errorf("Mount path already exists for volume %s", v.Name)
	}
	if err := mount(ctxt, v.DevicePath, dest, options); err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.MountPath = dest
	return nil
}

func (v *Volume) BindMount(dest string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "BindMount")
	co.Debugf(ctxt, "BindMount invoked for %s", v.Name)
	if v.DevicePath == "" {
		return fmt.Errorf("No device path found for volume %s.  Is the volume logged in?", v.Name)
	} else if v.MountPath == "" {
		return fmt.Errorf("Mount path doesn't exist for volume %s, cannot bind-mount an unmounted volume", v.Name)
	}
	if err := mount(ctxt, v.MountPath, dest, []string{"--bind"}); err != nil {
		co.Error(ctxt, err)
		return err
	}
	if v.BindMountPaths == nil {
		v.BindMountPaths = dsdk.NewStringSet(10, dest)
	} else {
		v.BindMountPaths.Add(dest)
	}
	return nil
}

func (v *Volume) UnBindMount(path string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "UnBindMount")
	co.Debugf(ctxt, "UnBindMount invoked for %s", v.Name)
	if v.BindMountPaths == nil || !v.BindMountPaths.Contains(path) {
		return fmt.Errorf("Volume is already unmounted from bind path: %s", path)
	}
	if err := unmount(ctxt, path); err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.BindMountPaths.Delete(path)
	return nil
}

func (v *Volume) Unmount() error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Unmount")
	co.Debugf(ctxt, "Unmount invoked for %s", v.Name)
	if v.MountPath == "" {
		return fmt.Errorf("Volume is already unmounted")
	}
	if err := unmount(ctxt, v.MountPath); err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.MountPath = ""
	return nil
}

func mount(ctxt context.Context, device, dest string, options []string) error {
	// Check/create directory
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		err = os.MkdirAll(dest, os.ModePerm)
		if err != nil {
			return err
		}
	}
	if strings.Contains(device, ":") {
		cmd := []string{"readlink", "-f", device}
		out, err := co.RunCmd(ctxt, cmd...)
		if err != nil {
			return err
		}
		device = out
	}
	// Mount to directory
	cmd := append([]string{"mount", device, dest}, options...)
	_, err := co.RunCmd(ctxt, cmd...)
	return err
}

func unmount(ctxt context.Context, path string) error {
	cmd := []string{"umount", path}
	_, err := co.RunCmd(ctxt, cmd...)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

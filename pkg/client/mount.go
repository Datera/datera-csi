package client

import (
	"context"
	"fmt"

	co "github.com/Datera/datera-csi/pkg/common"
)

func (v *Volume) Format(fsType string, fsArgs []string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Format")
	if v.FsType != "" {
		co.Warningf(ctxt, "Volume %s already formatted: %s, %s", v.Name, v.FsType, v.FsArgs)
		return nil
	}
	if err := format(ctxt, fsType, fsArgs); err != nil {
		return err
	}
	v.FsType = fsType
	v.FsArgs = fsArgs
	return nil
}

func format(ctxt context.Context, fsType string, fsArgs []string) error {
	cmd := append([]string{fmt.Sprintf("mkfs.%s", fsType)}, fsArgs...)
	if _, err := co.RunCmd(ctxt, cmd...); err != nil {
		return err
	}
	return nil
}

func (v *Volume) Mount(dest string, options []string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Mount")
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

func (v *Volume) Unmount() error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Unmount")
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
	cmd := append([]string{"mount", device, dest}, options...)
	_, err := co.RunCmd(ctxt, cmd...)
	return err
}

func unmount(ctxt context.Context, path string) error {
	cmd := []string{"umount", path}
	_, err := co.RunCmd(ctxt, cmd...)
	return err
}

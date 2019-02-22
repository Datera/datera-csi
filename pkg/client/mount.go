package client

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	co "github.com/Datera/datera-csi/pkg/common"
	dsdk "github.com/Datera/go-sdk/pkg/dsdk"
	unix "golang.org/x/sys/unix"
)

var (
	fsTypeDetect = regexp.MustCompile(`TYPE="(?P<fs>.*?)"`)
)

func (v *Volume) Format(fsType string, fsArgs []string, timeout int) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Format")
	co.Debugf(ctxt, "Format invoked for %s", v.Name)
	if v.Formatted {
		co.Warningf(ctxt, "Volume %s already formatted: %s, %s", v.Name, v.FsType, v.FsArgs)
		return nil
	} else if fs, err := findFs(ctxt, v.DevicePath); err == nil {
		v.Formatted = true
		v.FsType = fs
		co.Warningf(ctxt, "Volume %s already formatted: %s", v.Name, v.FsType)
		return nil
	}
	if err := format(ctxt, v.DevicePath, fsType, fsArgs, timeout); err != nil {
		return err
	}
	v.FsType = fsType
	v.FsArgs = fsArgs
	return nil
}

func format(ctxt context.Context, device, fsType string, fsArgs []string, timeout int) error {
	cmd := append(append([]string{fmt.Sprintf("mkfs.%s", fsType)}, fsArgs...), device)
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
	}
	// } else if v.MountPath != "" {
	// 	return fmt.Errorf("Mount path already exists for volume %s", v.Name)
	// }
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

func getMajorMinor(device string) (uint32, uint32, error) {
	s := unix.Stat_t{}
	if err := unix.Stat(device, &s); err != nil {
		return 0, 0, err
	}

	dev := uint64(s.Rdev)
	return unix.Major(dev), unix.Minor(dev), nil
}

func devLink(ctxt context.Context, device, dest string) error {
	major, minor, err := getMajorMinor(device)
	if err != nil {
		return err
	}
	cmd := []string{"mknod", dest, "b", strconv.FormatUint(uint64(major), 10), strconv.FormatUint(uint64(minor), 10)}
	_, err = co.RunCmd(ctxt, cmd...)
	if err != nil {
		return err
	}
	return nil
}

func findFs(ctxt context.Context, device string) (string, error) {
	cmd := []string{"blkid", device}
	out, err := co.RunCmd(ctxt, cmd...)
	if err != nil {
		return "", err
	}
	co.Debugf(ctxt, "Blkid result: %s", out)
	groups := co.GetCaptureGroups(fsTypeDetect, out)
	if k, ok := groups["fs"]; !ok {
		return "", fmt.Errorf("Couldn't find capture group")
	} else {
		return k, nil
	}
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
	if f, err := os.Stat(dest); err != nil && !f.IsDir() && strings.HasPrefix(device, "/dev/") {
		return devLink(ctxt, device, dest)
	} else {
		fs, err := findFs(ctxt, device)
		if fs == "" {
			co.Warning(ctxt, "Couldn't detect filesystem from blkid output")
			// Mount to directory
			cmd := append([]string{"mount", device, dest}, options...)
			_, err = co.RunCmd(ctxt, cmd...)
			return err
		}
		co.Debugf(ctxt, "Detected %s filesystem", fs)
		// Mount to directory
		cmd := append([]string{"mount", "-t", fs, device, dest}, options...)
		_, err = co.RunCmd(ctxt, cmd...)
		return err
	}
}

func unmount(ctxt context.Context, path string) error {
	var err error
	if f, err := os.Stat(path); err != nil && f.IsDir() {
		cmd := []string{"umount", path}
		_, err := co.RunCmd(ctxt, cmd...)
		if err != nil {
			os.RemoveAll(path)
			return err
		}
	}
	if err != nil {
		os.RemoveAll(path)
		return err
	}
	return os.RemoveAll(path)
}

package client

import (
	"context"
	"encoding/json"
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
	} else if mnt, err := findMnt(ctxt, v.DevicePath); err == nil {
		v.Formatted = true
		co.Warningf(ctxt, "Volume %s already formatted and mounted: %s", v.Name, mnt)
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
		if out, err := co.RunCmd(ctxt, cmd...); err != nil {
			co.Info(ctxt, err)
			if out != "" && strings.Contains(out, "will not make a filesystem here") {
				co.Warningf(ctxt, "Device %s is already mounted", device)
				return err
			}
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

func (v *Volume) Mount(dest string, options []string, fs string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "Mount")
	co.Debugf(ctxt, "Mount invoked for %s", v.Name)
	if v.DevicePath == "" {
		return fmt.Errorf("No device path found for volume %s.  Is the volume logged in?", v.Name)
	}
	// } else if v.MountPath != "" {
	// 	return fmt.Errorf("Mount path already exists for volume %s", v.Name)
	// }
	if err := mount(ctxt, v.DevicePath, dest, options, fs); err != nil {
		co.Error(ctxt, err)
		return err
	}
	v.MountPath = dest
	return nil
}

func (v *Volume) BindMount(dest, fs string) error {
	ctxt := context.WithValue(v.ctxt, co.ReqName, "BindMount")
	co.Debugf(ctxt, "BindMount invoked for %s", v.Name)
	if v.DevicePath == "" {
		return fmt.Errorf("No device path found for volume %s.  Is the volume logged in?", v.Name)
	} else if v.MountPath == "" {
		return fmt.Errorf("Mount path doesn't exist for volume %s, cannot bind-mount an unmounted volume", v.Name)
	}
	if err := mount(ctxt, v.MountPath, dest, []string{"--bind"}, fs); err != nil {
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

// This function is for linking a block device to a new location.  This is for raw block-mode support in kubernetes
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

// func findFs(ctxt context.Context, device string) (string, error) {
// 	cmd := []string{"blkid", fmt.Sprintf("'%s'", device), "-c", "/dev/null"}
// 	out, err := co.RunCmd(ctxt, cmd...)
// 	if err != nil {
// 		return "", err
// 	}
// 	co.Debugf(ctxt, "Blkid result (no cache): %s", out)
// 	groups := co.GetCaptureGroups(fsTypeDetect, out)
// 	if k, ok := groups["fs"]; !ok {
// 		cmd := []string{"blkid", "-c", "/dev/null"}
// 		out, err := co.RunCmd(ctxt, cmd...)
// 		if err != nil {
// 			return "", err
// 		}
// 		co.Debugf(ctxt, "Blkid full output (no cache): %s", out)
// 		return "", fmt.Errorf("Couldn't find capture group")
// 	} else {
// 		return k, nil
// 	}
// }

type LsBlk struct {
	BlockDevices []*LsBlkEntry
}

type LsBlkEntry struct {
	Name       string
	FsType     string
	Label      string
	Uuid       string
	MountPoint string
}

func findFs(ctxt context.Context, device string) (string, error) {
	cmd := []string{"lsblk", "-f", device, "--json"}
	out, err := co.RunCmd(ctxt, cmd...)
	if err != nil {
		return "", err
	}
	co.Debugf(ctxt, "lsblk output: %s", out)
	data := &LsBlk{}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return "", err
	}
	if len(data.BlockDevices) < 1 {
		cmd := []string{"lsblk", "-f"}
		out, err := co.RunCmd(ctxt, cmd...)
		if err != nil {
			return "", err
		}
		co.Debugf(ctxt, "lsblk full output: %s", out)
		return "", fmt.Errorf("No block devices returned from lsblk")
	}
	fs := data.BlockDevices[0].FsType
	if fs == "" {
		return "", fmt.Errorf("No filesystem found")
	}
	return fs, nil
}

func findMnt(ctxt context.Context, device string) (string, error) {
	cmd := []string{"grep", fmt.Sprintf("'%s'", device), "/proc/mounts"}
	out, err := co.RunCmd(ctxt, cmd...)
	if err != nil {
		return "", err
	}
	co.Debugf(ctxt, "%s result: %s", cmd, out)
	return out, nil
}

func isDevice(ctxt context.Context, file string) bool {
	f, _ := os.Stat(file)
	if !f.IsDir() && strings.HasPrefix(file, "/dev/") {
		return true
	}
	return false
}

func readlink(ctxt context.Context, device string) (string, error) {
	cmd := []string{"readlink", "-f", device}
	return co.RunCmd(ctxt, cmd...)
}

func deviceFromMount(ctxt context.Context, file string) (string, error) {
	cmd := []string{"sh", "-c", fmt.Sprintf(`"grep '%s' /proc/mounts | awk '{print \$1}'"`, file)}
	out, err := co.RunCmd(ctxt, cmd...)
	if err != nil {
		return "", err
	}
	dev, err := readlink(ctxt, out)
	// If readlink fails, we'll assume the device we pulled from /proc/mounts
	// is correct.  Some versions of readlink won't error out and instead will
	// return the file/directory that was passed in, in which case we'll just
	// return that
	if err != nil {
		return out, nil
	}
	return dev, nil
}

// Cases:
// /dev/disk/by-path/some-ip-and-iqn /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-<uuid>/globalmount
// /var/lib/kubelet/plugins/kubernetes.io/csi/pv/pvc-<uuid>/globalmount /var/lib/kubelet/plugins/kubernetes.io/csi/pv/new_mount

func mount(ctxt context.Context, source, dest string, options []string, fs string) error {
	co.Debugf(ctxt, "mount called. source: %s, dest: %s, options: %s, fs: %s", source, dest, options, fs)

	// Create destination directory if it doesn't exist
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		err = os.MkdirAll(dest, os.ModePerm)
		if err != nil {
			return err
		}
	}
	// Get the original device if this is a mount
	dev, err := deviceFromMount(ctxt, source)

	// If we couldn't resolve, then we're probably working with the device already
	if err != nil || strings.TrimSpace(dev) == "" {
		dev = source
	}

	// Check if we're bind-mounting
	bind := false
	for _, opt := range options {
		if opt == "--bind" {
			bind = true
		}
	}

	cmd := []string{}
	// Mount to directory.  If we're bind-mounting we can't specify filesystem
	if bind {
		cmd = append([]string{"mount", dev, dest}, options...)
	} else {
		cmd = append([]string{"mount", "-t", fs, dev, dest}, options...)
	}
	_, err = co.RunCmd(ctxt, cmd...)
	return err
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

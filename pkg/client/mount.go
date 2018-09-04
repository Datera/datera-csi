package client

import (
	"fmt"
)

func (v *Volume) Mount(dest string) error {
	if v.DevicePath == "" {
		return fmt.Errorf("No device path found for volume %s.  Is the volume logged in?", v.Name)
	} else if v.MountPath != "" {
		return fmt.Errorf("Mount path already exists for volume %s", v.Name)
	}
	return nil
}

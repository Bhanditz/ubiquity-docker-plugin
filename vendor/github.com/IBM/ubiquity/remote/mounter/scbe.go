package mounter

import (
	"fmt"
	"github.com/IBM/ubiquity/remote/mounter/block_device_mounter_utils"
	"github.com/IBM/ubiquity/utils"
	"log"
)

type scbeMounter struct {
	logger                  *log.Logger
	blockDeviceMounterUtils block_device_mounter_utils.BlockDeviceMounterUtils
}

func NewScbeMounter(logger *log.Logger) Mounter {
	blockDeviceMounterUtils := block_device_mounter_utils.NewBlockDeviceMounterUtils()
	return &scbeMounter{logger: logger, blockDeviceMounterUtils: blockDeviceMounterUtils}
}

func (s *scbeMounter) Mount(mountpoint string, volumeConfig map[string]interface{}) (string, error) {
	s.logger.Println("scbeMounter: Mount start")
	defer s.logger.Println("scbeMounter: Mount end")

	// Rescan OS
	if err := s.blockDeviceMounterUtils.RescanAll(true); err != nil {
		s.logger.Printf("RescanAll failed")
		return "", err
	}

	// Discover device
	volumeWWN := volumeConfig["wwn"].(string) // TODO use the const from local/scbe
	devicePath, err := s.blockDeviceMounterUtils.Discover(volumeWWN)
	if err != nil {
		s.logger.Printf(fmt.Sprintf("Discover device WWN [%s] failed", volumeWWN))
		return "", err
	}

	// Create mount point if needed   // TODO consider to move it inside the util
	exec := utils.NewExecutor()
	if _, err := exec.Stat(mountpoint); err != nil {
		s.logger.Printf("Create mountpoint directory " + mountpoint)
		if err := exec.MkdirAll(mountpoint, 0700); err != nil {
			s.logger.Printf("Fail to create mountpoint " + mountpoint)
			return "", err
		}
	}

	// Mount device and mkfs if needed
	fstype := "ext4" // TODO uses volumeConfig['fstype']
	if err := s.blockDeviceMounterUtils.MountDeviceFlow(devicePath, fstype, mountpoint); err != nil {
		s.logger.Printf("Fail to mount the device ", devicePath)
		return "", err
	}

	return mountpoint, nil
}

func (s *scbeMounter) Unmount(volumeConfig map[string]interface{}) error {
	s.logger.Println("scbeMounter: Unmount start")
	defer s.logger.Println("scbeMounter: Unmount end")

	volumeWWN := volumeConfig["wwn"].(string) // TODO use the const from local/scbe
	mountpoint := "/ubiquity/" + volumeWWN    // TODO get the ubiquity prefix from const
	devicePath, err := s.blockDeviceMounterUtils.Discover(volumeWWN)
	if err != nil {
		s.logger.Printf(fmt.Sprintf("Discover device WWN [%s] failed", volumeWWN))
		return err
	}

	if err := s.blockDeviceMounterUtils.UnmountDeviceFlow(devicePath); err != nil {
		s.logger.Printf("Fail to UnmountDeviceFlow the device ", devicePath)
		return err
	}

	s.logger.Printf(fmt.Sprintf("Delete mountpoint directory %s if exist", mountpoint))
	// TODO move this part to the util
	exec := utils.NewExecutor()
	if _, err := exec.Stat(mountpoint); err == nil {
		// TODO consider to add the prefix of the wwn in the OS (multipath -ll output)
		if err := exec.RemoveAll(mountpoint); err != nil {
			s.logger.Printf("Fail to remove mountpoint " + mountpoint)
			return err
		}
	}

	return nil

}
func (s *scbeMounter) ActionAfterDetach(volumeConfig map[string]interface{}) error {
	// Rescan OS
	if err := s.blockDeviceMounterUtils.RescanAll(true); err != nil {
		s.logger.Printf("RescanAll failed")
		return err
	}
	return nil
}

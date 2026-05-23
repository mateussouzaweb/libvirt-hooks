package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Driver represents a device driver on the system
type Driver struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ReadDriver reads the driver name for a given PCI device path
func ReadDriver(device *Device) (*Driver, error) {

	driver := &Driver{
		Name: "",
		Path: "",
	}

	// Check for the presence of the driver path
	driverPath := filepath.Join(device.Path, "driver")
	_, err := os.Stat(driverPath)
	if err != nil && !os.IsNotExist(err) {
		return driver, fmt.Errorf("error checking driver for device at %s: %w", device.Path, err)
	} else if os.IsNotExist(err) {
		return driver, nil
	}

	// Resolve the driver symlink to get the actual driver name
	driverValue, err := filepath.EvalSymlinks(driverPath)
	if err != nil {
		return driver, fmt.Errorf("error reading driver for device at %s: %w", device.Path, err)
	} else {
		driverValue = filepath.Base(driverValue)
	}

	driver.Name = driverValue
	driver.Path = fmt.Sprintf("/sys/bus/pci/drivers/%s", driverValue)

	return driver, nil
}

// BindDriver will bind the given device to the specified driver at system
func BindDriver(driver *Driver, device *Device) error {

	if driver.Path == "" || driver.Name == "" {
		return nil
	}

	bindPath := fmt.Sprintf("%s/bind", driver.Path)
	err := WriteSysFSValue(bindPath, device.ID)
	if err != nil {
		return fmt.Errorf("error binding device %s to driver %s: %w", device.ID, driver.Name, err)
	}

	time.Sleep(time.Second)
	return nil
}

// UnbindDriver will unbind the given device from its current driver at system
func UnbindDriver(driver *Driver, device *Device) error {

	if driver.Path == "" || driver.Name == "" {
		return nil
	}

	unbindPath := fmt.Sprintf("%s/unbind", driver.Path)
	err := WriteSysFSValue(unbindPath, device.ID)
	if err != nil {
		return fmt.Errorf("error unbinding device %s from driver %s: %w", device.ID, driver.Name, err)
	}

	time.Sleep(time.Second)
	return nil
}

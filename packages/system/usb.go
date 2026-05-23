package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// USB represents a USB device on the system
type USB struct {
	ID           string `json:"id"`
	Parent       string `json:"parent"`
	BusNumber    string `json:"busNumber"`
	DevNumber    string `json:"devNumber"`
	Vendor       string `json:"vendor"`
	Product      string `json:"product"`
	Manufacturer string `json:"manufacturer"`
	ProductName  string `json:"productName"`
}

// GetUSBs returns a list of USB devices associated with the given PCI device.
func GetUSBs(device *Device) ([]*USB, error) {

	USBs := make([]*USB, 0)

	// Look for USB root devices: <device>/usb*
	pattern := fmt.Sprintf("%s/usb[0-9]*", device.Path)
	roots, err := FindSysFSFolders(pattern)
	if err != nil {
		return USBs, fmt.Errorf("error searching for USB devices: %w", err)
	}

	// Walk through the root tree to find all USB devices under the given device path
	// A USB device is identified by the presence of idVendor and product files
	results := make([]string, 0)
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip unreadable entries
			}
			if !entry.IsDir() {
				return nil
			}

			idVendorPath := filepath.Join(path, "idVendor")
			if _, err := os.Stat(idVendorPath); err != nil {
				return nil // Not a valid USB device, skip
			}

			productPath := filepath.Join(path, "product")
			if _, err := os.Stat(productPath); err != nil {
				return nil // Not a valid USB device, skip
			}

			results = append(results, path)
			return nil
		})

		if err != nil {
			return USBs, fmt.Errorf("error searching for USB devices: %w", err)
		}
	}

	// Process each found USB device path
	for _, USBPath := range results {
		busNumberPath := filepath.Join(USBPath, "busnum")
		busNumber, err := ReadSysFSValue(busNumberPath)
		if err != nil {
			return USBs, fmt.Errorf("error reading bus number for USB device: %w", err)
		}

		devNumberPath := filepath.Join(USBPath, "devnum")
		devNumber, err := ReadSysFSValue(devNumberPath)
		if err != nil {
			return USBs, fmt.Errorf("error reading device number for USB device: %w", err)
		}

		vendorPath := filepath.Join(USBPath, "idVendor")
		vendor, err := ReadSysFSValue(vendorPath)
		if err != nil {
			return USBs, fmt.Errorf("error reading vendor ID for USB device: %w", err)
		}

		productPath := filepath.Join(USBPath, "idProduct")
		product, err := ReadSysFSValue(productPath)
		if err != nil {
			return USBs, fmt.Errorf("error reading product ID for USB device: %w", err)
		}

		manufacturerPath := filepath.Join(USBPath, "manufacturer")
		manufacturer, err := ReadSysFSValue(manufacturerPath)
		if err != nil {
			return USBs, fmt.Errorf("error reading manufacturer for USB device: %w", err)
		}

		productNamePath := filepath.Join(USBPath, "product")
		productName, err := ReadSysFSValue(productNamePath)
		if err != nil {
			return USBs, fmt.Errorf("error reading product name for USB device: %w", err)
		}

		// Skip entries that don't have valid information
		if vendor == "" && product == "" {
			continue
		}

		deviceID := fmt.Sprintf("%s:%s", vendor, product)
		deviceID = strings.ReplaceAll(deviceID, "0x", "")

		USB := &USB{
			ID:           deviceID,
			Parent:       device.ID,
			BusNumber:    busNumber,
			DevNumber:    devNumber,
			Vendor:       vendor,
			Product:      product,
			Manufacturer: manufacturer,
			ProductName:  productName,
		}

		USBs = append(USBs, USB)
	}

	return USBs, nil
}

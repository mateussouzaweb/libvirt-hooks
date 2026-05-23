package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Device represents a PCI device with its properties
type Device struct {
	IOMMUGroup  string `json:"iommuGroup"`
	ID          string `json:"id"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Vendor      string `json:"vendor"`
	Product     string `json:"product"`
	Description string `json:"description"`
}

// Type returns the type of the PCI device
const TYPE_VGA = "VGA"
const TYPE_AUDIO = "Audio"
const TYPE_NETWORK = "Network"
const TYPE_STORAGE = "Storage"
const TYPE_USB = "USB"
const TYPE_CONTROLLER = "Controller"
const TYPE_OTHER = "Other"

// ExtractDeviceType returns the type of the PCI device based on its description
func ExtractDeviceType(description string) string {

	description = strings.ToLower(strings.TrimSpace(description))

	if strings.Contains(description, "vga") {
		return TYPE_VGA
	} else if strings.Contains(description, "graphics") {
		return TYPE_VGA
	} else if strings.Contains(description, "display") {
		return TYPE_VGA
	} else if strings.Contains(description, "audio") {
		return TYPE_AUDIO
	} else if strings.Contains(description, "multimedia") {
		return TYPE_AUDIO
	} else if strings.Contains(description, "ethernet") {
		return TYPE_NETWORK
	} else if strings.Contains(description, "network") {
		return TYPE_NETWORK
	} else if strings.Contains(description, "wifi") {
		return TYPE_NETWORK
	} else if strings.Contains(description, "wireless") {
		return TYPE_NETWORK
	} else if strings.Contains(description, "usb") {
		return TYPE_USB
	} else if strings.Contains(description, "nvme") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "sata") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "ahci") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "ssd") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "solid state") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "storage") {
		return TYPE_STORAGE
	} else if strings.Contains(description, "controller") {
		return TYPE_CONTROLLER
	}

	return TYPE_OTHER
}

// ReadDeviceDescription runs lspci to get the description of a PCI device by its ID
func ReadDeviceDescription(deviceID string) (string, error) {

	var out bytes.Buffer
	cmd := exec.Command("lspci", "-nns", deviceID)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	description := strings.TrimRight(out.String(), "\n")
	return description, nil
}

// GetDevices returns a list of PCI devices grouped by their IOMMU groups
func GetDevices() ([]*Device, error) {

	devices := make([]*Device, 0)

	// Search for IOMMU groups available on the system
	pattern := "/sys/kernel/iommu_groups/[0-9]*"
	results, err := FindSysFSFolders(pattern)
	if err != nil {
		return devices, fmt.Errorf("error search for PCI devices: %w", err)
	}

	// Collect devices from each group
	// Group Number: e.g. "0", "1", "10", …
	// Device ID: e.g. "0000:00:1f.2"
	for _, groupPath := range results {

		groupNumber := filepath.Base(groupPath)
		devicesPath := filepath.Join(groupPath, "devices")
		deviceEntries, err := os.ReadDir(devicesPath)
		if err != nil {
			return devices, fmt.Errorf("error reading devices in %s: %w", groupPath, err)
		}

		for _, entry := range deviceEntries {
			deviceID := entry.Name()
			devicePath := filepath.Join(devicesPath, entry.Name())

			vendorPath := filepath.Join(devicePath, "vendor")
			vendor, err := ReadSysFSValue(vendorPath)
			if err != nil {
				return devices, fmt.Errorf("error reading vendor for %s: %w", deviceID, err)
			}

			productPath := filepath.Join(devicePath, "device")
			product, err := ReadSysFSValue(productPath)
			if err != nil {
				return devices, fmt.Errorf("error reading product for %s: %w", deviceID, err)
			}

			description, err := ReadDeviceDescription(deviceID)
			if err != nil {
				return devices, fmt.Errorf("error reading description for %s: %w", deviceID, err)
			}

			deviceType := ExtractDeviceType(description)
			device := &Device{
				IOMMUGroup:  groupNumber,
				ID:          deviceID,
				Path:        devicePath,
				Type:        deviceType,
				Vendor:      vendor,
				Product:     product,
				Description: description,
			}

			// Append the device to the list
			devices = append(devices, device)
		}
	}

	// Sort devices by IOMMU group and then by PCI address
	// IOMMUGroup is a string, but represents a number
	sort.Slice(devices, func(i, j int) bool {
		iGroup, _ := strconv.Atoi(devices[i].IOMMUGroup)
		jGroup, _ := strconv.Atoi(devices[j].IOMMUGroup)
		if iGroup == jGroup {
			return devices[i].ID < devices[j].ID
		}
		return iGroup < jGroup
	})

	return devices, nil
}

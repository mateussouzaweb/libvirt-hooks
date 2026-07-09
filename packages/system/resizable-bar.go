package system

import (
	"fmt"
	"math/bits"
)

// ResizableBARSize represents the size of the resizable BAR in MB and its corresponding key
type ResizableBARSize struct {
	Key int `json:"key"`
	MB  int `json:"mb"`
}

// ResizableBAR represent the BAR of one device
type ResizableBAR struct {
	Available bool                `json:"available"`
	Path      string              `json:"path"`
	Sizes     []*ResizableBARSize `json:"sizes"`
}

// ReadResizableBAR detect and give resizable bar information
func ReadResizableBAR(GPU *GPU) (*ResizableBAR, error) {

	BAR := &ResizableBAR{
		Available: false,
		Path:      "",
		Sizes:     make([]*ResizableBARSize, 0),
	}

	// Determine ResizableBAR path based on manufacturer
	// Nvidia GPUS - <device>/resource1_resize
	// AMD GPUs - <device>/resource2_resize
	// Intel GPUs - <device>/resource2_resize
	path := fmt.Sprintf("%s/resource2_resize", GPU.VideoDevice.Path)
	if GPU.Manufacturer == "NVIDIA" {
		path = fmt.Sprintf("%s/resource1_resize", GPU.VideoDevice.Path)
	}

	// Check for the presence of the path
	pathExists, err := FileExists(path)
	if err != nil {
		return BAR, fmt.Errorf("error checking file at %s: %w", path, err)
	} else if !pathExists {
		return BAR, nil
	}

	// Read ResizableBAR supported sizes
	sizesValue, err := ReadSysFSValue(path)
	if err != nil {
		return BAR, fmt.Errorf("error reading resizable BAR supported sizes: %w", err)
	}

	// Convert size from hex string to uint64 and determine supported sizes
	sizesMask := uint64(0)
	if _, err := fmt.Sscanf(sizesValue, "%x", &sizesMask); err != nil {
		return BAR, fmt.Errorf("error parsing resizable BAR mask %q: %w", sizesValue, err)
	}

	// Map supported sizes to their respective values in MB
	for i := range bits.Len64(sizesMask) {
		if sizesMask&(1<<i) != 0 {
			sizeMB := 1 << i
			sizeKey := bits.Len64(uint64(sizeMB)) - 1
			BAR.Sizes = append(BAR.Sizes, &ResizableBARSize{
				Key: sizeKey,
				MB:  sizeMB,
			})
		}
	}

	BAR.Available = true
	BAR.Path = path
	return BAR, nil
}

// SetResizableBAR resize the BAR size for given device
func SetResizableBAR(GPU *GPU, sizeKey int) error {

	if !GPU.ResizableBAR.Available || GPU.ResizableBAR.Path == "" {
		return nil
	}

	if sizeKey == 0 {
		return fmt.Errorf("size key must be greater than 0")
	}

	// Check if the provided size key is supported
	supported := false
	for _, size := range GPU.ResizableBAR.Sizes {
		if size.Key == sizeKey {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("size key %d is not supported by the device", sizeKey)
	}

	// Unbind the device from its current driver to allow resizing
	currentDriver, err := ReadDriver(GPU.VideoDevice)
	if err != nil {
		return fmt.Errorf("error reading driver for device at %s: %w", GPU.VideoDevice.Path, err)
	}

	if currentDriver.Name != "" {
		err = UnbindDriver(currentDriver, GPU.VideoDevice)
		if err != nil {
			return fmt.Errorf("error unbinding device from driver: %w", err)
		}
	}

	// Write the new size key to the resizable BAR path
	resizePath := GPU.ResizableBAR.Path
	sizeValue := fmt.Sprintf("%d", sizeKey)
	err = WriteSysFSValue(resizePath, sizeValue)
	if err != nil {
		return err
	}

	return nil
}

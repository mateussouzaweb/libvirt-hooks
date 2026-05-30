package system

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// GPU represents a GPU on the system
type GPU struct {
	Primary      bool          `json:"primary"`
	Manufacturer string        `json:"manufacturer"`
	ResetMethod  string        `json:"resetMethod"`
	VideoDevice  *Device       `json:"videoDevice"`
	VideoDriver  *Driver       `json:"videoDriver"`
	AudioDevice  *Device       `json:"audioDevice"`
	AudioDriver  *Driver       `json:"audioDriver"`
	ResizableBAR *ResizableBAR `json:"resizableBar"`
}

// GetGPUs returns a list of GPUs available on the system
// Video and audio devices are associated based on their PCI IDs
func GetGPUs() ([]*GPU, error) {

	devices, err := GetDevices()
	if err != nil {
		return nil, err
	}

	// Video and audio devices are often separate
	// We need to check for both and associate them based on their PCI IDs
	GPUs := make([]*GPU, 0)
	primary := true

	for _, videoDevice := range devices {
		if videoDevice.Type != TYPE_VGA {
			continue
		}

		manufacturer := videoDevice.Vendor
		if strings.Contains(videoDevice.Description, "AMD") {
			manufacturer = "AMD"
		} else if strings.Contains(videoDevice.Description, "Intel") {
			manufacturer = "Intel"
		} else if strings.Contains(videoDevice.Description, "NVIDIA") {
			manufacturer = "NVIDIA"
		}

		GPU := &GPU{
			Primary:      false,
			Manufacturer: manufacturer,
			ResetMethod:  "",
			VideoDevice:  videoDevice,
			VideoDriver:  &Driver{},
			AudioDevice:  &Device{},
			AudioDriver:  &Driver{},
			ResizableBAR: &ResizableBAR{},
		}

		// Find for matching device audio
		for _, audioDevice := range devices {
			if audioDevice.Type != TYPE_AUDIO {
				continue
			}
			if audioDevice.ID[:10] != videoDevice.ID[:10] {
				continue
			}

			GPU.AudioDevice = audioDevice
			break
		}

		// Read video driver information
		videoDriver, err := ReadDriver(videoDevice)
		if err != nil {
			return GPUs, fmt.Errorf("error reading video driver for device %s: %w", videoDevice.ID, err)
		} else {
			GPU.VideoDriver = videoDriver
		}

		// Read audio driver information
		audioDriver, err := ReadDriver(GPU.AudioDevice)
		if err != nil {
			return GPUs, fmt.Errorf("error reading audio driver for device %s: %w", GPU.AudioDevice.ID, err)
		} else {
			GPU.AudioDriver = audioDriver
		}

		// Read ResizableBAR
		resizableBAR, err := ReadResizableBAR(GPU)
		if err != nil {
			return GPUs, fmt.Errorf("error reading resizable BAR for device %s: %w", GPU.VideoDevice.ID, err)
		} else {
			GPU.ResizableBAR = resizableBAR
		}

		// Read reset method from video device
		// Audio device should have the same reset method since they share the same PCI ID
		resetMethodPath := fmt.Sprintf("%s/reset_method", GPU.VideoDevice.Path)
		resetMethod, err := ReadSysFSValue(resetMethodPath)
		if err != nil {
			return GPUs, fmt.Errorf("error reading reset method for device %s: %w", GPU.VideoDevice.ID, err)
		} else {
			GPU.ResetMethod = resetMethod
		}

		// Mark the first GPU as primary
		if primary {
			GPU.Primary = true
			primary = false
		}

		GPUs = append(GPUs, GPU)
	}

	return GPUs, nil
}

// ReleaseGPU will release the given GPU from system
func ReleaseGPU(GPU *GPU) error {

	// Unbind video from driver at system
	releasedDrivers := []string{"", "pcieport", "vfio-pci"}
	if !slices.Contains(releasedDrivers, GPU.VideoDriver.Name) {
		err := UnbindDriver(GPU.VideoDriver, GPU.VideoDevice)
		if err != nil {
			return err
		}
	}

	// Unbind audio from driver at system
	if !slices.Contains(releasedDrivers, GPU.AudioDriver.Name) {
		err := UnbindDriver(GPU.AudioDriver, GPU.AudioDevice)
		if err != nil {
			return err
		}
	}

	// Apply ResizableBAR reduction for AMD GPU
	if GPU.VideoDriver.Name == "amdgpu" && GPU.ResizableBAR.Available {
		lowerSize := GPU.ResizableBAR.Sizes[0]
		err := SetResizableBAR(GPU, lowerSize.Key)
		if err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	return nil
}

// RestoreGPU will restore the given GPU to system
func RestoreGPU(GPU *GPU) error {

	// Prevent restoring ResizableBAR to avoid busy error
	GPU.ResizableBAR.Available = false

	// Restore ResizableBAR value to GPU
	if GPU.ResizableBAR.Available {
		maxSize := GPU.ResizableBAR.Sizes[len(GPU.ResizableBAR.Sizes)-1]
		err := SetResizableBAR(GPU, maxSize.Key)
		if err != nil {
			return err
		}

		time.Sleep(time.Second)
	}

	// Restore video driver to the GPU
	restoredDrivers := []string{"", "pcieport", "vfio-pci"}
	if !slices.Contains(restoredDrivers, GPU.VideoDriver.Name) {
		err := BindDriver(GPU.VideoDriver, GPU.VideoDevice)
		if err != nil {
			return err
		}
	}

	// Restore audio driver to the GPU
	if !slices.Contains(restoredDrivers, GPU.AudioDriver.Name) {
		err := BindDriver(GPU.AudioDriver, GPU.AudioDevice)
		if err != nil {
			return err
		}
	}

	return nil
}

package hooks

import (
	"errors"
	"fmt"
	"slices"
	"syscall"
	"time"

	"github.com/mateussouzaweb/libvirt-hooks/packages/state"
	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

// QemuPrepareBegin handles prepare begin qemu hook
func QemuPrepareBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook prepare begin for VM %s", machine.Name))

	// Add the current virtual machine to the state
	machine.Status = system.STATUS_PREPARING
	stateTmp.VMs = append(stateTmp.VMs, machine)

	// Extract necessary information from state to perform required actions
	CPUs := stateTmp.CPUs
	GPUs := stateTmp.GPUs
	displayManager := stateTmp.DisplayManager
	virtualConsoles := stateTmp.VirtualConsoles
	frameBuffers := stateTmp.FrameBuffers

	// Set CPU scaling governor to performance on all cores
	// Action will improve performance and reduce latency
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "scaling CPU to performance mode")
	for _, CPU := range CPUs {
		CPU.ScalingGovernor = "performance"
		err := system.SetCPUScalingGovernor(CPU)
		if err != nil {
			return fmt.Errorf("error occurred while setting CPU scaling governor: %v", err)
		}
	}

	// Read VCPUPin specs to extract which CPUs should be preserved
	preserveCPUs := []*system.CPU{}
	for _, CPU := range CPUs {
		found := slices.Contains(machine.CPUSet, CPU.Number)
		if !found {
			preserveCPUs = append(preserveCPUs, CPU)
		}
	}

	// Preserve CPU cores for the host system by allowing only the cores that are not used by the VM
	if len(preserveCPUs) > 0 {
		system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "preserving CPU cores for host system")
		err := system.SetAllowedCPUs(preserveCPUs)
		if err != nil {
			return fmt.Errorf("error occurred while setting allowed CPUs: %v", err)
		}
	}

	// Check for GPU devices and attach if specified
	for _, deviceID := range machine.PCISet {
		for _, GPU := range GPUs {
			if GPU.VideoDevice.ID != deviceID {
				continue
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("GPU detected with ID %s primary=%t", GPU.VideoDevice.ID, GPU.Primary))

			// When GPU is primary, stop display manager and unbind virtual consoles and frame buffers before releasing the GPU to avoid issues with the display server and potential crashes
			if GPU.Primary {
				if displayManager.Name != "" {
					system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("stopping display manager %s", displayManager.Name))
					err := system.StopDisplayManager(displayManager)
					if err != nil {
						return fmt.Errorf("error occurred while stopping display manager: %v", err)
					}
				}

				if len(virtualConsoles) > 0 {
					for _, virtualConsole := range virtualConsoles {
						system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("unbinding virtual console %s", virtualConsole.Number))
						err := system.UnbindVirtualConsole(virtualConsole)
						if err != nil {
							return fmt.Errorf("error occurred while unbinding virtual console: %v", err)
						}
					}
				}

				if len(frameBuffers) > 0 {
					for _, frameBuffer := range frameBuffers {
						system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("unbinding frame buffer %s", frameBuffer.Name))
						err := system.UnbindFrameBuffer(frameBuffer)
						if err != nil {
							return fmt.Errorf("error occurred while unbinding frame buffer: %v", err)
						}
					}
				}
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("releasing GPU %s", GPU.VideoDevice.ID))
			err := system.ReleaseGPU(GPU)
			if err != nil {
				return fmt.Errorf("error occurred while releasing GPU %s: %v", GPU.VideoDevice.ID, err)
			}

			time.Sleep(time.Second)
		}
	}

	return nil
}

// StartBegin handles start begin qemu hook
func QemuStartBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook start begin for VM %s", machine.Name))
	machine.Status = system.STATUS_STARTING

	// Update the current virtual machine status in the state
	for _, vm := range stateTmp.VMs {
		if vm.Name == machine.Name {
			vm.Status = system.STATUS_STARTING
			break
		}
	}

	return nil
}

// StartedBegin handles started begin qemu hook
func QemuStartedBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook started begin for VM %s", machine.Name))
	machine.Status = system.STATUS_RUNNING

	// Update the current virtual machine status in the state
	for _, vm := range stateTmp.VMs {
		if vm.Name == machine.Name {
			vm.Status = system.STATUS_RUNNING
			break
		}
	}

	return nil
}

// StoppedEnd handles stopped end qemu hook
func QemuStoppedEnd(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook stopped end for VM %s", machine.Name))
	machine.Status = system.STATUS_STOPPING

	// Update the current virtual machine status in the state
	for _, vm := range stateTmp.VMs {
		if vm.Name == machine.Name {
			vm.Status = system.STATUS_STOPPING
			break
		}
	}

	return nil
}

// ReleaseEnd handles release end qemu hook
func QemuReleaseEnd(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook release end for VM %s", machine.Name))
	machine.Status = system.STATUS_SHUTOFF

	// Update the current virtual machine status in the state
	foundMachine := false
	for _, vm := range stateTmp.VMs {
		if vm.Name == machine.Name {
			vm.Status = system.STATUS_SHUTOFF
			foundMachine = true
			break
		}
	}

	// If the machine was not found in the state, skip processing
	if !foundMachine {
		return nil
	}

	// Extract necessary information from state to perform required actions
	VMs := stateTmp.VMs
	CPUs := stateTmp.CPUs
	GPUs := stateTmp.GPUs
	USBs := stateTmp.USBs
	displayManager := stateTmp.DisplayManager
	virtualConsoles := stateTmp.VirtualConsoles
	frameBuffers := stateTmp.FrameBuffers

	// Release GPU if it was attached
	// Check for GPU devices and detach if specified
	for _, deviceID := range machine.PCISet {
		for _, GPU := range GPUs {
			if GPU.VideoDevice.ID != deviceID {
				continue
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("GPU detected with ID %s primary=%t", GPU.VideoDevice.ID, GPU.Primary))

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("restoring GPU %s", GPU.VideoDevice.ID))
			err := system.RestoreGPU(GPU)
			if err != nil {
				if errors.Is(err, syscall.EBUSY) {
					system.WriteLog(system.LOG_WARNING, "qemu-hooks", fmt.Sprintf("device %s is busy and cannot be fully restored", GPU.VideoDevice.ID))
				} else {
					return fmt.Errorf("error occurred while restoring GPU %s: %v", GPU.VideoDevice.ID, err)
				}
			}

			// When GPU is primary, start display manager and bind virtual consoles and frame buffers after restoring the GPU to avoid issues with the display server and potential crashes
			if GPU.Primary {
				if len(frameBuffers) > 0 {
					for _, frameBuffer := range frameBuffers {
						system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("restoring frame buffer %s", frameBuffer.Name))
						err = system.BindFrameBuffer(frameBuffer)
						if err != nil {
							return fmt.Errorf("error occurred while binding frame buffer: %v", err)
						}
					}
				}

				if len(virtualConsoles) > 0 {
					for _, virtualConsole := range virtualConsoles {
						system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("restoring virtual console %s", virtualConsole.Name))
						err := system.BindVirtualConsoles(virtualConsole)
						if err != nil {
							return fmt.Errorf("error occurred while binding virtual console: %v", err)
						}
					}
				}

				if displayManager.Name != "" {
					system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("starting display manager %s", displayManager.Name))
					err := system.StartDisplayManager(displayManager)
					if err != nil {
						return fmt.Errorf("error occurred while starting display manager: %v", err)
					}
				}
			}
		}
	}

	// Count how many VMs are still running
	runningStatuses := []string{
		system.STATUS_PREPARING,
		system.STATUS_STARTING,
		system.STATUS_RUNNING,
	}

	runningVMs := 0
	for _, vm := range VMs {
		if slices.Contains(runningStatuses, vm.Status) {
			runningVMs++
		}
	}

	// Restore CPU scaling governor to original values
	// Perform action only when there is no longer any running VM
	if runningVMs == 0 {
		system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "restoring CPU scaling governor to original values")
		for _, CPU := range CPUs {
			err := system.SetCPUScalingGovernor(CPU)
			if err != nil {
				return fmt.Errorf("error occurred while restoring CPU scaling governor: %v", err)
			}
		}

		// Allow all cores again
		system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "allowing all CPU cores again")
		err := system.SetAllowedCPUs([]*system.CPU{})
		if err != nil {
			return fmt.Errorf("error occurred while setting allowed CPUs: %v", err)
		}
	}

	// Reconnect USB devices that were detached from the VM
	// Necessary because some devices may not be properly released from VM
	for _, USBDevice := range machine.USBSet {
		for _, USB := range USBs {
			if USB.ID != USBDevice {
				continue
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("restoring USB device %s", USB.ID))

			err := system.UnbindUSB(USB)
			if err != nil {
				return fmt.Errorf("error occurred while restoring USB device %s: %v", USB.ID, err)
			}

			err = system.BindUSB(USB)
			if err != nil {
				return fmt.Errorf("error occurred while restoring USB device %s: %v", USB.ID, err)
			}
		}
	}

	return nil
}

// MigrateBegin handles migrate begin qemu hook
func QemuMigrateBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook migrate begin for VM %s", machine.Name))
	return nil
}

// RestoreBegin handles restore begin qemu hook
func QemuRestoreBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook restore begin for VM %s", machine.Name))
	return nil
}

// ReconnectBegin handles reconnect begin qemu hook
func QemuReconnectBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook reconnect begin for VM %s", machine.Name))
	return nil
}

// AttachBegin handles attach begin qemu hook
func QemuAttachBegin(machine *system.VirtualMachine, stateTmp *state.State) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook attach begin for VM %s", machine.Name))
	return nil
}

package hooks

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/mateussouzaweb/libvirt-hooks/packages/state"
	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

// QemuPrepareBegin handles prepare begin qemu hook
func QemuPrepareBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook prepare begin for VM %s", machine.Name))

	// Read temporary state
	stateTmp, err := state.ReadTemporaryState()
	if err != nil {
		return err
	}

	// Add the current virtual machine to the state
	machine.Status = "running"
	stateTmp.VMs = append(stateTmp.VMs, machine)

	// Write updated state back to file to be used by release hook
	err = state.SaveTemporaryState(stateTmp)
	if err != nil {
		return fmt.Errorf("error occurred while writing state file: %v", err)
	}

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
		found := false
		for _, vCPUPin := range machine.Specs.CPUTune.VCPUPin {
			if vCPUPin.CPUSet == CPU.Number {
				found = true
				break
			}
		}
		if !found {
			preserveCPUs = append(preserveCPUs, CPU)
		}
	}

	// Preserve CPU cores for the host system by allowing only the cores that are not used by the VM
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "preserving CPU cores for host system")
	err = system.SetAllowedCPUs(preserveCPUs)
	if err != nil {
		return fmt.Errorf("error occurred while setting allowed CPUs: %v", err)
	}

	// Check for GPU devices and attach if specified
	for _, hostDev := range machine.Specs.Devices.Hostdevs {
		if hostDev.SubsysPCI == nil {
			continue
		}

		address := hostDev.SubsysPCI.Source.Address
		deviceID := fmt.Sprintf(
			"%04x:%02x:%02x.%x",
			*address.Domain,
			*address.Bus,
			*address.Slot,
			*address.Function,
		)

		for _, GPU := range GPUs {
			if GPU.VideoDevice.ID != deviceID {
				continue
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("GPU detected with ID %s [primary: %t]", GPU.VideoDevice.ID, GPU.Primary))

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
						err = system.UnbindFrameBuffer(frameBuffer)
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
func QemuStartBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook start begin for VM %s", machine.Name))
	return nil
}

// StartedBegin handles started begin qemu hook
func QemuStartedBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook started begin for VM %s", machine.Name))
	return nil
}

// StoppedEnd handles stopped end qemu hook
func QemuStoppedEnd(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook stopped end for VM %s", machine.Name))
	return nil
}

// ReleaseEnd handles release end qemu hook
func QemuReleaseEnd(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook release end for VM %s", machine.Name))

	// Read current state from file to get system information
	stateTmp, err := state.ReadTemporaryState()
	if err != nil {
		return err
	}

	// Remove the current virtual machine from the state
	for i, vm := range stateTmp.VMs {
		if vm.Name == machine.Name {
			stateTmp.VMs = append(stateTmp.VMs[:i], stateTmp.VMs[i+1:]...)
			break
		}
	}

	// Write updated state
	// When there is no more running VMs, the file will be removed
	err = state.SaveTemporaryState(stateTmp)
	if err != nil {
		return err
	}

	// Extract necessary information from state to perform required actions
	VMs := stateTmp.VMs
	CPUs := stateTmp.CPUs
	GPUs := stateTmp.GPUs
	displayManager := stateTmp.DisplayManager
	virtualConsoles := stateTmp.VirtualConsoles
	frameBuffers := stateTmp.FrameBuffers

	// Release GPU if it was attached
	// Check for GPU devices and detach if specified
	for _, hostDev := range machine.Specs.Devices.Hostdevs {
		if hostDev.SubsysPCI == nil {
			continue
		}

		address := hostDev.SubsysPCI.Source.Address
		deviceID := fmt.Sprintf(
			"%04x:%02x:%02x.%x",
			*address.Domain,
			*address.Bus,
			*address.Slot,
			*address.Function,
		)

		for _, GPU := range GPUs {
			if GPU.VideoDevice.ID != deviceID {
				continue
			}

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("GPU detected with ID %s [primary: %t]", GPU.VideoDevice.ID, GPU.Primary))

			system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("restoring GPU %s", GPU.VideoDevice.ID))
			err = system.RestoreGPU(GPU)
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

	// Restore CPU scaling governor to original values when the last VM has stopped
	if len(VMs) == 0 {
		system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "restoring CPU scaling governor to original values")
		for _, CPU := range CPUs {
			err := system.SetCPUScalingGovernor(CPU)
			if err != nil {
				return fmt.Errorf("error occurred while restoring CPU scaling governor: %v", err)
			}
		}

		// Allow all cores again
		system.WriteLog(system.LOG_NOTICE, "qemu-hooks", "allowing all CPU cores again")
		err = system.SetAllowedCPUs([]*system.CPU{})
		if err != nil {
			return fmt.Errorf("error occurred while setting allowed CPUs: %v", err)
		}
	}

	return nil
}

// MigrateBegin handles migrate begin qemu hook
func QemuMigrateBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook migrate begin for VM %s", machine.Name))
	return nil
}

// RestoreBegin handles restore begin qemu hook
func QemuRestoreBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook restore begin for VM %s", machine.Name))
	return nil
}

// ReconnectBegin handles reconnect begin qemu hook
func QemuReconnectBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook reconnect begin for VM %s", machine.Name))
	return nil
}

// AttachBegin handles attach begin qemu hook
func QemuAttachBegin(machine *system.VirtualMachine) error {
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("running hook attach begin for VM %s", machine.Name))
	return nil
}

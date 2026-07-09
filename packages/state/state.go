package state

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

// Location of the temporary state file
const STATE_FILE = "/tmp/qemu-state.json"

// State represents the current state of the system and virtual machines
type State struct {
	Populated       bool                     `json:"populated"`
	Devices         []*system.Device         `json:"devices"`
	CPUs            []*system.CPU            `json:"cpus"`
	GPUs            []*system.GPU            `json:"gpus"`
	USBs            []*system.USB            `json:"usbs"`
	VMs             []*system.VirtualMachine `json:"vms"`
	FrameBuffers    []*system.FrameBuffer    `json:"frameBuffers"`
	VirtualConsoles []*system.VirtualConsole `json:"virtualConsoles"`
	DisplayManager  *system.DisplayManager   `json:"displayManager"`
}

// NewState creates and populates a state with system information
func NewState() *State {
	return &State{
		Populated:       false,
		Devices:         make([]*system.Device, 0),
		CPUs:            make([]*system.CPU, 0),
		GPUs:            make([]*system.GPU, 0),
		USBs:            make([]*system.USB, 0),
		VMs:             make([]*system.VirtualMachine, 0),
		FrameBuffers:    make([]*system.FrameBuffer, 0),
		VirtualConsoles: make([]*system.VirtualConsole, 0),
		DisplayManager:  &system.DisplayManager{},
	}
}

// ReadState reads the state information from file
func ReadState(source string) (*State, error) {

	state := NewState()
	exists, err := system.FileExists(source)
	if err != nil {
		return state, fmt.Errorf("error checking state file: %w", err)
	} else if !exists {
		return state, nil
	}

	content, err := os.ReadFile(source)
	if err != nil {
		return state, fmt.Errorf("error reading state file: %w", err)
	}

	err = json.Unmarshal(content, state)
	if err != nil {
		return state, fmt.Errorf("error unmarshaling state file: %w", err)
	}

	return state, nil
}

// WriteState writes the state information to file
func WriteState(state *State, destination string) error {

	result, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error writing state: %w", err)
	}

	err = os.WriteFile(destination, result, 0666)
	if err != nil {
		return fmt.Errorf("error writing state file: %w", err)
	}

	return nil
}

// RemoveState removes the state file if it exists
func RemoveState(source string) error {

	exists, err := system.FileExists(source)
	if err != nil {
		return fmt.Errorf("error checking state file: %w", err)
	} else if !exists {
		return nil
	}

	err = os.Remove(source)
	if err != nil {
		return fmt.Errorf("error removing state file: %w", err)
	}

	return nil
}

// PopulateState populates a state with system information
func PopulateState(state *State) error {

	devices, err := system.GetDevices()
	if err != nil {
		return err
	} else {
		state.Devices = devices
	}

	CPUs, err := system.GetCPUs()
	if err != nil {
		return err
	} else {
		state.CPUs = CPUs
	}

	GPUs, err := system.GetGPUs()
	if err != nil {
		return err
	} else {
		state.GPUs = GPUs
	}

	for _, device := range state.Devices {
		if device.Type == system.TYPE_USB {
			USBs, err := system.GetUSBs(device)
			if err != nil {
				return err
			} else {
				state.USBs = append(state.USBs, USBs...)
			}
		}
	}

	frameBuffers, err := system.GetFrameBuffers()
	if err != nil {
		return err
	} else {
		state.FrameBuffers = frameBuffers
	}

	virtualConsoles, err := system.GetVirtualConsoles()
	if err != nil {
		return err
	} else {
		state.VirtualConsoles = virtualConsoles
	}

	displayManager, err := system.GetDisplayManager()
	if err != nil {
		return err
	} else {
		state.DisplayManager = displayManager
	}

	state.Populated = true
	return nil
}

// DumpState retrieves the current state and prints it as JSON
func DumpState() error {

	state := NewState()
	err := PopulateState(state)
	if err != nil {
		return fmt.Errorf("error populating state: %w", err)
	}

	result, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("error dumping state: %w", err)
	}

	fmt.Println(string(result))
	return nil
}

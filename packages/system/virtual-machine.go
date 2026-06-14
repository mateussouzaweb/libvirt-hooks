package system

import (
	"fmt"
	"io"
	"os"
	"strings"

	"libvirt.org/go/libvirtxml"
)

type Specs = libvirtxml.Domain

// VirtualMachine represents a virtual machine
type VirtualMachine struct {
	Name   string   `json:"name"`
	UUID   string   `json:"uuid"`
	Status string   `json:"status"`
	CPUSet []string `json:"cpuSet"`
	PCISet []string `json:"pciSet"`
	USBSet []string `json:"usbSet"`
	XML    string   `json:"-"`
	Specs  Specs    `json:"-"`
}

// Virtual machine status constants
const STATUS_PREPARING = "preparing"
const STATUS_STARTING = "starting"
const STATUS_RUNNING = "running"
const STATUS_STOPPING = "stopping"
const STATUS_SHUTOFF = "shutoff"
const STATUS_UNKNOWN = "unknown"

// ReadVirtualMachineXMLFromStdin reads data from stdin and returns its content
func ReadVirtualMachineXMLFromStdin() (string, error) {

	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("error reading stdin: %w", err)
	} else if (stat.Mode() & os.ModeCharDevice) == 0 {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("error reading stdin: %w", err)
		}
		return string(content), nil
	}

	return "", nil
}

// ReadVirtualMachineXMLFromFile reads the XML file and returns its content
func ReadVirtualMachineXMLFromFile(guestName string) (string, error) {

	xmlPath := fmt.Sprintf("/etc/libvirt/qemu/%s.xml", guestName)
	_, err := os.Stat(xmlPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("error occurred while checking VM file: %v", err)
	}
	if os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(xmlPath)
	if err != nil {
		return "", fmt.Errorf("error occurred while reading VM file: %v", err)
	}

	return string(content), nil
}

// ReadVirtualMachine reads virtual machine information from stdin or XML file
func ReadVirtualMachine(guestName string, fromStdin bool) (*VirtualMachine, error) {

	machine := &VirtualMachine{
		Name:   "",
		UUID:   "",
		Status: STATUS_UNKNOWN,
		CPUSet: []string{},
		PCISet: []string{},
		USBSet: []string{},
		XML:    "",
		Specs:  Specs{},
	}

	// Try to read XML specs from stdin if enabled
	if machine.XML == "" && fromStdin {
		guestXML, err := ReadVirtualMachineXMLFromStdin()
		if err != nil {
			return machine, fmt.Errorf("error occurred while reading virtual machine XML from stdin: %v", err)
		}
		machine.XML = guestXML
	}

	// If XML is still empty, try to read from file
	if machine.XML == "" {
		guestXML, err := ReadVirtualMachineXMLFromFile(guestName)
		if err != nil {
			return machine, fmt.Errorf("error occurred while reading virtual machine XML from file: %v", err)
		}
		machine.XML = guestXML
	}

	// If XML is still empty, return an error
	if machine.XML == "" {
		return machine, fmt.Errorf("no virtual machine XML found for guest '%s'", guestName)
	}

	// Read specs from XML
	// @see https://libvirt.org/formatdomain.html
	if err := machine.Specs.Unmarshal(machine.XML); err != nil {
		return machine, fmt.Errorf("error occurred while parsing virtual machine XML: %v", err)
	}

	machine.Name = machine.Specs.Name
	machine.UUID = machine.Specs.UUID

	// Read CPU set
	if machine.Specs.CPUTune != nil {
		for _, vCPUPin := range machine.Specs.CPUTune.VCPUPin {
			if vCPUPin.CPUSet != "" {
				machine.CPUSet = append(machine.CPUSet, vCPUPin.CPUSet)
			}
		}
	}

	// Read PCI devices
	if machine.Specs.Devices != nil {
		for _, hostDev := range machine.Specs.Devices.Hostdevs {
			if hostDev.SubsysPCI == nil {
				continue
			}
			if hostDev.SubsysPCI.Source == nil {
				continue
			}
			if hostDev.SubsysPCI.Source.Address == nil {
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

			machine.PCISet = append(machine.PCISet, deviceID)
		}
	}

	// Read USB devices
	if machine.Specs.Devices != nil {
		for _, hostDev := range machine.Specs.Devices.Hostdevs {
			if hostDev.SubsysUSB == nil {
				continue
			}
			if hostDev.SubsysUSB.Source == nil {
				continue
			}
			if hostDev.SubsysUSB.Source.Vendor == nil {
				continue
			}
			if hostDev.SubsysUSB.Source.Product == nil {
				continue
			}

			vendor := *hostDev.SubsysUSB.Source.Vendor
			product := *hostDev.SubsysUSB.Source.Product
			deviceID := fmt.Sprintf("%s:%s", vendor.ID, product.ID)
			deviceID = strings.ReplaceAll(deviceID, "0x", "")

			machine.USBSet = append(machine.USBSet, deviceID)
		}
	}

	return machine, nil
}

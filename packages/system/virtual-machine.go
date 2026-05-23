package system

import (
	"fmt"
	"io"
	"os"

	"libvirt.org/go/libvirtxml"
)

type Specs = libvirtxml.Domain

// VirtualMachine represents a virtual machine
type VirtualMachine struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Status string `json:"status"`
	XML    string `json:"-"`
	Specs  Specs  `json:"-"`
}

// ReadVirtualMachineXMLFromStdin reads data from stdin and returns its content
func ReadVirtualMachineXMLFromStdin() (string, error) {

	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("Error reading stdin: %v\n", err)
	} else if (stat.Mode() & os.ModeCharDevice) == 0 {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("Error reading stdin: %v\n", err)
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
		Status: "",
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

	return machine, nil
}

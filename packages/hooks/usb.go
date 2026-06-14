package hooks

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/mateussouzaweb/libvirt-hooks/packages/state"
	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

// USBHandleAction processes USB events to perform actions on QEMU VMs
func USBHandleAction(action string, busNumber string, devNumber string, product string) error {

	// Validate action
	if action != "add" && action != "remove" {
		return fmt.Errorf("invalid action for usb command: %s", action)
	}

	// Check for required USB information
	if busNumber == "" || devNumber == "" || product == "" {
		return fmt.Errorf("missing required USB information for usb command")
	}

	// Extract USB ID reference
	productParts := strings.Split(product, "/")
	if len(productParts) < 2 {
		return fmt.Errorf("invalid product format: %s", product)
	}

	// Use hexadecimal strings with zero-pad to result in 4 chars
	var vendorNumber, productNumber int64
	fmt.Sscanf(productParts[0], "%x", &vendorNumber)
	fmt.Sscanf(productParts[1], "%x", &productNumber)

	vendorID := fmt.Sprintf("%04x", vendorNumber)
	productID := fmt.Sprintf("%04x", productNumber)

	// Read temporary state
	stateTmp, err := state.ReadState(state.STATE_FILE)
	if err != nil {
		return fmt.Errorf("error reading temporary state: %w", err)
	}

	// If there is no state data or no tracked VMs, skip processing
	if !stateTmp.Populated || len(stateTmp.VMs) == 0 {
		return nil
	}

	// Detect PCI device that is parent of the USB device
	PCIDevice := ""
	for _, usb := range stateTmp.USBs {
		if usb.BusNumber == busNumber && usb.DevNumber == devNumber {
			PCIDevice = usb.Parent
			break
		}
		if usb.Vendor == vendorID && usb.Product == productID {
			PCIDevice = usb.Parent
			break
		}
	}

	// Check if there is a running VM can be targeted for USB actions
	// When the VM passthrough the PCI device, we can skip udev actions
	targetVM := &system.VirtualMachine{}
	for _, vm := range stateTmp.VMs {
		if vm.Status != system.STATUS_RUNNING {
			continue
		}
		if !slices.Contains(vm.PCISet, PCIDevice) {
			targetVM = vm
			break
		}
	}

	// Is there is no valid target VM, skip processing
	if targetVM.Name == "" {
		return nil
	}

	// Log the USB event and intended action
	target := targetVM.Name
	details := fmt.Sprintf("action=%s bus=%s dev=%s product=%s", action, busNumber, devNumber, product)
	system.WriteLog(system.LOG_NOTICE, "qemu-hooks", fmt.Sprintf("performing usb action for VM %s %s", target, details))

	// Determine libvirt command
	var command string
	switch action {
	case "add":
		command = "attach-device"
	case "remove":
		command = "detach-device"
	}

	// Prepare XML for libvirt device
	virshXML := fmt.Sprintf(`<hostdev mode="subsystem" type="usb" managed="yes">
  <source startupPolicy="optional">
    <vendor id="0x%s"/>
    <product id="0x%s"/>
  </source>
</hostdev>`, vendorID, productID)

	// Run command on libvirt with bash wrapper
	// Necessary to ensure correct profile environment
	virshCmd := fmt.Sprintf("virsh %s %s --live /dev/stdin", command, target)
	bashCmd := exec.Command("/bin/bash", "-l", "-c", virshCmd)
	bashCmd.Stdin = strings.NewReader(virshXML)

	if err := bashCmd.Run(); err != nil {
		return fmt.Errorf("error executing virsh command: %w", err)
	}

	return nil
}

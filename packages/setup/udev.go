package setup

import (
	"bytes"
	"embed"
	"fmt"
	"os"

	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

//go:embed *.rules
var rulesFS embed.FS

// ReloadUdevRules reloads the udev rules to apply any changes made to the rules files
func ReloadUdevRules() error {

	// NOTE: command requires root privileges
	_, err := system.RunCommand("sudo", "udevadm", "control", "--reload-rules")
	if err != nil {
		return fmt.Errorf("error reloading udev rules: %w", err)
	}

	return nil
}

// InstallUdevRules sets up udev rules for handling USB device events with libvirt hooks
func InstallUdevRules() error {

	// Get the path to the current executable to use in udev rules
	scriptPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting current script path: %w", err)
	}

	// Create udev rules directory if it doesn't exist
	err = os.MkdirAll("/etc/udev/rules.d", 0755)
	if err != nil {
		return fmt.Errorf("error creating udev rules directory: %w", err)
	}

	// Read the embedded udev rules template
	rulesPath := "/etc/udev/rules.d/90-libvirt-usb.rules"
	rulesContent, err := rulesFS.ReadFile("usb.rules")
	if err != nil {
		return fmt.Errorf("error reading embedded udev rules: %w", err)
	}

	// Replace placeholder in rules with actual script path
	rulesContent = bytes.ReplaceAll(
		rulesContent,
		[]byte("{{SCRIPT_PATH}}"),
		[]byte(scriptPath),
	)

	// Write udev rules to the appropriate location
	err = os.WriteFile(rulesPath, rulesContent, 0644)
	if err != nil {
		return fmt.Errorf("error writing udev rules at %s: %w", rulesPath, err)
	}

	// Reload udev rules to apply changes
	return ReloadUdevRules()
}

// UninstallUdevRules removes the udev rules for handling USB device events with libvirt hooks
func UninstallUdevRules() error {

	// Remove udev rules file
	rulesPath := "/etc/udev/rules.d/90-libvirt-usb.rules"
	err := os.Remove(rulesPath)
	if err != nil {
		return fmt.Errorf("error removing udev rules at %s: %w", rulesPath, err)
	}

	// Reload udev rules to apply changes
	return ReloadUdevRules()
}

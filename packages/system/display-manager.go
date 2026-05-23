package system

import (
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// DisplayManager represents the display manager running on the system
type DisplayManager struct {
	Name    string `json:"name"`
	Service string `json:"service"`
}

// GetDisplayManager checks and returns the system display manager
func GetDisplayManager() (*DisplayManager, error) {

	// List of common display managers to check for
	displayManagers := []string{
		"sddm",
		"gdm",
		"lightdm",
		"lxdm",
		"xdm",
		"mdm",
		"display-manager",
	}

	detected := &DisplayManager{}
	for _, manager := range displayManagers {

		// Check if the display manager service is available
		service := fmt.Sprintf("%s.service", manager)
		result, err := RunCommand("systemctl", "show", "-p", "LoadState", service)
		if err != nil {
			return detected, fmt.Errorf("error checking display manager: %w", err)
		} else if result == "LoadState=not-found" {
			continue
		}

		detected.Name = manager
		detected.Service = service
		return detected, nil
	}

	return detected, nil
}

// IsDisplayManagerActive checks if the display manager is currently active on the system
func IsDisplayManagerActive(displayManager *DisplayManager) (bool, error) {

	if displayManager.Service == "" {
		return false, nil
	}

	// Check if the display manager service is active
	// Is-active exits non-zero when inactive — that's expected, not an error
	result, err := RunCommand("systemctl", "is-active", displayManager.Service)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 3 {
			return false, nil // service is inactive, not a real error
		}

		return false, fmt.Errorf("error checking display manager status: %w", err)
	} else if result == "active" {
		return true, nil
	}

	return false, nil
}

// StopDisplayManager stops the running display manager on system
func StopDisplayManager(displayManager *DisplayManager) error {

	if displayManager.Service == "" {
		return nil
	}

	// Check if the display manager service is active
	result, err := IsDisplayManagerActive(displayManager)
	if err != nil {
		return err
	} else if !result {
		return nil
	}

	// Stop the display manager service and wait for it to stop
	_, err = RunCommand("systemctl", "stop", displayManager.Service)
	if err != nil {
		return fmt.Errorf("error stopping display manager: %w", err)
	}

	// Wait for the display manager to stop before proceeding
	for {
		time.Sleep(time.Second)
		result, err := IsDisplayManagerActive(displayManager)
		if err != nil {
			return err
		} else if !result {
			break
		}
	}

	return nil
}

// StartDisplayManager starts the previously active display manager
func StartDisplayManager(displayManager *DisplayManager) error {

	if displayManager.Service == "" {
		return nil
	}

	// Check if the display manager service is active
	result, err := IsDisplayManagerActive(displayManager)
	if err != nil {
		return err
	} else if result {
		return nil
	}

	// Start the display manager service again
	_, err = RunCommand("systemctl", "start", displayManager.Service)
	if err != nil {
		return fmt.Errorf("error starting display manager: %w", err)
	}

	// Wait one second before proceeding to ensure the display manager has time to start
	time.Sleep(time.Second)

	return nil
}

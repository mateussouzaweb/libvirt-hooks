package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mateussouzaweb/libvirt-hooks/packages/hooks"
	"github.com/mateussouzaweb/libvirt-hooks/packages/setup"
	"github.com/mateussouzaweb/libvirt-hooks/packages/state"
	"github.com/mateussouzaweb/libvirt-hooks/packages/system"
)

// PrintHelp prints usage information for the command-line tool
func PrintHelp() error {

	scriptPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("error getting current script path: %w", err)
	}

	fmt.Printf("Usage: %s <command> [args]\n", scriptPath)
	fmt.Printf("Commands:\n")
	fmt.Printf("  help - Show this help message.\n")
	fmt.Printf("  state - Read and display system state.\n")
	fmt.Printf("  install - Install udev rules for USB handling.\n")
	fmt.Printf("  uninstall - Uninstall udev rules for USB handling.\n")
	fmt.Printf("  usb - Handle udev USB actions on QEMU.\n")
	fmt.Printf("  <guest> <event> <state> - Handle libvirt hooks for QEMU events.\n")

	return nil
}

// HandleCommand processes arguments and executes the appropriate actions
func HandleCommand(args []string) error {

	// If no arguments are provided, print help
	if len(args) == 0 {
		return PrintHelp()
	}

	command := args[0]

	// Help command to print usage information
	if command == "help" || command == "--help" || command == "-h" {
		return PrintHelp()
	}

	// State command to retrieve current state of the system and VMs
	if command == "state" {
		return state.DumpState()
	}

	// Install command
	if command == "install" {
		return setup.InstallUdevRules()
	}

	// Uninstall command
	if command == "uninstall" {
		return setup.UninstallUdevRules()
	}

	// USB command to add and remove USB devices on QEMU VMs
	// This command is triggered by udev rules
	// Udev uses environment variables to pass information
	if command == "usb" {
		action := os.Getenv("ACTION")
		busNumber := os.Getenv("BUSNUM")
		devNumber := os.Getenv("DEVNUM")
		product := os.Getenv("PRODUCT")
		return hooks.USBHandleAction(action, busNumber, devNumber, product)
	}

	// QEMU hooks have multiple commands
	// Map all know commands to their respective handlers
	if len(args) > 3 {
		guestName := args[0]
		guestEvent := args[1]
		guestState := args[2]

		// Read virtual machine information from stdin or XML file
		machine, err := system.ReadVirtualMachine(guestName, true)
		if err != nil {
			return fmt.Errorf("Error reading virtual machine info: %w", err)
		}

		// Handle known QEMU events and states
		if guestEvent == "prepare" && guestState == "begin" {
			return hooks.QemuPrepareBegin(machine)
		}
		if guestEvent == "start" && guestState == "begin" {
			return hooks.QemuStartBegin(machine)
		}
		if guestEvent == "started" && guestState == "begin" {
			return hooks.QemuStartedBegin(machine)
		}
		if guestEvent == "stopped" && guestState == "end" {
			return hooks.QemuStoppedEnd(machine)
		}
		if guestEvent == "release" && guestState == "end" {
			return hooks.QemuReleaseEnd(machine)
		}
		if guestEvent == "migrate" && guestState == "begin" {
			return hooks.QemuMigrateBegin(machine)
		}
		if guestEvent == "restore" && guestState == "begin" {
			return hooks.QemuRestoreBegin(machine)
		}
		if guestEvent == "reconnect" && guestState == "begin" {
			return hooks.QemuReconnectBegin(machine)
		}
		if guestEvent == "attach" && guestState == "begin" {
			return hooks.QemuAttachBegin(machine)
		}
	}

	// Unknown command
	return fmt.Errorf("Unknown command: %s", command)
}

// Main command
func main() {

	// Exit with proper code
	exitCode := 0
	defer os.Exit(exitCode)

	// Graceful init and shutdown support
	exit := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	// Capture exit (CTRL-C)
	go func() {
		<-exit
		done <- true
	}()

	// Run command
	go func() {
		if err := HandleCommand(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			system.WriteLog(system.LOG_ERROR, "qemu-hooks", err.Error())
			exitCode = 1
		}

		done <- true
	}()

	// Wait for completion or exit signal
	<-done
}

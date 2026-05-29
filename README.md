# Libvirt Hooks for KVM / QEMU

Custom scripts for KVM / QEMU based on [libvirt hooks](https://libvirt.org/hooks.html).

## Overview

You may already know KVM / QEMU with libvirt and how hooks can be used to run more powerful and advanced virtual machines.

This project is a set of hooks for the missing pieces of libvirt for homelab and desktop users, with deeper automation and action by parsing virtual machine details and acting where libvirt doesn't touch. With this script, you may not need to set up hooks for your desktop; just run the VM and let the script do the hard work for you automatically.

## Features

**- Real hook automation**\
Everything is fully automated. Since the script can understand aspects of your environment, it can detect additional actions to perform for you when running virtual machines with KVM / QEMU.

**- Handling for main GPU passthrough**\
When you have only one GPU in your system, the hook will automatically free the GPU for VFIO usage by stopping the current display manager and unbinding virtual consoles or framebuffers. It also manages the resizable BAR and releases GPU drivers when starting and stopping the virtual machine.

**- Supports secondary GPU passthrough**\
The script also works with systems based on multiple GPUs.

**- CPU core isolation**\
Automatically detects which CPU cores should be used by the virtual machine and by the host system in order to isolate cores and increase performance.

**- CPU scaling governor**\
Enable additional performance by setting the correct scaling governor when virtual machines are powered on.

**- USB passthrough**\
Frees USB devices for passthrough, and by combining it with udev rules, automatically passes through USB devices to a running virtual machine for easier access.

**- Environment debugging**\
Includes commands to inspect the system and provide highly detailed information such as PCI devices, CPU, GPUs, USBs, Display Manager, Virtual Consoles, Framebuffers, etc.

**- Consistent logging**\
All performed actions are logged to the host message bus for easier debugging. Messages can be read with the `dmesg` command along with other system actions.

**- Extra scripting support**\
You can use this hook for most of the automation and combine with smaller custom ones to perform additional tasks if needed.

## Installation

Go to the project RELEASES page and download the latest version of the script for your Linux architecture:

```bash
# Download compiled binary
wget https://github.com/mateussouzaweb/libvirt-hooks/releases

# Move binary to correct location
sudo mv $FILE /etc/libvirt/hooks/qemu
sudo chmod +x /etc/libvirt/hooks/qemu

# Install udev rules
sudo /etc/libvirt/hooks/qemu install
```

**That is it!**\
Now configure your virtual machine details on KVM / QEMU and let this script automate actions for you, such as passing through your GPU or setting the CPU scaling governor.

## Extra Scripts

If you need to perform additional actions with QEMU hooks in Libvirt, you should put your custom scripts into the ``/etc/libvirt/hooks/qemu.d/`` folder.
This folder is already officially supported by libvirt and supports multiple scripts. For example:

- ``/etc/libvirt/hooks/qemu.d/start.sh`` - script to run before VMs start.
- ``/etc/libvirt/hooks/qemu.d/stop.sh`` - script to run after VMs have been stopped.

See the [official documentation](https://libvirt.org/hooks.html) for more details.

## Building

This project was developed in Go and provides a single binary file to handle all aspects of desktop virtualization for you.

If you want to manually build the project, there is a build script here too. Simply run the script and it will build binaries for both AMD64 and ARM64:

```bash
go run build.go
```

Then install the binary to the target destination:

```bash
# Move binary
chmod +x bin/qemu-$(ARCH); \
sudo mv bin/qemu-$(ARCH) /etc/libvirt/hooks/qemu

# Install udev rules
sudo /etc/libvirt/hooks/qemu install
```

## Developing & Debugging

If you want to increase the features of the script or debug the output, here are a few tips:

```bash
# Run as sudo
sudo su
cd /etc/libvirt/hooks/

# Basic commands
./qemu help
./qemu install
./qemu uninstall

# Environment debugging
./qemu state | jq .

# Udev USB commands
# Pass details as environment variables like udev does
ACTION="add" BUSNUM="" DEVNUM="" PRODUCT="" ./qemu usb

# Libvirt hooks for VM actions
./qemu $VM prepare begin -
./qemu $VM release end - 

# Check system logs
sudo dmesg
```

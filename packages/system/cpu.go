package system

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CPU represents a CPU core on the system
type CPU struct {
	Number          string `json:"number"`
	Path            string `json:"path"`
	ScalingGovernor string `json:"scalingGovernor"`
}

// GetCPUs returns a list of CPU cores available on the system
func GetCPUs() ([]*CPU, error) {

	CPUs := make([]*CPU, 0)

	// Search for CPUs cores available on the system
	results, err := FindSysFSFolders("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil {
		return CPUs, fmt.Errorf("error searching for CPUs: %w", err)
	}

	// Transform each detected result
	for _, CPUPath := range results {
		CPUNumber := strings.TrimPrefix(filepath.Base(CPUPath), "cpu")
		CPUs = append(CPUs, &CPU{
			Number:          CPUNumber,
			Path:            CPUPath,
			ScalingGovernor: "",
		})
	}

	// Read scaling governor off each CPU
	for _, CPU := range CPUs {
		err := ReadCPUScalingGovernor(CPU)
		if err != nil {
			return CPUs, err
		}
	}

	// Sort CPU cores by number
	sort.Slice(CPUs, func(i, j int) bool {
		iNumber, _ := strconv.Atoi(CPUs[i].Number)
		jNumber, _ := strconv.Atoi(CPUs[j].Number)
		return iNumber < jNumber
	})

	return CPUs, nil
}

// ReadCPUScalingGovernor checks for the scaling governor for a given CPU and set its value if found
func ReadCPUScalingGovernor(CPU *CPU) error {

	if CPU.Path == "" {
		return nil
	}

	scalingGovernorPath := fmt.Sprintf("%s/cpufreq/scaling_governor", CPU.Path)
	scalingGovernorValue, err := ReadSysFSValue(scalingGovernorPath)
	if err != nil {
		return fmt.Errorf("error reading scaling governor for CPU %s: %w", CPU.Number, err)
	}

	CPU.ScalingGovernor = scalingGovernorValue
	return nil
}

// SetCPUScalingGovernor sets scaling governor to defined mode on given CPU
func SetCPUScalingGovernor(CPU *CPU) error {

	if CPU.Path == "" {
		return nil
	}

	scalingGovernorPath := fmt.Sprintf("%s/cpufreq/scaling_governor", CPU.Path)
	err := WriteSysFSValue(scalingGovernorPath, CPU.ScalingGovernor)
	if err != nil {
		return fmt.Errorf("error writing scaling governor for CPU %s: %w", CPU.Number, err)
	}

	return nil
}

// SetAllowedCPUs allows the usage of specific CPU cores by pinning them on system
// This list should be used to allow cores for the host system while avoiding their use for the VM
// Passing an empty list will allow the usage of all CPU cores again
func SetAllowedCPUs(CPUs []*CPU) error {

	cores := make([]string, len(CPUs))
	for i, CPU := range CPUs {
		cores[i] = CPU.Number
	}

	allowedCPUs := fmt.Sprintf("AllowedCPUs=%s", strings.Join(cores, ","))
	runtimes := []string{
		"user.slice",
		"system.slice",
		"init.scope",
	}

	for _, runtime := range runtimes {
		_, err := RunCommand("systemctl", "set-property", "--runtime", "--", runtime, allowedCPUs)
		if err != nil {
			return fmt.Errorf("error setting CPU pinning for %s: %w", runtime, err)
		}
	}

	return nil
}

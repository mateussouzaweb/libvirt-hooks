package system

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// VirtualConsole represents a virtual console on the system
type VirtualConsole struct {
	Number string `json:"number"`
	Name   string `json:"name"`
	Path   string `json:"path"`
}

// GetVirtualConsoles returns a list of virtual consoles on the system
func GetVirtualConsoles() ([]*VirtualConsole, error) {

	consoles := make([]*VirtualConsole, 0)

	// Search for virtual consoles available on the system
	pattern := "/sys/class/vtconsole/vtcon[0-9]*"
	results, err := FindSysFSFolders(pattern)
	if err != nil {
		return consoles, fmt.Errorf("error searching for virtual consoles: %w", err)
	}

	// Transform each detected result
	for _, consolePath := range results {
		consoleNamePath := fmt.Sprintf("%s/name", consolePath)
		consoleNameValue, err := ReadSysFSValue(consoleNamePath)
		if err != nil {
			return nil, fmt.Errorf("error reading virtual console name for: %w", err)
		} else if !strings.Contains(consoleNameValue, "frame buffer") {
			continue
		}

		consoleNumber := filepath.Base(consolePath)
		consoleNumber = strings.TrimPrefix(consoleNumber, "vtcon")
		consoles = append(consoles, &VirtualConsole{
			Number: consoleNumber,
			Name:   consoleNameValue,
			Path:   consolePath,
		})
	}

	// Sort virtual consoles by number
	sort.Slice(consoles, func(i int, j int) bool {
		iNumber, _ := strconv.Atoi(consoles[i].Number)
		jNumber, _ := strconv.Atoi(consoles[j].Number)
		return iNumber < jNumber
	})

	return consoles, nil
}

// UnbindVirtualConsoles unbinds virtual consoles on the system
func UnbindVirtualConsole(virtualConsole *VirtualConsole) error {

	if virtualConsole.Path == "" {
		return nil
	}

	// Unbind console by writing "0" to its bind file
	bindPath := fmt.Sprintf("%s/bind", virtualConsole.Path)
	err := WriteSysFSValue(bindPath, "0")
	if err != nil {
		return fmt.Errorf("error unbinding virtual console %s: %w", virtualConsole.Name, err)
	}

	return nil
}

// BindVirtualConsoles binds virtual consoles on the system
func BindVirtualConsoles(virtualConsole *VirtualConsole) error {

	if virtualConsole.Path == "" {
		return nil
	}

	// Bind console by writing "1" to its bind file
	bindPath := fmt.Sprintf("%s/bind", virtualConsole.Path)
	err := WriteSysFSValue(bindPath, "1")
	if err != nil {
		return fmt.Errorf("error binding virtual console %s: %w", virtualConsole.Name, err)
	}

	return nil
}

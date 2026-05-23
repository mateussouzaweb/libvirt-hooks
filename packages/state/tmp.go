package state

import (
	"fmt"
	"os"
)

// Location of the temporary state file
const TMP_STATE_FILE = "/tmp/qemu-state.json"

// ReadTemporaryState reads the temporary state file and returns its content as a State struct
func ReadTemporaryState() (*State, error) {

	// Read current state from file to get system information
	state, err := ReadState(TMP_STATE_FILE)
	if err != nil {
		return state, fmt.Errorf("error occurred while reading state file: %v", err)
	}

	// Populate state with system information if not already populated
	if len(state.CPUs) == 0 {
		err := PopulateState(state)
		if err != nil {
			return state, fmt.Errorf("error occurred while populating state: %v", err)
		}
	}

	return state, nil
}

// SaveTemporaryState saves the state information to the temporary state file
func SaveTemporaryState(state *State) error {

	// When no more VMs are running, remove the state file
	remainingVMs := len(state.VMs)
	if remainingVMs == 0 {
		err := os.Remove(TMP_STATE_FILE)
		if err != nil {
			return fmt.Errorf("error occurred while removing state file: %v", err)
		}
	}

	// Write updated state back to file to be used by release hook
	err := WriteState(state, TMP_STATE_FILE)
	if err != nil {
		return fmt.Errorf("error occurred while writing state file: %v", err)
	}

	return nil
}

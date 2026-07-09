package system

import (
	"fmt"
	"os"
)

// FileExists check if file exists and is not a directory
func FileExists(path string) (bool, error) {

	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("error checking file at %s: %w", path, err)
	} else if os.IsNotExist(err) {
		return false, nil
	} else if info.IsDir() {
		return false, nil
	}

	return true, nil
}

// FolderExists check if folder exists and is a directory
func FolderExists(path string) (bool, error) {

	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("error checking folder at %s: %w", path, err)
	} else if os.IsNotExist(err) {
		return false, nil
	} else if !info.IsDir() {
		return false, nil
	}

	return true, nil
}

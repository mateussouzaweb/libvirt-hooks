package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindSysFSFolders search for folder results based on given pattern
func FindSysFSFolders(pattern string) ([]string, error) {

	results := []string{}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return results, fmt.Errorf("error searching for folders: %w", err)
	}

	// Check on each detected match
	for _, path := range matches {
		pathInfo, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return results, fmt.Errorf("error checking for folder at %s: %w", pathInfo, err)
		} else if os.IsNotExist(err) {
			continue
		} else if !pathInfo.IsDir() {
			continue
		}

		results = append(results, path)
	}

	return results, nil
}

// FindSysFSFiles search for file results based on given pattern
func FindSysFSFiles(pattern string) ([]string, error) {

	results := []string{}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return results, fmt.Errorf("error searching for files: %w", err)
	}

	// Check on each detected match
	// We accept regular files and symlinks
	for _, path := range matches {
		pathInfo, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return results, fmt.Errorf("error checking for file at %s: %w", pathInfo, err)
		} else if os.IsNotExist(err) {
			continue
		} else if pathInfo.IsDir() {
			continue
		}

		results = append(results, path)
	}

	return results, nil
}

// ReadSysFSValue reads the value of sysfs content
func ReadSysFSValue(path string) (string, error) {

	pathInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("error checking file at %s: %w", path, err)
	} else if os.IsNotExist(err) {
		return "", fmt.Errorf("file not found at %s", path)
	} else if pathInfo.IsDir() {
		return "", fmt.Errorf("path is not a valid file at %s", path)
	}

	value, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading file value at %s: %w", path, err)
	}

	return strings.TrimSpace(string(value)), nil
}

// WriteSysFSValue writes the given value on sysfs file
func WriteSysFSValue(path string, value string) error {

	pathInfo, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error checking file at %s: %w", path, err)
	} else if os.IsNotExist(err) {
		return fmt.Errorf("file not found at %s", path)
	} else if pathInfo.IsDir() {
		return fmt.Errorf("path is not a valid file at %s", path)
	}

	err = os.WriteFile(path, []byte(value+"\n"), pathInfo.Mode())
	if err != nil {
		return fmt.Errorf("error writing value at %s: %w", path, err)
	}

	return nil
}

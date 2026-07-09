package system

import (
	"fmt"
	"os"
)

// FileExist check if file exist and is not a directory
func FileExist(path string) (bool, error) {

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

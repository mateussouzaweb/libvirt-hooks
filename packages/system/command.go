package system

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// RunCommand runs an system command and returns its stdout output
func RunCommand(name string, args ...string) (string, error) {
	var out bytes.Buffer

	cmd := exec.Command(name, args...)
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	return strings.TrimRight(out.String(), "\n"), err
}

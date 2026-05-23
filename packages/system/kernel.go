package system

import (
	"errors"
	"fmt"
	"os"
)

// Log levels matching kernel log levels (syslog priority)
const LOG_EMERGENCY = 0 // System is unusable
const LOG_ALERT = 1     // Action must be taken immediately
const LOG_CRITCIAL = 2  // Critical conditions
const LOG_ERROR = 3     // Error conditions
const LOG_WARNING = 4   // Warning conditions
const LOG_NOTICE = 5    // Normal but significant condition
const LOG_INFO = 6      // Informational
const LOG_DEBUG = 7     // Debug-level messages

// WriteLog append message to dmesg
func WriteLog(level int, namespace string, message string) error {

	// Open kernel log message file
	file, err := os.OpenFile("/dev/kmsg", os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/kmsg: %w", err)
	}

	// Make sure to close file on finish
	defer func() {
		errors.Join(err, file.Close())
	}()

	// Make sure that level is correct
	if level < LOG_EMERGENCY || level > LOG_DEBUG {
		level = LOG_INFO
	}

	// Write message to logger
	line := fmt.Sprintf("<%d>%s: %s\n", level, namespace, message)
	_, err = file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write to /dev/kmsg: %w", err)
	}

	return err
}

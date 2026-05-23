package system

import (
	"fmt"
	"path/filepath"
)

// FrameBuffer represents a framebuffer device on the system
type FrameBuffer struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetFrameBuffers return the framebuffer devices available on the system
func GetFrameBuffers() ([]*FrameBuffer, error) {

	frameBuffers := make([]*FrameBuffer, 0)

	// Search for framebuffers available on the system
	results, err := FindSysFSFolders("/sys/bus/platform/drivers/*-framebuffer")
	if err != nil {
		return frameBuffers, fmt.Errorf("error searching for framebuffers: %w", err)
	}

	for _, frameBufferPath := range results {

		frameBufferName := filepath.Base(frameBufferPath)
		boundedPattern := fmt.Sprintf("%s/%s.[0-9]+", frameBufferPath, frameBufferName)
		boundedResults, err := FindSysFSFiles(boundedPattern)
		if err != nil {
			return frameBuffers, fmt.Errorf("error searching for framebuffers: %w", err)
		}
		if len(boundedResults) == 0 {
			continue
		}

		frameBuffer := &FrameBuffer{
			Path: frameBufferPath,
			Name: boundedResults[0],
		}

		frameBuffers = append(frameBuffers, frameBuffer)
	}

	return frameBuffers, nil
}

// UnbindFrameBuffer unbinds framebuffer on the system
func UnbindFrameBuffer(frameBuffer *FrameBuffer) error {

	if frameBuffer.Path == "" || frameBuffer.Name == "" {
		return nil
	}

	unbindPath := fmt.Sprintf("%s/unbind", frameBuffer.Path)
	err := WriteSysFSValue(unbindPath, frameBuffer.Name)
	if err != nil {
		return fmt.Errorf("Error unbinding framebuffer: %v\n", err)
	}

	return nil
}

// BindFrameBuffer binds framebuffer on the system
func BindFrameBuffer(frameBuffer *FrameBuffer) error {

	if frameBuffer.Path == "" || frameBuffer.Name == "" {
		return nil
	}

	bindPath := fmt.Sprintf("%s/bind", frameBuffer.Path)
	err := WriteSysFSValue(bindPath, frameBuffer.Name)
	if err != nil {
		return fmt.Errorf("Error binding framebuffer: %v\n", err)
	}

	return nil
}

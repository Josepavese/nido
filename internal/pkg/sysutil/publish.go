package sysutil

import "fmt"

// PublishFile atomically moves a staged file on the destination filesystem,
// refusing to overwrite user data or a competing writer's artifact.
func PublishFile(source, destination string) error {
	if err := publishFile(source, destination); err != nil {
		return fmt.Errorf("publish without overwriting %s: %w", destination, err)
	}
	return nil
}

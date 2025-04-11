package internal

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
)

var (
	homeDir string
	once    sync.Once
)

// initHomeDir initializes the homeDir variable.
func initHomeDir() {
	usr, err := user.Current()
	if err != nil {
		// Fallback to using the HOME environment variable
		homeDir = os.Getenv("HOME")
	} else {
		homeDir = usr.HomeDir
	}
}

// copyFile copies a file from src to dst and retains the file permissions.
func copyFile(src, dst string) error {
	// Read the source file
	sourceData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get the file information to retrieve permissions
	fileInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Extract the permissions from the file information
	sourcePermissions := fileInfo.Mode().Perm()

	// Write the data to the destination with the same permissions
	return os.WriteFile(dst, sourceData, sourcePermissions)
}

// ExpandPath expands the ~ in paths to the absolute home directory path.
func ExpandPath(path string) (string, error) {
	once.Do(initHomeDir) // Ensure the homeDir is initialized only once

	if strings.HasPrefix(path, "~") {
		path = strings.Replace(path, "~", homeDir, 1)
	}
	return filepath.Abs(path)
}

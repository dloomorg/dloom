package internal

import (
	"os"
	"os/exec"
)

// RunShellCommand runs a shell command string in the user's environment.
func RunShellCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

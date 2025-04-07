package internal

import (
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"

	"os/exec"
	"strings"
)

// RunShellCommand executes a shell command
func RunShellCommand(cfg *Config, commandStr string, logger *logging.Logger) error {
	if cfg.Verbose {
		logger.LogInfo("Running command %s", commandStr)
	}

	if strings.TrimSpace(commandStr) == "" {
		return nil
	}

	cmd := exec.Command("bash", "-c", commandStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if cfg.DryRun {
		logger.LogInfo("Would run command: %s", commandStr)
	} else {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute command '%s': %w", commandStr, err)
		}
	}

	return nil
}

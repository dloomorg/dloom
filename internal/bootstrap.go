package internal

import (
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type BootstrapOptions struct {
	// Config is the application configuration
	Config *Config

	// Target is the repository URL or directory path to bootstrap
	Target string
}

// Bootstrap handles the repository cloning and git installation, or bootstrapping an existing directory
func Bootstrap(opts BootstrapOptions, logger *logging.Logger) error {
	// Check if target is a URL or directory
	if isURL(opts.Target) {
		return bootstrapFromURL(opts.Target, logger, opts.Config)
	}
	return bootstrapFromDirectory(opts.Target, logger, opts.Config)
}

func bootstrapFromURL(repoURL string, logger *logging.Logger, cfg *Config) error {
	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		logger.LogInfo("Git is not installed. Installing git...")
		if err := installGit(logger); err != nil {
			return fmt.Errorf("failed to install git: %w", err)
		}
	}

	// Extract repository name from URL
	repoName := getRepoName(repoURL)
	if repoName == "" {
		return fmt.Errorf("invalid repository URL: %s", repoURL)
	}

	// Check if repository directory already exists
	if _, err := os.Stat(repoName); err == nil {
		logger.LogInfo("Directory %s already exists, bootstrapping from existing directory", repoName)
		return bootstrapFromDirectory(repoName, logger, cfg)
	}

	if cfg.Verbose {
		logger.LogInfo("Bootstrapping repository %s", repoURL)
	}

	// Clone the repository
	cloneCmd := exec.Command("git", "clone", repoURL)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr

	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return bootstrapFromDirectory(repoName, logger, cfg)
}

func bootstrapFromDirectory(dir string, logger *logging.Logger, cfg *Config) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absPath)
	}

	// Change to the directory
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", absPath, err)
	}

	if !isDloomSupportedDirectory(".") {
		return fmt.Errorf("directory is not a dloom supported directory: %s", absPath)
	}

	if cfg.Verbose {
		logger.LogInfo("Bootstrapping dotfiles from %s", absPath)
	}
	return bootstrapDotfiles(".", logger, cfg)
}

func isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") || strings.HasPrefix(str, "git@")
}

func isDloomSupportedDirectory(dir string) bool {
	// we can later add checks here to see if the directory is actually a dotfile repo
	// Maybe should contain a dloom.yaml
	return true
}

func bootstrapDotfiles(dir string, logger *logging.Logger, cfg *Config) error {
	bootstrapScript := filepath.Join(dir, "scripts", "bootstrap.sh")

	// Check if bootstrap script exists
	if _, err := os.Stat(bootstrapScript); os.IsNotExist(err) {
		logger.LogInfo("No bootstrap script found at %s", bootstrapScript)
		logger.LogInfo("Feel free to add one at scripts/bootstrap.sh to set up your machine")
		return nil
	}

	if err := os.Chmod(bootstrapScript, 0755); err != nil {
		return fmt.Errorf("failed to make bootstrap script executable: %w", err)
	}

	if cfg.Verbose {
		logger.LogInfo("Running bootstrap script: %s", bootstrapScript)
	}

	// Run the bootstrap script using absolute path
	cmd := exec.Command("/bin/bash", bootstrapScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bootstrap script failed: %w", err)
	}

	logger.LogInfo("Bootstrap complete! Run 'dloom link' to link your dotfiles")
	return nil
}

func getRepoName(repoURL string) string {
	// Remove .git suffix if present
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Extract the last part of the URL
	parts := strings.Split(repoURL, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func installGit(logger *logging.Logger) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("brew", "install", "git")
	case "linux":
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "git")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

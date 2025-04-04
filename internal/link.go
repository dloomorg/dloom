package internal

import (
	"errors"
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
	"strings"
)

const permissions = 0750

type LinkOptions struct {
	// Config is the application configuration
	Config *Config

	// Packages is the list of package names to link
	Packages []string
}

// LinkPackages creates symlinks for all specified packages
func LinkPackages(opts LinkOptions, logger *logging.Logger) error {
	if len(opts.Packages) == 0 {
		return errors.New("no packages specified")
	}

	for _, pkg := range opts.Packages {
		if err := LinkPackage(pkg, opts.Config, logger); err != nil {
			return fmt.Errorf("failed to link package %s: %w", pkg, err)
		}

		if opts.Config.Verbose {
			logger.LogInfo("Successfully linked package: %s", pkg)
		}
	}

	return nil
}

// LinkPackage creates symlinks for a single package
func LinkPackage(pkgName string, cfg *Config, logger *logging.Logger) error {
	// Check if package has conditions and if they match
	pkgConfig := cfg.GetEffectiveConfig(pkgName, "")
	if pkgConfig.Conditions != nil && !cfg.MatchesConditions(pkgConfig.Conditions, logger) {
		if cfg.ShouldBeVerbose(pkgName, "") {
			logger.LogTrace("Skipping package %s: conditions not met", pkgName)
		}
		return nil
	}

	// Get the absolute path to the source package
	pkgDir, err := cfg.GetSourcePath(pkgName)
	if err != nil {
		return fmt.Errorf("failed to get source path for package %s: %w", pkgName, err)
	}

	// Ensure package directory exists
	pkgDirAbs, err := filepath.Abs(pkgDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(pkgDirAbs); os.IsNotExist(err) {
		return fmt.Errorf("package directory %s does not exist", pkgDirAbs)
	}

	// Walk through the package directory
	return filepath.Walk(pkgDirAbs, func(sourcePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if sourcePath == pkgDirAbs {
			return nil
		}

		// Calculate relative path from package directory
		relPath, err := filepath.Rel(pkgDirAbs, sourcePath)
		if err != nil {
			return err
		}

		// Check file-specific conditions
		fileConfig := cfg.GetEffectiveConfig(pkgName, relPath)
		if fileConfig.Conditions != nil && !cfg.MatchesConditions(fileConfig.Conditions, logger) {
			if cfg.ShouldBeVerbose(pkgName, relPath) {
				logger.LogTrace("Skipping file %s: conditions not met", relPath)
			}
			return nil
		}

		// Construct target path
		targetPath, err := cfg.GetTargetPath(pkgName, relPath)
		if err != nil {
			return fmt.Errorf("failed to get target path for package %s: %w", pkgName, err)
		}

		// If it's a directory, create it in the target directory
		if info.IsDir() {
			if cfg.IsDryRun(pkgName, relPath) {
				return nil
			}

			if err := os.MkdirAll(targetPath, permissions); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

			if cfg.ShouldBeVerbose(pkgName, relPath) {
				logger.LogTrace("Created directory: %s", targetPath)
			}
			return nil
		}

		// It's a file, handle symlinking
		return linkFile(sourcePath, targetPath, relPath, pkgName, cfg, logger)
	})
}

// linkFile creates a symlink from sourcePath to targetPath
func linkFile(sourcePath, targetPath, relPath, pkgName string, cfg *Config, logger *logging.Logger) error {
	// Create parent directories if needed
	targetDir := filepath.Dir(targetPath)

	if !cfg.IsDryRun(pkgName, relPath) {
		if err := os.MkdirAll(targetDir, permissions); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", targetDir, err)
		}
	}

	// Check if target already exists
	_, err := os.Lstat(targetPath)
	if err == nil {
		// Target exists, check if it's already the correct symlink
		linkDest, err := os.Readlink(targetPath)
		if err == nil && linkDest == sourcePath {
			// Already correctly linked
			if cfg.ShouldBeVerbose(pkgName, relPath) {
				logger.LogTrace("Already linked: %s", relPath)
			}
			return nil
		}

		// Target exists but is not the correct symlink
		if !cfg.ShouldForce(pkgName, relPath) {
			// only ask confirmation if we are running in non-dryrun mode
			if !cfg.IsDryRun(pkgName, relPath) {
				// Ask user for confirmation before removing
				logger.LogInfoNoReturn("Target already exists: %s, rel-path: %s, pkg: %s. Replace? [y/N] ", targetPath, relPath, pkgName)

				var response string
				_, err = fmt.Scanln(&response)
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}

				if strings.ToLower(response) != "y" {
					logger.LogInfo("Skipping file: %s", relPath)
					return nil
				}
			}
		}

		// Backup if backup directory is set
		backupPath, err := cfg.GetBackupPath(pkgName, relPath)
		if err != nil {
			return fmt.Errorf("failed to get backup path for package %s: %w", pkgName, err)
		}

		if backupPath != "" {
			backupDir := filepath.Dir(backupPath)

			if cfg.IsDryRun(pkgName, relPath) {
				logger.LogDryRun("Would backup %s to %s", targetPath, backupPath)
			} else {
				if err := os.MkdirAll(backupDir, permissions); err != nil {
					return fmt.Errorf("failed to create backup directory %s: %w", backupDir, err)
				}

				// Copy the file to backup location
				if err := copyFile(targetPath, backupPath); err != nil {
					return fmt.Errorf("failed to backup file %s: %w", targetPath, err)
				}

				logger.LogInfo("Backed up %s to %s", targetPath, backupPath)
			}
		}

		// Remove existing target
		if cfg.IsDryRun(pkgName, relPath) {
			logger.LogDryRun("Would remove existing target: %s", targetPath)
		} else {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("failed to remove existing target %s: %w", targetPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("failed to check target %s: %w", targetPath, err)
	}

	// Create symlink
	if cfg.IsDryRun(pkgName, relPath) {
		logger.LogDryRun("Would link: %s -> %s", targetPath, sourcePath)
	} else {
		// Expand source path
		srcAbsPath, err := ExpandPath(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to expand source path %s: %w", sourcePath, err)
		}
		// Expand target path
		destAbsPath, err := ExpandPath(targetPath)
		if err != nil {
			return fmt.Errorf("failed to expand target path %s: %w", targetPath, err)
		}

		if err := os.Symlink(srcAbsPath, destAbsPath); err != nil {
			return fmt.Errorf("failed to create symlink from %s to %s: %w", srcAbsPath, destAbsPath, err)
		}

		if cfg.ShouldBeVerbose(pkgName, relPath) {
			logger.LogTrace("Linked: %s -> %s", targetPath, sourcePath)
		}
	}

	return nil
}

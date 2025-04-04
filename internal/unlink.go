package internal

import (
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type UnlinkOptions struct {
	// Config is the application configuration
	Config *Config

	// Packages is the list of package names to unlink
	Packages []string
}

// UnlinkPackages removes symlinks for all specified packages
func UnlinkPackages(opts UnlinkOptions, logger *logging.Logger) error {
	if len(opts.Packages) == 0 {
		return fmt.Errorf("no packages specified")
	}

	for _, pkg := range opts.Packages {
		if err := UnlinkPackage(pkg, opts.Config, logger); err != nil {
			return fmt.Errorf("failed to unlink package %s: %w", pkg, err)
		}

		if opts.Config.Verbose {
			logger.LogTrace("Successfully unlinked package: %s", pkg)
		}
	}

	return nil
}

// UnlinkPackage removes symlinks for a single package
func UnlinkPackage(pkgName string, cfg *Config, logger *logging.Logger) error {
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

	// Track empty directories to potentially remove them later
	emptyDirs := make(map[string]bool)

	// Walk through the package directory
	err = filepath.Walk(pkgDirAbs, func(sourcePath string, info os.FileInfo, err error) error {
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
			return fmt.Errorf("failed to get target path for %s: %w", relPath, err)
		}

		// If it's a directory, add to the map of directories to check later
		if info.IsDir() {
			emptyDirs[targetPath] = true
			return nil
		}

		// It's a file, handle unlinking
		return unlinkFile(sourcePath, targetPath, relPath, pkgName, cfg, logger)
	})

	if err != nil {
		return err
	}

	// Clean up empty directories, starting from the deepest ones
	if !cfg.IsDryRun(pkgName, "") {
		var dirs []string
		for dir := range emptyDirs {
			dirs = append(dirs, dir)
		}

		// Sort directories by depth (deepest first)
		sortByDepth(dirs)

		for _, dir := range dirs {
			// Check if directory is empty
			entries, err := os.ReadDir(dir)
			if err != nil {
				// Directory might not exist or other error, just skip
				continue
			}

			if len(entries) == 0 {
				if cfg.IsDryRun(pkgName, "") {
					logger.LogDryRun("Would remove empty directory: %s", dir)
				} else {
					if err := os.Remove(dir); err != nil {
						// Non-critical error, just log and continue
						if cfg.ShouldBeVerbose(pkgName, "") {
							logger.LogWarning("Failed to remove directory %s: %v", dir, err)
						}
					} else if cfg.ShouldBeVerbose(pkgName, "") {
						logger.LogTrace("Removed empty directory: %s", dir)
					}
				}
			}
		}
	}

	return nil
}

// unlinkFile removes a symlink if it points to the expected source
func unlinkFile(sourcePath, targetPath, relPath, pkgName string, cfg *Config, logger *logging.Logger) error {
	// Check if target exists
	fi, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Target doesn't exist, nothing to unlink
			if cfg.ShouldBeVerbose(pkgName, relPath) {
				logger.LogTrace("No symlink found at %s", targetPath)
			}
			return nil
		}
		return fmt.Errorf("failed to check target %s: %w", targetPath, err)
	}

	// Check if it's a symlink
	if fi.Mode()&os.ModeSymlink == 0 {
		// Not a symlink, leave it alone
		if cfg.ShouldBeVerbose(pkgName, relPath) {
			logger.LogTrace("Not a symlink: %s", targetPath)
		}
		return nil
	}

	// Check if the symlink points to our source file
	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink %s: %w", targetPath, err)
	}

	// Only remove if it points to our source file
	if linkDest == sourcePath {
		if cfg.IsDryRun(pkgName, relPath) {
			logger.LogDryRun("Would remove symlink: %s -> %s", targetPath, sourcePath)
		} else {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("failed to remove symlink %s: %w", targetPath, err)
			}

			if cfg.ShouldBeVerbose(pkgName, relPath) {
				logger.LogTrace("Removed symlink: %s", targetPath)
			}
		}
	} else if cfg.ShouldBeVerbose(pkgName, relPath) {
		logger.LogTrace("Symlink points elsewhere, not removing: %s -> %s\n", targetPath, linkDest)
	}

	// Restore from backup if available
	backupPath, err := cfg.GetBackupPath(pkgName, relPath)
	if err != nil {
		return fmt.Errorf("failed to get backup path for package %s: %w", pkgName, err)
	}

	if backupPath != "" {
		// Check if backup exists
		if _, err := os.Stat(backupPath); err == nil {
			if cfg.IsDryRun(pkgName, relPath) {
				logger.LogDryRun("Would restore from backup: %s -> %s", backupPath, targetPath)
			} else {
				// Copy backup back to target
				if err := copyFile(backupPath, targetPath); err != nil {
					return fmt.Errorf("failed to restore from backup %s: %w", backupPath, err)
				}

				if cfg.ShouldBeVerbose(pkgName, relPath) {
					logger.LogTrace("Restored from backup: %s -> %s", backupPath, targetPath)
				}

				// Remove the backup
				if err := os.Remove(backupPath); err != nil {
					logger.LogWarning("Warning: Failed to remove backup %s: %v", backupPath, err)
				}
			}
		}
	}

	return nil
}

// sortByDepth sorts paths by depth (deepest first)
func sortByDepth(paths []string) {
	sort.Slice(paths, func(i, j int) bool {
		// Count separators as a proxy for depth
		depthI := strings.Count(paths[i], string(os.PathSeparator))
		depthJ := strings.Count(paths[j], string(os.PathSeparator))
		return depthI > depthJ
	})
}

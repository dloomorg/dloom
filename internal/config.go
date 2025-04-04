package internal

import (
	"fmt"
	"github.com/swaranga/dloom/internal/conditions"
	"github.com/swaranga/dloom/internal/logging"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SourceDir      string                    `yaml:"source_dir"`
	TargetDir      string                    `yaml:"target_dir"`
	BackupDir      string                    `yaml:"backup_dir"`
	Force          bool                      `yaml:"force"`
	Verbose        bool                      `yaml:"verbose"`
	DryRun         bool                      `yaml:"dry_run"`
	Packages       map[string]*PackageConfig `yaml:"link_overrides"`
	IgnorePackages []string                  `yaml:"ignore_packages"`
}

type PackageConfig struct {
	SourceDir  string                 `yaml:"source_dir"`
	TargetDir  string                 `yaml:"target_dir"`
	BackupDir  string                 `yaml:"backup_dir"`
	Force      *bool                  `yaml:"force"`
	Verbose    *bool                  `yaml:"verbose"`
	DryRun     bool                   `yaml:"dry_run"`
	Conditions *ConditionSet          `yaml:"conditions"`
	Files      map[string]*FileConfig `yaml:"file_overrides"`
}

type FileConfig struct {
	TargetDir  string        `yaml:"target_dir"`
	TargetName string        `yaml:"target_name"`
	BackupDir  string        `yaml:"backup_dir"`
	Force      *bool         `yaml:"force"`
	Verbose    *bool         `yaml:"verbose"`
	DryRun     bool          `yaml:"dry_run"`
	Conditions *ConditionSet `yaml:"conditions"`
}

type ConditionSet struct {
	OS                []string          `yaml:"os"`
	Distro            []string          `yaml:"distro"`
	Executable        []string          `yaml:"executable"`
	ExecutableVersion map[string]string `yaml:"executable_version"`
	User              []string          `yaml:"user"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	return &Config{
		SourceDir:      ".",     // Current directory by default
		TargetDir:      homeDir, // User's home directory by default
		BackupDir:      filepath.Join(homeDir, ".dloom/backups"),
		Force:          false,
		Verbose:        false,
		DryRun:         false,
		Packages:       make(map[string]*PackageConfig),
		IgnorePackages: []string{
			".git",
			".idea",
			".gitignore"},
	}
}

// LoadConfig loads configuration from the specified file
// If the file doesn't exist, returns default config
func LoadConfig(configPath string, logger *logging.Logger) (*Config, error) {
	config := DefaultConfig()

	// If no config path specified, look in default locations
	// First try to find a
	if configPath == "" {
		logger.LogTrace("No config file path specified, using defaults")
		// First, try current directory and see if a dloom/config.yaml exists
		// If not, try ~/.config/dloom/config.yaml
		currentDir, err := os.Getwd()
		if err == nil {
			configPath = filepath.Join(currentDir, "dloom", "config.yaml")
			logger.LogTrace("Attempting to load config file: %s", configPath)
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				// Next, try ~/.config/dloom/config.yaml
				homeDir, err := os.UserHomeDir()
				if err == nil {
					configPath = filepath.Join(homeDir, ".config", "dloom", "config.yaml")
					logger.LogTrace("Attempting to load config file: %s", configPath)
					if _, err := os.Stat(configPath); os.IsNotExist(err) {
						// No config file found, use defaults
						return config, nil
					}
				}
			}
		}
	}

	// Read config file
	logger.LogTrace("Loading config file: %s", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return defaults
			logger.LogWarning("Config file: %s not found, using defaults", configPath)
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	logger.LogTrace("Deserializing config: %s", data)

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Verbose {
		logger.LogTrace("Loaded config: %+v", config)
	}

	return config, nil
}

// GetSourcePath returns the full path for a source package
func (c *Config) GetSourcePath(packageName string) (string, error) {
	// Check for package-specific source directory
	if pkg, exists := c.Packages[packageName]; exists && pkg.SourceDir != "" {
		return pkg.SourceDir, nil
	}

	// Fall back to global source directory + package name
	return ExpandPath(filepath.Join(c.SourceDir, packageName))
}

// GetTargetPath returns the full path for a target in the target directory
func (c *Config) GetTargetPath(packageName, relativePath string) (string, error) {
	// Get the effective configuration for this file
	effectiveConfig := c.GetEffectiveConfig(packageName, relativePath)

	// Construct the target path using the effective target directory
	return ExpandPath(filepath.Join(effectiveConfig.TargetDir, effectiveConfig.TargetName))
}

// GetBackupPath returns the path where a file should be backed up
func (c *Config) GetBackupPath(packageName, relativePath string) (string, error) {
	// Get the effective configuration for this file
	effectiveConfig := c.GetEffectiveConfig(packageName, relativePath)

	// If backup directory is empty, backups are disabled
	if effectiveConfig.BackupDir == "" {
		return "", nil
	}

	// Otherwise, construct the backup path
	return ExpandPath(filepath.Join(effectiveConfig.BackupDir, relativePath))
}

// ShouldBeVerbose returns whether verbose output should be enabled for a specific file
func (c *Config) ShouldBeVerbose(packageName, relativePath string) bool {
	// Get the effective configuration for this file
	effectiveConfig := c.GetEffectiveConfig(packageName, relativePath)

	// The Verbose field should never be nil in the effective config
	// since it's initialized with the global value
	return *effectiveConfig.Verbose
}

// IsDryRun returns whether dry run mode is enabled for a specific file
func (c *Config) IsDryRun(packageName, relativePath string) bool {
	// Get the effective configuration for this file
	effectiveConfig := c.GetEffectiveConfig(packageName, relativePath)

	// Return the effective dry run setting
	return effectiveConfig.DryRun
}

// ShouldIgnorePackage checks if a package or a file should be skipped from Symlinking
func (c *Config) ShouldIgnorePackage(relPath string) bool {
	for _, ignoredPkg := range c.IgnorePackages {
		if strings.HasSuffix(relPath, ignoredPkg) {
			return true
		}
	}
	return false
}

// ShouldForce returns whether force mode is enabled for a specific file
func (c *Config) ShouldForce(packageName, relativePath string) bool {
	// Get the effective configuration for this file
	effectiveConfig := c.GetEffectiveConfig(packageName, relativePath)

	// The Force field should never be nil in the effective config
	// since it's initialized with the global value
	return *effectiveConfig.Force
}

// GetEffectiveConfig returns the effective configuration for a specific file
// by merging global, package-level, and file-specific settings
func (c *Config) GetEffectiveConfig(packageName, relativePath string) *FileConfig {
	// Start with default settings derived from global config
	effectiveConfig := &FileConfig{
		TargetDir:  c.TargetDir,
		TargetName: relativePath, // Default to the file name
		BackupDir:  c.BackupDir,
		Force:      &c.Force,
		Verbose:    &c.Verbose,
		DryRun:     c.DryRun,
		Conditions: nil, // No conditions at global level
	}

	// Apply package-level settings if they exist
	pkg, pkgExists := c.Packages[packageName]
	if pkgExists {
		// Override with package settings where specified
		if pkg.TargetDir != "" {
			effectiveConfig.TargetDir = pkg.TargetDir
		}

		if pkg.BackupDir != "" {
			effectiveConfig.BackupDir = pkg.BackupDir
		}

		if pkg.Force != nil {
			effectiveConfig.Force = pkg.Force
		}

		if pkg.Verbose != nil {
			effectiveConfig.Verbose = pkg.Verbose
		}

		if pkg.DryRun {
			effectiveConfig.DryRun = pkg.DryRun
		}

		// Apply package conditions
		effectiveConfig.Conditions = pkg.Conditions

		// Check for file-specific settings
		var fileConfig *FileConfig

		// First, try exact file match; regardless of declaration order
		// Get the file name from the relative path
		var relativePathTargetName = filepath.Base(relativePath)
		if fc, exists := pkg.Files[relativePathTargetName]; exists {
			fileConfig = fc
		} else {
			// Try regex matches
			for pattern, fc := range pkg.Files {
				// Only process entries that start with "regex:"
				if strings.HasPrefix(pattern, "regex:") {
					// Extract the actual regex pattern
					regexPattern := strings.TrimPrefix(pattern, "regex:")

					matched, err := regexp.MatchString(regexPattern, relativePath)
					if err == nil && matched {
						fileConfig = fc
						break
					}
				}
			}
		}

		// Apply file-specific settings if found
		if fileConfig != nil {
			if fileConfig.TargetDir != "" {
				effectiveConfig.TargetDir = fileConfig.TargetDir
			}

			if fileConfig.TargetName != "" {
				// First extract the directory from the relative path
				// and then join it with the target name
				relativePathDir := filepath.Dir(relativePath)
				effectiveConfig.TargetName = filepath.Join(relativePathDir, fileConfig.TargetName)
			}

			if fileConfig.BackupDir != "" {
				effectiveConfig.BackupDir = fileConfig.BackupDir
			}

			if fileConfig.Force != nil {
				effectiveConfig.Force = fileConfig.Force
			}

			if fileConfig.Verbose != nil {
				effectiveConfig.Verbose = fileConfig.Verbose
			}

			if fileConfig.DryRun {
				effectiveConfig.DryRun = fileConfig.DryRun
			}

			// For conditions, file conditions completely override package conditions
			// rather than merging them
			if fileConfig.Conditions != nil {
				effectiveConfig.Conditions = fileConfig.Conditions
			}
		}
	}

	return effectiveConfig
}

// MatchesConditions checks if the current environment matches the given conditions
func (c *Config) MatchesConditions(conditionSet *ConditionSet, logger *logging.Logger) bool {
	if conditionSet == nil {
		return true // No conditions means always match
	}

	// Check OS conditions
	if len(conditionSet.OS) > 0 &&
		!conditions.MatchesOSCondition(conditionSet.OS) {
		return false
	}

	// Check Linux distro condition
	if len(conditionSet.Distro) > 0 &&
		!conditions.MatchesDistroCondition(conditionSet.Distro) {
		return false
	}

	// Check executable conditions
	if len(conditionSet.Executable) > 0 &&
		!conditions.MatchesExecutableCondition(conditionSet.Executable) {
		return false
	}

	// Check executable version conditions
	if len(conditionSet.ExecutableVersion) > 0 &&
		!conditions.MatchesExecutableVersionCondition(conditionSet.ExecutableVersion, logger) {
		return false
	}

	// Check user conditions
	if len(conditionSet.User) > 0 &&
		!conditions.MatchesUserCondition(conditionSet.User) {
		return false
	}

	return true
}

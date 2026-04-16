package internal

import (
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArgKind classifies a CLI argument as either a bare package name or a path.
type ArgKind int

const (
	// ArgBare is a bare package name like "vim" or "zsh"
	ArgBare ArgKind = iota
	// ArgPath is a filesystem path like ".", "..", "~/dotfiles", "/abs/path"
	ArgPath
)

// ClassifyArg determines whether a raw argument is a bare package name or a path.
func ClassifyArg(arg string) ArgKind {
	if arg == "." || arg == ".." {
		return ArgPath
	}
	if strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~") ||
		strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") {
		return ArgPath
	}
	if strings.Contains(arg, "/") {
		return ArgPath
	}
	return ArgBare
}

// isAbsoluteArg returns true if the raw argument is an absolute or home-relative
// path (starts with "/" or "~"), as opposed to a relative path like "./" or "../".
func isAbsoluteArg(arg string) bool {
	return strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "~")
}

// ResolveArgs takes raw CLI arguments and the current config, classifies each,
// resolves paths, and returns a deduplicated list of package names along with
// the effective Config to use (which may differ from the input cfg if an
// external dotfiles directory was specified).
//
// Path resolution strategy:
//   - "." or paths that resolve to source_dir → enumerate all packages
//   - Relative paths ("./vim", "vim/", "../foo") that are children of source_dir
//     → extract the relative portion as a package name
//   - Absolute/home paths ("/abs/path", "~/dotfiles") → always treated as a
//     dotfiles root to enumerate, even if technically a child of source_dir.
//     This prevents surprising behavior when source_dir defaults to CWD.
func ResolveArgs(args []string, cfg *Config, logger *logging.Logger) ([]string, *Config, error) {
	effectiveCfg := cfg
	seen := make(map[string]bool)
	var packages []string
	var overrideDir string

	for _, arg := range args {
		kind := ClassifyArg(arg)

		if kind == ArgBare {
			if !seen[arg] {
				seen[arg] = true
				packages = append(packages, arg)
			}
			continue
		}

		// It's a path — resolve to absolute
		absPath, err := ExpandPath(arg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve path %s: %w", arg, err)
		}
		absPath = filepath.Clean(absPath)

		// Verify the path exists and is a directory
		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil, fmt.Errorf("directory does not exist: %s", absPath)
			}
			return nil, nil, fmt.Errorf("failed to access %s: %w", absPath, err)
		}
		if !info.IsDir() {
			return nil, nil, fmt.Errorf("not a directory: %s", absPath)
		}

		sourceDir := filepath.Clean(cfg.SourceDir)

		// Resolve symlinks for both paths to ensure consistent comparison
		// (e.g., on macOS /var -> /private/var)
		absPathResolved, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			absPathResolved = absPath
		}
		sourceDirResolved, err := filepath.EvalSymlinks(sourceDir)
		if err != nil {
			sourceDirResolved = sourceDir
		}

		if absPathResolved == sourceDirResolved {
			// Path equals source_dir — enumerate all packages
			pkgs, err := EnumeratePackages(sourceDir, cfg.IgnorePackages, logger)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to enumerate packages in %s: %w", sourceDir, err)
			}
			for _, pkg := range pkgs {
				if !seen[pkg] {
					seen[pkg] = true
					packages = append(packages, pkg)
				}
			}
		} else if !isAbsoluteArg(arg) && isSubdirOf(absPathResolved, sourceDirResolved) {
			// Relative path that is a child of source_dir — extract package name.
			// Only relative args (./vim, vim/, ../x) get this treatment;
			// absolute paths always go to the external-directory branch below.
			relName, err := filepath.Rel(sourceDirResolved, absPathResolved)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to compute relative path: %w", err)
			}
			if !seen[relName] {
				seen[relName] = true
				packages = append(packages, relName)
			}
		} else {
			// External directory — override source_dir and enumerate packages.
			if overrideDir != "" && overrideDir != absPath {
				return nil, nil, fmt.Errorf("conflicting source directories: %s and %s", overrideDir, absPath)
			}
			overrideDir = absPath

			// Load config from the external directory if available
			externalConfigPath := filepath.Join(absPath, "dloom", "config.yaml")
			newCfg, err := LoadConfig(externalConfigPath, logger)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load config from %s: %w", absPath, err)
			}
			newCfg.SourceDir = absPath

			// Preserve CLI flag overrides from the original config
			newCfg.Force = cfg.Force
			newCfg.Verbose = cfg.Verbose
			newCfg.DryRun = cfg.DryRun
			if cfg.TargetDir != "" {
				newCfg.TargetDir = cfg.TargetDir
			}

			effectiveCfg = newCfg

			// Enumerate packages from the external directory
			pkgs, err := EnumeratePackages(absPath, effectiveCfg.IgnorePackages, logger)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to enumerate packages in %s: %w", absPath, err)
			}
			for _, pkg := range pkgs {
				if !seen[pkg] {
					seen[pkg] = true
					packages = append(packages, pkg)
				}
			}
		}
	}

	return packages, effectiveCfg, nil
}

// EnumeratePackages lists top-level directories in sourceDir that are not
// in the ignore list. Root-level files are skipped with an info log.
// Returns sorted package names.
func EnumeratePackages(sourceDir string, ignorePackages []string, logger *logging.Logger) ([]string, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", sourceDir, err)
	}

	var packages []string
	for _, entry := range entries {
		if !entry.IsDir() {
			logger.LogInfo("Skipping root-level file (not in a package): %s", entry.Name())
			continue
		}

		name := entry.Name()
		if shouldIgnoreName(name, ignorePackages) {
			continue
		}

		packages = append(packages, name)
	}

	sort.Strings(packages)
	return packages, nil
}

// shouldIgnoreName checks if a name matches any entry in the ignore list.
// Uses the same suffix-matching logic as Config.ShouldIgnorePackage.
func shouldIgnoreName(name string, ignorePackages []string) bool {
	for _, ignored := range ignorePackages {
		if strings.HasSuffix(name, ignored) {
			return true
		}
	}
	return false
}

// isSubdirOf checks if child is a subdirectory of parent.
// Both paths must be absolute and clean.
func isSubdirOf(child, parent string) bool {
	// Ensure parent ends with separator for prefix matching
	parentPrefix := parent
	if !strings.HasSuffix(parentPrefix, string(filepath.Separator)) {
		parentPrefix += string(filepath.Separator)
	}
	return strings.HasPrefix(child, parentPrefix)
}

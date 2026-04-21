package internal

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
	"strings"
)

type AdoptOptions struct {
	Config      *Config
	Package     string
	Targets     []string
	Interactive bool
}

type adoptAction int

const (
	adoptActionNoop adoptAction = iota
	adoptActionCopy
	adoptActionRelink
	adoptActionSkip
)

type adoptCandidate struct {
	relPath    string
	sourcePath string
	targetPath string
	action     adoptAction
	skipReason string
}

func AdoptPackage(opts AdoptOptions, logger *logging.Logger) error {
	if opts.Package == "" {
		return errors.New("no package specified")
	}
	if len(opts.Targets) == 0 {
		return errors.New("no targets specified")
	}
	if opts.Config.Force {
		return errors.New("adopt does not support --force; resolve source conflicts manually")
	}

	if err := validateAdoptPackageConfig(opts.Package, opts.Config); err != nil {
		return err
	}

	pkgConfig := opts.Config.GetEffectiveConfig(opts.Package, "")
	if pkgConfig.Conditions != nil && !opts.Config.MatchesConditions(pkgConfig.Conditions, logger) {
		return fmt.Errorf("package %s cannot be adopted because its conditions do not match the current environment", opts.Package)
	}

	sourceRoot, err := opts.Config.GetSourcePath(opts.Package)
	if err != nil {
		return fmt.Errorf("failed to resolve source path for package %s: %w", opts.Package, err)
	}
	sourceRoot, err = ExpandPath(sourceRoot)
	if err != nil {
		return fmt.Errorf("failed to resolve source path for package %s: %w", opts.Package, err)
	}

	targetRoot, err := ExpandPath(pkgConfig.TargetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve target path for package %s: %w", opts.Package, err)
	}

	candidates, err := collectAdoptCandidates(opts.Package, opts.Targets, sourceRoot, targetRoot, opts.Config, logger)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		logger.LogWarning("No files found to adopt")
		return nil
	}

	for i := range candidates {
		if err := planAdoptCandidate(&candidates[i]); err != nil {
			return err
		}
	}

	var adoptedCount int
	for _, candidate := range candidates {
		adopted, err := executeAdoptCandidate(candidate, opts.Package, opts.Config, logger, opts.Interactive)
		if err != nil {
			return fmt.Errorf("failed to adopt %s: %w", candidate.targetPath, err)
		}
		if adopted {
			adoptedCount++
		}
	}

	if opts.Config.Verbose {
		if opts.Config.DryRun {
			logger.LogInfo("Dry run planned %d file(s) for adoption into package: %s", adoptedCount, opts.Package)
		} else {
			logger.LogInfo("Successfully adopted %d file(s) into package: %s", adoptedCount, opts.Package)
		}
	}

	return nil
}

func validateAdoptPackageConfig(pkgName string, cfg *Config) error {
	pkg, exists := cfg.Packages[pkgName]
	if !exists {
		return nil
	}

	for pattern, fileCfg := range pkg.Files {
		if strings.HasPrefix(pattern, "regex:") {
			return fmt.Errorf("package %s cannot be adopted because regex file_overrides are not supported", pkgName)
		}
		if fileCfg != nil && (fileCfg.TargetName != "" || fileCfg.TargetDir != "") {
			return fmt.Errorf("package %s cannot be adopted because file_overrides with target_name or target_dir are not supported", pkgName)
		}
	}

	return nil
}

func collectAdoptCandidates(pkgName string, targets []string, sourceRoot, targetRoot string, cfg *Config, logger *logging.Logger) ([]adoptCandidate, error) {
	seenTargets := make(map[string]bool)
	seenSources := make(map[string]string)
	var candidates []adoptCandidate

	addCandidate := func(targetPath string, info os.FileInfo) error {
		relPath, err := pathRelativeToRoot(targetRoot, targetPath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if relPath != "." && cfg.ShouldIgnorePackage(relPath) {
				if cfg.Verbose {
					logger.LogTrace("Skipping directory %s: ignored by configuration", targetPath)
				}
				return filepath.SkipDir
			}
			return nil
		}

		if cfg.ShouldIgnorePackage(relPath) {
			if cfg.Verbose {
				logger.LogTrace("Skipping file %s: ignored by configuration", targetPath)
			}
			return nil
		}

		fileConfig := cfg.GetEffectiveConfig(pkgName, relPath)
		if fileConfig.Conditions != nil && !cfg.MatchesConditions(fileConfig.Conditions, logger) {
			return fmt.Errorf("file %s cannot be adopted because its conditions do not match the current environment", targetPath)
		}

		sourcePath := filepath.Join(sourceRoot, relPath)
		if previousTarget, exists := seenSources[sourcePath]; exists && previousTarget != targetPath {
			return fmt.Errorf("target paths %s and %s both map to source path %s", previousTarget, targetPath, sourcePath)
		}
		if seenTargets[targetPath] {
			return nil
		}

		seenTargets[targetPath] = true
		seenSources[sourcePath] = targetPath
		candidates = append(candidates, adoptCandidate{
			relPath:    relPath,
			sourcePath: sourcePath,
			targetPath: targetPath,
		})

		return nil
	}

	for _, rawTarget := range targets {
		targetPath, err := ExpandPath(rawTarget)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target path %s: %w", rawTarget, err)
		}

		info, err := os.Lstat(targetPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("target path does not exist: %s", targetPath)
			}
			return nil, fmt.Errorf("failed to access target path %s: %w", targetPath, err)
		}

		if info.IsDir() {
			err = filepath.Walk(targetPath, func(path string, walkInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				return addCandidate(path, walkInfo)
			})
			if err != nil {
				return nil, err
			}
			continue
		}

		if err := addCandidate(targetPath, info); err != nil {
			return nil, err
		}
	}

	return candidates, nil
}

func planAdoptCandidate(candidate *adoptCandidate) error {
	if pathsEqual(candidate.targetPath, candidate.sourcePath) {
		return fmt.Errorf("source and target resolve to the same path: %s", candidate.targetPath)
	}

	targetInfo, err := os.Lstat(candidate.targetPath)
	if err != nil {
		return fmt.Errorf("failed to inspect target path %s: %w", candidate.targetPath, err)
	}
	if targetInfo.IsDir() {
		return fmt.Errorf("target path %s is a directory; only files can be adopted", candidate.targetPath)
	}

	if targetInfo.Mode()&os.ModeSymlink != 0 {
		linkDest, err := resolveLinkDestination(candidate.targetPath)
		if err != nil {
			return fmt.Errorf("failed to inspect symlink %s: %w", candidate.targetPath, err)
		}

		if pathsEqual(linkDest, candidate.sourcePath) {
			if _, err := os.Stat(candidate.sourcePath); err != nil {
				return fmt.Errorf("target %s points to missing source %s", candidate.targetPath, candidate.sourcePath)
			}
			candidate.action = adoptActionNoop
			return nil
		}

		candidate.action = adoptActionSkip
		candidate.skipReason = fmt.Sprintf("already a symlink -> %s", linkDest)
		return nil
	}

	if !targetInfo.Mode().IsRegular() {
		candidate.action = adoptActionSkip
		candidate.skipReason = "not a regular file"
		return nil
	}

	sourceInfo, err := os.Lstat(candidate.sourcePath)
	if err == nil {
		if sourceInfo.IsDir() {
			return fmt.Errorf("source path already exists as a directory: %s", candidate.sourcePath)
		}
		if sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.Mode().IsRegular() {
			return fmt.Errorf("source path already exists and is not a regular file: %s", candidate.sourcePath)
		}

		equal, err := filesEqual(candidate.targetPath, candidate.sourcePath)
		if err != nil {
			return fmt.Errorf("failed to compare %s and %s: %w", candidate.targetPath, candidate.sourcePath, err)
		}
		if !equal {
			return fmt.Errorf("source file already exists with different contents: %s", candidate.sourcePath)
		}

		candidate.action = adoptActionRelink
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect source path %s: %w", candidate.sourcePath, err)
	}

	candidate.action = adoptActionCopy
	return nil
}

func executeAdoptCandidate(candidate adoptCandidate, pkgName string, cfg *Config, logger *logging.Logger, interactive bool) (bool, error) {
	if candidate.action == adoptActionSkip {
		if cfg.IsDryRun(pkgName, candidate.relPath) {
			logger.LogDryRun("Would skip: %s (%s)", candidate.targetPath, candidate.skipReason)
		} else if cfg.ShouldBeVerbose(pkgName, candidate.relPath) {
			logger.LogTrace("Skipping: %s (%s)", candidate.targetPath, candidate.skipReason)
		}
		return false, nil
	}

	if candidate.action == adoptActionNoop {
		if cfg.IsDryRun(pkgName, candidate.relPath) {
			logger.LogDryRun("Would keep existing managed symlink: %s -> %s", candidate.targetPath, candidate.sourcePath)
		} else if cfg.ShouldBeVerbose(pkgName, candidate.relPath) {
			logger.LogTrace("Already adopted: %s", candidate.targetPath)
		}
		return false, nil
	}

	if interactive {
		confirmed, err := confirmAdoptCandidate(candidate, logger)
		if err != nil {
			return false, err
		}
		if !confirmed {
			if cfg.IsDryRun(pkgName, candidate.relPath) {
				logger.LogDryRun("Would skip: %s (not confirmed)", candidate.targetPath)
			} else if cfg.ShouldBeVerbose(pkgName, candidate.relPath) {
				logger.LogTrace("Skipping file: %s", candidate.relPath)
			}
			return false, nil
		}
	}

	if cfg.IsDryRun(pkgName, candidate.relPath) {
		switch candidate.action {
		case adoptActionRelink:
			logger.LogDryRun("Would replace %s with symlink to existing source %s", candidate.targetPath, candidate.sourcePath)
		default:
			logger.LogDryRun("Would adopt: %s -> %s", candidate.targetPath, candidate.sourcePath)
		}
		return true, nil
	}

	if err := os.MkdirAll(filepath.Dir(candidate.sourcePath), permissions); err != nil {
		return false, fmt.Errorf("failed to create source directory for %s: %w", candidate.sourcePath, err)
	}

	if candidate.action == adoptActionCopy {
		if err := copyFile(candidate.targetPath, candidate.sourcePath); err != nil {
			return false, fmt.Errorf("failed to copy %s to %s: %w", candidate.targetPath, candidate.sourcePath, err)
		}
	}

	if err := os.Remove(candidate.targetPath); err != nil {
		if candidate.action == adoptActionCopy {
			if cleanupErr := os.Remove(candidate.sourcePath); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
				return false, fmt.Errorf("failed to remove target %s: %w (also failed to remove copied source %s: %v)", candidate.targetPath, err, candidate.sourcePath, cleanupErr)
			}
		}
		return false, fmt.Errorf("failed to remove target %s: %w", candidate.targetPath, err)
	}

	if err := createAbsoluteSymlink(candidate.sourcePath, candidate.targetPath); err != nil {
		restoreErr := copyFile(candidate.sourcePath, candidate.targetPath)
		if restoreErr != nil {
			return false, fmt.Errorf("failed to create symlink %s -> %s: %w (original content is still available at %s; restoring target failed: %v)", candidate.targetPath, candidate.sourcePath, err, candidate.sourcePath, restoreErr)
		}

		if candidate.action == adoptActionCopy {
			if cleanupErr := os.Remove(candidate.sourcePath); cleanupErr != nil && !os.IsNotExist(cleanupErr) {
				return false, fmt.Errorf("failed to create symlink %s -> %s: %w (restored target, but failed to remove copied source %s: %v)", candidate.targetPath, candidate.sourcePath, err, candidate.sourcePath, cleanupErr)
			}
		}

		return false, fmt.Errorf("failed to create symlink %s -> %s: %w (restored original target)", candidate.targetPath, candidate.sourcePath, err)
	}

	if cfg.ShouldBeVerbose(pkgName, candidate.relPath) {
		switch candidate.action {
		case adoptActionRelink:
			logger.LogTrace("Replaced file with symlink: %s -> %s", candidate.targetPath, candidate.sourcePath)
		default:
			logger.LogTrace("Adopted: %s -> %s", candidate.targetPath, candidate.sourcePath)
		}
	}

	return true, nil
}

func pathRelativeToRoot(rootPath, targetPath string) (string, error) {
	tryRel := func(root, target string) (string, bool, error) {
		relPath, err := filepath.Rel(root, target)
		if err != nil {
			return "", false, err
		}
		if relPath == "." {
			return relPath, true, nil
		}

		parentPrefix := ".." + string(os.PathSeparator)
		if relPath == ".." || strings.HasPrefix(relPath, parentPrefix) {
			return "", false, nil
		}

		return filepath.Clean(relPath), true, nil
	}

	if relPath, ok, err := tryRel(filepath.Clean(rootPath), filepath.Clean(targetPath)); err != nil {
		return "", fmt.Errorf("failed to resolve target path %s relative to %s: %w", targetPath, rootPath, err)
	} else if ok {
		return relPath, nil
	}

	rootResolved := resolvePathForComparison(rootPath)
	targetResolved := resolvePathForComparison(targetPath)
	if relPath, ok, err := tryRel(rootResolved, targetResolved); err != nil {
		return "", fmt.Errorf("failed to resolve target path %s relative to %s: %w", targetPath, rootPath, err)
	} else if ok {
		return relPath, nil
	}

	return "", fmt.Errorf("target path %s is outside package target directory %s", targetPath, rootPath)
}

func resolvePathForComparison(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(path)
}

func pathsEqual(a, b string) bool {
	return resolvePathForComparison(a) == resolvePathForComparison(b)
}

func resolveLinkDestination(path string) (string, error) {
	linkDest, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(linkDest) {
		return filepath.Clean(linkDest), nil
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), linkDest)), nil
}

func filesEqual(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	dataA, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	dataB, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}

	return bytes.Equal(dataA, dataB), nil
}

func createAbsoluteSymlink(sourcePath, targetPath string) error {
	sourceAbs, err := ExpandPath(sourcePath)
	if err != nil {
		return err
	}
	targetAbs, err := ExpandPath(targetPath)
	if err != nil {
		return err
	}

	return os.Symlink(sourceAbs, targetAbs)
}

func confirmAdoptCandidate(candidate adoptCandidate, logger *logging.Logger) (bool, error) {
	logger.LogInfoNoReturn("Adopt %s -> %s? [y/N] ", candidate.targetPath, candidate.sourcePath)

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes", nil
}

// Package main provides the entry point for the dloom command
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/dloomorg/dloom/internal"
	"github.com/dloomorg/dloom/internal/logging"
	"os"
	"path/filepath"
)

// Command-line flags
var (
	configPath string
	force      bool
	verbose    bool
	dryRun     bool
	sourceDir  string
	targetDir  string
	noColor    bool
)

func init() {
	// Define command-line flags
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")

	flag.BoolVar(&noColor, "no-color", false, "Whether to disable color output")

	flag.BoolVar(&force, "force", false, "Force overwriting existing files")
	flag.BoolVar(&force, "f", false, "Force overwriting existing files (shorthand)")

	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")

	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	flag.BoolVar(&dryRun, "d", false, "Show what would be done without making changes (shorthand)")
	flag.BoolVar(&dryRun, "n", false, "Show what would be done without making changes (shorthand)")

	flag.StringVar(&sourceDir, "source", "", "Source directory (defaults to current directory)")
	flag.StringVar(&sourceDir, "src", "", "Source directory (shorthand)")
	flag.StringVar(&sourceDir, "s", "", "Source directory (shorthand)")

	flag.StringVar(&targetDir, "target", "", "Target directory (defaults to home directory)")
	flag.StringVar(&targetDir, "dest", "", "Target directory (alias)")
	flag.StringVar(&targetDir, "t", "", "Target directory (shorthand)")
}

func main() {
	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dloom - Dotfile manager and system bootstrapper\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  dloom [options] <command> [packages...]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  link      Create symlinks for specified packages\n")
		fmt.Fprintf(os.Stderr, "  unlink    Remove symlinks for specified packages\n")
		fmt.Fprintf(os.Stderr, "  setup     Run specified setup scripts\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  dloom link vim bash        # Link vim and bash packages\n")
		fmt.Fprintf(os.Stderr, "  dloom -v -d link vim       # Verbose dry-run for vim package\n")
	}

	// Parse flags
	flag.Parse()

	// Need at least one argument (the command)
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	// initialized the logger
	logger := logging.Logger{
		UseColors: !noColor,
	}

	// Load config
	cfg, err := internal.LoadConfig(configPath, &logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override config with command-line flags
	if force {
		cfg.Force = true
	}
	if verbose {
		cfg.Verbose = true
	}
	if dryRun {
		cfg.DryRun = true
	}
	if sourceDir != "" {
		cfg.SourceDir = sourceDir
	}
	if targetDir != "" {
		cfg.TargetDir = targetDir
	}

	// Get absolute path for source directory
	if !filepath.IsAbs(cfg.SourceDir) {
		absSourceDir, err := filepath.Abs(cfg.SourceDir)
		if err == nil {
			cfg.SourceDir = absSourceDir
		}
	}

	// Handle command
	command := args[0]
	cmdArgs := args[1:]

	var cmdErr error
	switch command {
	case "link":
		cmdErr = handleLink(cmdArgs, cfg, &logger)
	case "unlink":
		cmdErr = handleUnlink(cmdArgs, cfg, &logger)
	case "setup":
		cmdErr = handleSetup(cmdArgs, cfg, &logger)
	default:
		logger.LogError("Unknown command: %s", command)
		flag.Usage()
		os.Exit(1)
	}

	if cmdErr != nil {
		logger.LogError("Error: %v\n", cmdErr)
		os.Exit(1)
	}
}

// handleLink handles the "link" command
func handleLink(args []string, cfg *internal.Config, logger *logging.Logger) error {
	if len(args) == 0 {
		return errors.New("no packages specified for link command")
	}

	opts := internal.LinkOptions{
		Config:   cfg,
		Packages: args,
	}

	return internal.LinkPackages(opts, logger)
}

// handleUnlink handles the "unlink" command
func handleUnlink(args []string, cfg *internal.Config, logger *logging.Logger) error {
	if len(args) == 0 {
		return errors.New("no packages specified for unlink command\n" +
			"Use: dloom unlink <package>... or dloom -p <package>[,<package>...] unlink")
	}

	opts := internal.UnlinkOptions{
		Config:   cfg,
		Packages: args,
	}

	return internal.UnlinkPackages(opts, logger)
}

// handleSetup handles the "setup" command
func handleSetup(args []string, cfg *internal.Config, logger *logging.Logger) error {
	if len(args) == 0 {
		return errors.New("no scripts specified for setup command")
	}

	opts := internal.SetupOptions{
		Config:  cfg,
		Scripts: args,
	}

	return internal.RunScripts(opts, logger)
}

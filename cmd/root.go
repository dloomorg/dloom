package cmd

import (
	"fmt"
	"github.com/dloomorg/dloom/internal"
	"github.com/dloomorg/dloom/internal/logging"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var Version = "dev"

var (
	cfg    *internal.Config
	logger *logging.Logger

	// flags
	configPath string
	force      bool
	verbose    bool
	dryRun     bool
	sourceDir  string
	targetDir  string
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "dloom",
	Short: "Dotfile manager and system bootstrapper",
	RunE: func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Println("dloom version:", Version)
			return nil
		}
		// If no version flag, show help
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		logger = &logging.Logger{UseColors: !noColor}

		var err error
		cfg, err = internal.LoadConfig(configPath, logger)
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

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

		// Absolute source dir
		if !filepath.IsAbs(cfg.SourceDir) {
			if abs, err := filepath.Abs(cfg.SourceDir); err == nil {
				cfg.SourceDir = abs
			}
		}

		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP("version", "V", false, "Print the version of dloom and exit")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVarP(&force, "force", "f", false, "Force overwrite existing files")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run (show actions without executing)")
	rootCmd.PersistentFlags().StringVarP(&sourceDir, "source", "s", "", "Source directory")
	rootCmd.PersistentFlags().StringVarP(&targetDir, "target", "t", "", "Target directory")
}

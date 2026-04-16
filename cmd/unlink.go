package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink [packages/paths...]",
	Short: "Remove symlinks for specified packages",
	Long: `Remove symlinks for specified packages.

Arguments can be package names (e.g., vim, zsh) or paths:
  dloom unlink vim zsh        # Unlink specific packages
  dloom unlink .              # Unlink all packages in the current directory
  dloom unlink ~/dotfiles/    # Unlink all packages from a specific directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no packages specified for unlink command")
		}

		packages, effectiveCfg, err := internal.ResolveArgs(args, cfg, logger)
		if err != nil {
			return err
		}

		if len(packages) == 0 {
			logger.LogWarning("No packages found to unlink")
			return nil
		}

		opts := internal.UnlinkOptions{
			Config:   effectiveCfg,
			Packages: packages,
		}

		return internal.UnlinkPackages(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

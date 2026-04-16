package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [packages/paths...]",
	Short: "Create symlinks for specified packages",
	Long: `Create symlinks for specified packages.

Arguments can be package names (e.g., vim, zsh) or paths:
  dloom link vim zsh        # Link specific packages
  dloom link .              # Link all packages in the current directory
  dloom link ~/dotfiles/    # Link all packages from a specific directory`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no packages specified for link command")
		}

		packages, effectiveCfg, err := internal.ResolveArgs(args, cfg, logger)
		if err != nil {
			return err
		}

		if len(packages) == 0 {
			logger.LogWarning("No packages found to link")
			return nil
		}

		opts := internal.LinkOptions{
			Config:   effectiveCfg,
			Packages: packages,
		}

		return internal.LinkPackages(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}

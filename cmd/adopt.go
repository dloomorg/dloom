package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var adoptInteractive bool

var adoptCmd = &cobra.Command{
	Use:   "adopt <package> <target-path...>",
	Short: "Adopt existing files into a package and symlink them back",
	Long: `Adopt existing files into a package and replace them with symlinks.

The first argument must be a package name. Remaining arguments are existing
target files or directories that should be moved into the package source tree.
Existing symlinks are skipped.

Examples:
  dloom adopt zsh ~/.zshrc
  dloom adopt ghostty ~/.config/ghostty
  dloom -d adopt git ~/.gitconfig ~/.config/git
  dloom adopt -i hypr ~/.config/hypr`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("adopt requires a package name and at least one target path")
		}

		pkg := args[0]
		if internal.ClassifyArg(pkg) != internal.ArgBare {
			return errors.New("adopt requires a package name as the first argument")
		}

		opts := internal.AdoptOptions{
			Config:      cfg,
			Package:     pkg,
			Targets:     args[1:],
			Interactive: adoptInteractive,
		}

		return internal.AdoptPackage(opts, logger)
	},
}

func init() {
	adoptCmd.Flags().BoolVarP(&adoptInteractive, "interactive", "i", false, "Prompt before adopting each file")
	rootCmd.AddCommand(adoptCmd)
}

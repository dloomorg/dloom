package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link [packages...]",
	Short: "Create symlinks for specified packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no packages specified for link command")
		}

		opts := internal.LinkOptions{
			Config:   cfg,
			Packages: args,
		}

		return internal.LinkPackages(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}

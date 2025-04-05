package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink [packages...]",
	Short: "Remove symlinks for specified packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no packages specified for unlink command")
		}

		opts := internal.UnlinkOptions{
			Config:   cfg,
			Packages: args,
		}

		return internal.UnlinkPackages(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(unlinkCmd)
}

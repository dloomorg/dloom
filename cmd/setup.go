package cmd

import (
	"errors"
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup [scripts...]",
	Short: "Run specified setup scripts",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("no scripts specified for setup command")
		}

		opts := internal.SetupOptions{
			Config:  cfg,
			Scripts: args,
		}

		return internal.RunScripts(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

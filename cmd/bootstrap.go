package cmd

import (
	"github.com/dloomorg/dloom/internal"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap [repository-url|directory]",
	Short: "Clone a dotfile repository and install git if needed, or bootstrap system with an existing directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		opts := internal.BootstrapOptions{
			Config:   cfg,
			Target: target,
		}
		return internal.Bootstrap(opts, logger)
	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
} 
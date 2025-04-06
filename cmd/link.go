package cmd

import (
	"errors"
	"fmt"
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

		if err := internal.LinkPackages(opts, logger); err != nil {
			return err
		}

		if postCmd != "" {
			if err := internal.RunShellCommand(cfg, postCmd, logger); err != nil {
				return fmt.Errorf("post command failed: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}

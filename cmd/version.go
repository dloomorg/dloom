package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of dloom",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("dloom version: ", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

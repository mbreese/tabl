package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of tabgo",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("tabgo tab file utility v0.0.1")
	},
}

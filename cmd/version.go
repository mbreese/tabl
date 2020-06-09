package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var help bool

func init() {
	versionCmd.Flags().BoolVarP(&help, "help", "h", false, "Show help")
	versionCmd.Flags().MarkHidden("help")
	//rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tabl version %s\n", rootCmd.Version)
	},
	DisableFlagsInUseLine: true,
}

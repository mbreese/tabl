package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mbreese/tabgo/textfile"
	"github.com/spf13/cobra"
)

func init() {
	// lessCmd.Flags().BoolVarP(&ShowLineNum, "show-linenum", "L", false, "Show line number")
	lessCmd.Flags().BoolVar(&NoHeader, "noheader", false, "File has no header")
	lessCmd.Flags().IntVar(&MinWidth, "min-width", 0, "Minimum column width")
	lessCmd.Flags().IntVar(&MaxWidth, "max-width", 0, "Maximum column width")
	rootCmd.AddCommand(lessCmd)
}

var lessCmd = &cobra.Command{
	Use:   "less",
	Short: "Page through a tab-delimited text file",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("Missing filename")
		}
		_, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return fmt.Errorf("Missing file: %s", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		txt := textfile.NewTabFile(args[0])
		textfile.NewTextPager(txt).
			WithShowLineNum(ShowLineNum).
			WithMaxWidth(MaxWidth).
			WithMinWidth(MinWidth).
			WithHasHeader(!NoHeader).
			Show()
	},
}
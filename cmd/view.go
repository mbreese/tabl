package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mbreese/tabgo/textfile"
	"github.com/spf13/cobra"
)

// NoHeader -- the file has no header
var NoHeader bool

// ShowComments -- include the heading comments in the output
var ShowComments bool

// ShowLineNum -- include the line number in the output
var ShowLineNum bool

// MinWidth -- minimum column width
var MinWidth int = 0

// MaxWidth -- minimum column width
var MaxWidth int = 0

func init() {
	viewCmd.Flags().BoolVarP(&ShowComments, "show-comments", "H", false, "Show comments")
	viewCmd.Flags().BoolVarP(&ShowLineNum, "show-linenum", "L", false, "Show line number")
	viewCmd.Flags().BoolVar(&NoHeader, "no-header", false, "File has no header")
	viewCmd.Flags().IntVar(&MinWidth, "min-width", 0, "Minimum column width")
	viewCmd.Flags().IntVar(&MaxWidth, "max-width", 0, "Maximum column width")
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View a tab-delimited text file",
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
		textfile.NewTextViewer(txt).
			WithShowComments(ShowComments).
			WithShowLineNum(ShowLineNum).
			WithMaxWidth(MaxWidth).
			WithMinWidth(MinWidth).
			WithHasHeader(!NoHeader).
			WriteFile(os.Stdout)
	},
}
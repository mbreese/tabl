package cmd

import (
	"fmt"
	"os"

	"github.com/mbreese/tabl/textfile"
	"github.com/spf13/cobra"
)

// IsCSV -- the file is a CSV file
var IsCSV bool

// NoHeader -- the file has no header
var NoHeader bool

// HeaderComment -- the header is the last commented line
var HeaderComment bool

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
	viewCmd.Flags().BoolVar(&IsCSV, "csv", false, "The file is a CSV file")
	//viewCmd.Flags().BoolVar(&HeaderComment, "header-comment", false, "The header is the last commented line")
	//viewCmd.Flags().BoolVar(&NoHeader, "no-header", false, "File has no header")
	viewCmd.Flags().IntVar(&MinWidth, "min", 0, "Minimum column width")
	viewCmd.Flags().IntVar(&MaxWidth, "max", 0, "Maximum column width")
	rootCmd.AddCommand(viewCmd)
}

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "Pretty-print of a tabular file",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] != "-" {
			_, err := os.Stat(args[0])
			if os.IsNotExist(err) {
				return fmt.Errorf("Missing file: %s", args[0])
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			args = []string{"-"}
		}
		var txt *textfile.DelimitedTextFile
		if !IsCSV {
			txt = textfile.NewTabFile(args[0])
		} else {
			txt = textfile.NewCSVFile(args[0])
		}

		// by default we won't process headers as special in the "view" mode
		txt = txt.WithNoHeader(true)

		textfile.NewTextViewer(txt).
			WithShowComments(ShowComments).
			WithShowLineNum(ShowLineNum).
			WithMaxWidth(MaxWidth).
			WithMinWidth(MinWidth).
			WriteFile(os.Stdout)
	},
}

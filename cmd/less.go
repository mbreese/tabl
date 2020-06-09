package cmd

import (
	"fmt"
	"os"

	"github.com/mbreese/tabl/textfile"
	"github.com/spf13/cobra"
)

func init() {
	// lessCmd.Flags().BoolVarP(&ShowLineNum, "show-linenum", "L", false, "Show line number")
	lessCmd.Flags().BoolVar(&NoHeader, "no-header", false, "File has no header")
	lessCmd.Flags().BoolVar(&HeaderComment, "header-comment", false, "The header is the last commented line")
	lessCmd.Flags().BoolVar(&IsCSV, "csv", false, "The file is a CSV file")
	lessCmd.Flags().IntVar(&MinWidth, "min", 0, "Minimum column width")
	lessCmd.Flags().IntVar(&MaxWidth, "max", 0, "Maximum column width")
	rootCmd.AddCommand(lessCmd)
}

var lessCmd = &cobra.Command{
	Use:   "less",
	Short: "Page through a tab-delimited text file",
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

		txt = txt.WithNoHeader(NoHeader).
			WithHeaderComment(HeaderComment)

		textfile.NewTextPager(txt).
			WithShowLineNum(ShowLineNum).
			WithMaxWidth(MaxWidth).
			WithMinWidth(MinWidth).
			Show()
	},
}

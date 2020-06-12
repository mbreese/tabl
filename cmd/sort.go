package cmd

import (
	"fmt"
	"os"

	"github.com/mbreese/tabl/textfile"
	"github.com/spf13/cobra"
)

var sortCols MultiColumnVar

func init() {
	sortCmd.Flags().BoolVarP(&ShowComments, "show-comments", "H", false, "Show comments")
	sortCmd.Flags().BoolVar(&IsCSV, "csv", false, "The file is a CSV file")
	sortCmd.Flags().BoolVar(&HeaderComment, "header-comment", false, "The header is the last commented line")
	sortCmd.Flags().BoolVar(&NoHeader, "no-header", false, "File has no header")
	sortCmd.Flags().VarP(&sortCols, "key", "k", "Columns to sort by (multiple allowed, comma separated, end with ':n' for numeric sort, ':r' for reverse sort)")
	// exportCmd.Flags().StringVar(&ExportCols, "cols", "", "Columns to export (comma separated, names or indexes, requried)")

	// sortCmd.MarkFlagRequired("key")

	rootCmd.AddCommand(sortCmd)
}

var sortCmd = &cobra.Command{
	Use:   "sort [file]",
	Short: "Sort a file by columns",
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
		if len(sortCols.Values) == 0 {
			// TODO: make the default sort by all columns in text mode
			fmt.Fprintln(os.Stderr, "Missing value for --key (at least one column to sort by is required)")
			return
		}

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
		txt = txt.WithNoHeader(NoHeader).WithHeaderComment(HeaderComment)

		err := textfile.NewTextSorter(txt, sortCols.Values).
			WithShowComments(ShowComments).
			WriteFile(os.Stdout)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	},
}

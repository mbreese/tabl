package cmd

import (
	"fmt"
	"os"

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

var (
	rootCmd = &cobra.Command{
		Use:     "tabl",
		Short:   "Utilities for working with tab-delimited text files",
		Version: "0.1.3",
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

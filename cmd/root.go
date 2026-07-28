package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/compgen-io/tabl/textfile"
	"github.com/spf13/cobra"
)

// IsCSV -- the file is a CSV file
var IsCSV bool

// IsParquet -- the file is a Parquet file
var IsParquet bool

// NAString -- the value written for NULL cells (Parquet only)
var NAString string

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

// parquetMagic is the marker written at the start (and end) of a parquet file
var parquetMagic = []byte("PAR1")

// looksLikeParquet sniffs the leading magic bytes, in the same spirit as the
// gzip detection the delimited reader already does.
func looksLikeParquet(fname string) bool {
	if fname == "-" {
		return false
	}

	f, err := os.Open(fname)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, len(parquetMagic))
	if _, err := io.ReadFull(f, buf); err != nil {
		return false
	}
	return bytes.Equal(buf, parquetMagic)
}

// openReader builds the appropriate reader for a file. --parquet and --csv
// force the format; otherwise we sniff for parquet and fall back to text.
func openReader(cmd *cobra.Command, fname string) (textfile.RecordReader, error) {
	if IsParquet || (!IsCSV && looksLikeParquet(fname)) {
		if err := rejectTextOnlyFlags(cmd); err != nil {
			return nil, err
		}
		pq := textfile.NewParquetFile(fname).WithNAString(NAString)
		// open now so a bad path or corrupt file is reported before we start
		// writing output
		if err := pq.Open(); err != nil {
			return nil, err
		}
		return pq, nil
	}

	var txt *textfile.DelimitedTextFile
	if IsCSV {
		txt = textfile.NewCSVFile(fname)
	} else {
		txt = textfile.NewTabFile(fname)
	}

	return txt.WithNoHeader(NoHeader).WithHeaderComment(HeaderComment), nil
}

// rejectTextOnlyFlags errors on flags that mean nothing for parquet input. We
// check Changed so that only flags the user actually typed complain.
func rejectTextOnlyFlags(cmd *cobra.Command) error {
	for _, name := range []string{"csv", "no-header", "header-comment", "show-comments"} {
		if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
			return fmt.Errorf("--%s cannot be used with parquet input", name)
		}
	}
	return nil
}

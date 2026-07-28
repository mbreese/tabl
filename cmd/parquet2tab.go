package cmd

import (
	"fmt"
	"os"

	"github.com/compgen-io/tabl/textfile"
	"github.com/spf13/cobra"
)

func init() {
	parquet2TabCmd.Flags().StringVar(&NAString, "na", "", "Value to write for NULL cells")
	rootCmd.AddCommand(parquet2TabCmd)
}

var parquet2TabCmd = &cobra.Command{
	Use:   "parquet2tab [file]",
	Short: "Convert a Parquet file to tab-delimited format",
	Long: `Convert a Parquet file to tab-delimited format.

The parquet schema becomes the header. Nested structs are flattened into
dotted column names (addr.city), and lists and maps are written as JSON.

NULL cells are written as an empty value unless --na is given.

Note that parquet files can't be read from a pipe -- the schema lives in a
footer at the end of the file, so a real file path is required.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Missing [file]")
		}
		if args[0] == "-" {
			return textfile.ErrParquetStdin
		}
		_, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return fmt.Errorf("Missing file: %s", args[0])
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		pq := textfile.NewParquetFile(args[0]).WithNAString(NAString)
		if err := pq.Open(); err != nil {
			er(err)
		}

		if err := textfile.NewCSVExporter(pq).WriteFile(os.Stdout); err != nil {
			er(err)
		}
	},
}

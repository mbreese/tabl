package cmd

import (
	"container/list"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mbreese/tabl/textfile"
	"github.com/spf13/cobra"
)

func init() {
	exportCmd.Flags().BoolVarP(&ShowComments, "show-comments", "H", false, "Show comments")
	exportCmd.Flags().BoolVar(&IsCSV, "csv", false, "The file is a CSV file")
	exportCmd.Flags().BoolVar(&HeaderComment, "header-comment", false, "The header is the last commented line")
	exportCmd.Flags().BoolVar(&NoHeader, "no-header", false, "File has no header")
	// exportCmd.Flags().StringVar(&ExportCols, "cols", "", "Columns to export (comma separated, names or indexes, requried)")
	rootCmd.MarkFlagRequired("cols")

	exportCmd.Flags()
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export [cols] [file]",
	Short: "Extract columns from a tabular file",
	Args: func(cmd *cobra.Command, args []string) error {

		if len(args) == 0 {
			return fmt.Errorf("Missing [cols] and [file]")
		}

		if len(args) > 1 && args[1] != "-" {
			_, err := os.Stat(args[1])
			if os.IsNotExist(err) {
				return fmt.Errorf("Missing file: %s", args[1])
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 1 {
			args = []string{args[0], "-"}
		}
		var txt *textfile.DelimitedTextFile
		if !IsCSV {
			txt = textfile.NewTabFile(args[1])
		} else {
			txt = textfile.NewCSVFile(args[1])
		}

		cols, err := ParseColumnList(args[0])
		if err != nil {
			panic(err)
		}

		// by default we won't process headers as special in the "view" mode
		txt = txt.WithNoHeader(NoHeader).WithHeaderComment(HeaderComment)

		err = textfile.NewTextExporter(txt, cols).
			WithShowComments(ShowComments).
			WriteFile(os.Stdout)

		if err != nil {
			panic(err)
		}
	},
}

// ParseColumnList will take a comma-separated list of columns and parse that into a list of TextColumn object
// Examples of lists:
//   1,2,4,5
//   1,"gene name",description
//   3-5,7
//
// Named columns won't be converted to their index values until needed. Column numbers should be 1-based for input, but
// will be 0-based for the TextColumn index.
func ParseColumnList(buf string) ([]*textfile.TextColumn, error) {
	var sb strings.Builder
	colList := list.New()
	colNum := list.New()
	inQuote := false
	singleQuote := false
	isNumber := true

	for len(buf) > 0 {
		r, l := utf8.DecodeRuneInString(buf)

		if r == utf8.RuneError {
			panic(fmt.Errorf("Error processing column list: %s", buf))
		}

		if inQuote {
			if r == '"' && !singleQuote {
				inQuote = false
			} else if r == '\'' && singleQuote {
				inQuote = false
			} else {
				sb.WriteRune(r)
			}
		} else if r == '"' {
			inQuote = true
			isNumber = false
			singleQuote = false
		} else if r == '\'' {
			inQuote = true
			isNumber = false
			singleQuote = true
		} else if r == ',' {
			colList.PushBack(sb.String())
			colNum.PushBack(isNumber)
			sb.Reset()
			isNumber = true
		} else {
			if strings.IndexRune("0123456789-", r) == -1 {
				isNumber = false
			}
			sb.WriteRune(r)
		}
		buf = buf[l:]
	}
	if sb.Len() > 0 {
		colList.PushBack(sb.String())
		colNum.PushBack(isNumber)
	}

	outList := list.New()

	e := colList.Front()
	e2 := colNum.Front()

	for e != nil {
		s, _ := e.Value.(string)
		isNum, _ := e2.Value.(bool)

		if isNum {
			if strings.IndexRune(s, '-') == -1 {
				// this is just a simple number
				val, err := strconv.Atoi(s)
				if err != nil {
					return nil, err
				}
				outList.PushBack(textfile.NewIndexColumn(val - 1))

			} else {
				// this is a range ex: 1-3
				one := s[0:strings.IndexRune(s, '-')]
				two := s[strings.IndexRune(s, '-')+1:]

				val1, err1 := strconv.Atoi(one)
				val2, err2 := strconv.Atoi(two)

				if err1 != nil {
					return nil, err1
				}
				if err2 != nil {
					return nil, err2
				}

				for j := val1; j <= val2; j++ {
					outList.PushBack(textfile.NewIndexColumn(j - 1))
				}
			}
		} else {
			outList.PushBack(textfile.NewNamedColumn(s))
		}
		e = e.Next()
		e2 = e2.Next()
	}

	cols := make([]*textfile.TextColumn, outList.Len())
	el := outList.Front()
	for i := 0; i < len(cols); i++ {
		v, _ := el.Value.(*textfile.TextColumn)
		cols[i] = v
		el = el.Next()
	}

	return cols, nil
}

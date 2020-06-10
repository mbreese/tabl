package textfile

import (
	"fmt"
	"io"
	"strings"
)

// TextExporter is used to export specific columns from a tab delimited file
type TextExporter struct {
	txt          *DelimitedTextFile
	cols         []*TextColumn
	showComments bool
}

// TextColumn - the column to export. Initially, the idx is set to -1 for named columns.
type TextColumn struct {
	name string // the name is only used to then find the index
	idx  int    // this value is -1 when starting for a named column.
}

//String - write TextColumn as a string
func (col *TextColumn) String() string {
	if col.idx == -1 {
		return col.name
	}

	return fmt.Sprintf("idx:%d", col.idx)
}

// NewNamedColumn - the column to export. For columns specified by name, idx should initially be -1.
// It will be populated then on the first pass.
func NewNamedColumn(name string) *TextColumn {
	return &TextColumn{
		name: name,
		idx:  -1,
	}
}

// NewIndexColumn - the column to export. For columns defined by index, idx is the column number (0-based).
func NewIndexColumn(idx int) *TextColumn {
	return &TextColumn{
		name: "",
		idx:  idx,
	}
}

// NewTextExporter - create a new text exporter
func NewTextExporter(f *DelimitedTextFile, cols []*TextColumn) *TextExporter {
	return &TextExporter{
		txt:          f,
		cols:         cols,
		showComments: false,
	}
}

// WithShowComments - set showing comments
func (tex *TextExporter) WithShowComments(b bool) *TextExporter {
	tex.showComments = b
	return tex
}

// WriteFile - write the selected columns to the given stream
func (tex *TextExporter) WriteFile(out io.Writer) error {
	var line *TextRecord
	var err error = nil
	wroteHeader := false

	for err == nil {
		line, err = tex.txt.ReadLine()
		if err != nil {
			break
		}

		if line.Values == nil {
			// comment
			if tex.showComments {
				fmt.Fprintln(out, strings.TrimSuffix(strings.TrimSuffix(line.RawString, "\n"), "\r"))
			}
			continue
		}

		if !wroteHeader {
			err := tex.populateColIndex()
			if err != nil {
				return err
			}
			if !tex.txt.noHeader {
				err := tex.writeHeader(out)
				if err != nil {
					return err
				}
			}
			wroteHeader = true
		}
		err := tex.writeLine(out, line)
		if err != nil {
			return err
		}
	}

	tex.txt.Close()
	return nil

}

func (tex *TextExporter) populateColIndex() error {
	for _, col := range tex.cols {
		if col.idx == -1 {
			for i, header := range tex.txt.Header {
				if header == col.name {
					col.idx = i
					break
				}
			}
			if col.idx == -1 {
				return fmt.Errorf("Missing column: %s\n\nHave:%v\nHeaderComment: %v", col.name, tex.txt.Header, tex.txt.headerComment)
			}
		}
	}
	return nil
}
func (tex *TextExporter) writeHeader(out io.Writer) error {
	for i, col := range tex.cols {
		if i > 0 {
			fmt.Fprint(out, "\t")
		}
		if col.idx >= len(tex.txt.Header) {
			fmt.Fprint(out, "\n")
			return fmt.Errorf("Column index out of bounds: %d", col.idx+1)
		}
		fmt.Fprint(out, tex.txt.Header[col.idx])
	}
	fmt.Fprint(out, "\n")

	return nil
}

func (tex *TextExporter) writeLine(out io.Writer, line *TextRecord) error {
	for i, col := range tex.cols {
		if i > 0 {
			fmt.Fprint(out, "\t")
		}
		if col.idx >= len(tex.txt.Header) {
			fmt.Fprint(out, "\n")
			return fmt.Errorf("Column index out of bounds: %d", col.idx+1)
		}
		fmt.Fprint(out, line.Values[col.idx])
	}
	fmt.Fprint(out, "\n")
	return nil
}

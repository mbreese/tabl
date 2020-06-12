package textfile

import (
	"fmt"
	"io"
)

// TextExporter is used to export specific columns from a tab delimited file
type TextExporter struct {
	txt          *DelimitedTextFile
	cols         []*TextColumn
	showComments bool
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
				fmt.Fprint(out, line.RawString)
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
			fmt.Fprint(out, tex.txt.Delim)
		}
		if col.idx >= len(tex.txt.Header) {
			if tex.txt.IsCrLf {
				fmt.Fprint(out, "\r")
			}
			fmt.Fprint(out, "\n")
			return fmt.Errorf("Column index out of bounds: %d", col.idx+1)
		}
		if tex.txt.Quote != 0 {
			fmt.Fprint(out, tex.txt.Quote)
		}
		fmt.Fprint(out, tex.txt.Header[col.idx])
		if tex.txt.Quote != 0 {
			fmt.Fprint(out, tex.txt.Quote)
		}
	}
	if tex.txt.IsCrLf {
		fmt.Fprint(out, "\r")
	}
	fmt.Fprint(out, "\n")

	return nil
}

func (tex *TextExporter) writeLine(out io.Writer, line *TextRecord) error {
	for i, col := range tex.cols {
		if i > 0 {
			fmt.Fprint(out, tex.txt.Delim)
		}
		if col.idx >= len(tex.txt.Header) {
			if tex.txt.IsCrLf {
				fmt.Fprint(out, "\r")
			}
			fmt.Fprint(out, "\n")
			return fmt.Errorf("Column index out of bounds: %d", col.idx+1)
		}
		if tex.txt.Quote != 0 {
			fmt.Fprint(out, tex.txt.Quote)
		}
		fmt.Fprint(out, line.Values[col.idx])
		if tex.txt.Quote != 0 {
			fmt.Fprint(out, tex.txt.Quote)
		}
	}
	if tex.txt.IsCrLf {
		fmt.Fprint(out, "\r")
	}
	fmt.Fprint(out, "\n")

	return nil
}

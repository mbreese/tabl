package textfile

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// CSVExporter is used to export specific columns from a tab delimited file
type CSVExporter struct {
	txt          *DelimitedTextFile
	showComments bool
}

// NewCSVExporter - create a new text exporter
func NewCSVExporter(f *DelimitedTextFile) *CSVExporter {
	return &CSVExporter{
		txt:          f,
		showComments: false,
	}
}

// WithShowComments - set showing comments
func (tex *CSVExporter) WithShowComments(b bool) *CSVExporter {
	tex.showComments = b
	return tex
}

// WriteFile - write the selected columns to the given stream
func (tex *CSVExporter) WriteFile(out io.Writer) error {
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

func (tex *CSVExporter) writeHeader(out io.Writer) error {
	if tex.txt.noHeader {
		return nil
	}
	for i, v := range tex.txt.Header {
		if i > 0 {
			fmt.Fprint(out, "\t")
		}
		fmt.Fprint(out, quoteTab(v))
	}
	fmt.Fprint(out, "\n")

	return nil
}

func (tex *CSVExporter) writeLine(out io.Writer, line *TextRecord) error {
	if tex.txt.noHeader {
		return nil
	}
	for i, v := range line.Values {
		if i > 0 {
			fmt.Fprint(out, "\t")
		}
		fmt.Fprint(out, quoteTab(v))
	}
	fmt.Fprint(out, "\n")
	return nil
}

func quoteTab(s string) string {
	var sb strings.Builder

	for len(s) > 0 {
		r, l := utf8.DecodeRuneInString(s)
		s = s[l:]

		switch r {
		case '\a':
			sb.WriteString("\\a")
		case '\b':
			sb.WriteString("\\b")
		case '\f':
			sb.WriteString("\\f")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		case '\v':
			sb.WriteString("\\v")
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()

}

package textfile

import (
	"container/list"
	"fmt"
	"io"
	"strings"

	"github.com/mbreese/tabgo/support"
)

const linesForEstimation int = 10000

// TextViewer is a viewer for tab-delimited data, it handles formatting and showing the data on a stream
type TextViewer struct {
	txt          *DelimitedTextFile
	showComments bool
	showLineNum  bool
	hasHeader    bool
	minWidth     int
	maxWidth     int
	colNames     []string
	colWidth     []int
}

// NewTextViewer - create a new text viewer
func NewTextViewer(f *DelimitedTextFile) *TextViewer {
	return &TextViewer{
		txt:          f,
		showComments: false,
		showLineNum:  false,
		hasHeader:    true,
		minWidth:     0,
		maxWidth:     0,
		colNames:     nil,
		colWidth:     nil,
	}
}

// WithHasHeader - set is there is a header
func (tv *TextViewer) WithHasHeader(b bool) *TextViewer {
	tv.hasHeader = b
	return tv
}

// WithShowLineNum - set showing line numbers
func (tv *TextViewer) WithShowLineNum(b bool) *TextViewer {
	tv.showLineNum = b
	return tv
}

// WithShowComments - set showing comments
func (tv *TextViewer) WithShowComments(b bool) *TextViewer {
	tv.showComments = b
	return tv
}

// WithMinWidth - set min column width
func (tv *TextViewer) WithMinWidth(i int) *TextViewer {
	tv.minWidth = i
	return tv
}

// WithMaxWidth - set max column width
func (tv *TextViewer) WithMaxWidth(i int) *TextViewer {
	tv.maxWidth = i
	return tv
}

// WriteFile - format and write a delimited text file to a stream
func (tv *TextViewer) WriteFile(out io.Writer) {
	var line *TextRecord
	var err error = nil

	lines := list.New()

	headerIdx := -1
	// we will need to auto-determine the column widths

	for i := 0; i < linesForEstimation; i++ {
		line, err = tv.txt.ReadLine()
		if err != nil {
			break
		}
		lines.PushBack(line)

		if line.Values == nil {
			continue
		}

		if tv.colNames == nil {
			if tv.hasHeader {
				headerIdx = i
			}
			tv.colNames = make([]string, len(line.Values))
			copy(tv.colNames, line.Values)
		}

		if len(tv.colNames) < len(line.Values) {
			newNames := make([]string, len(line.Values))
			for j := 0; j < len(newNames); j++ {
				newNames[j] = ""
			}
			copy(newNames, tv.colNames)
			tv.colNames = newNames
		}

		if tv.colWidth == nil {
			tv.colWidth = make([]int, len(line.Values))
			for j := 0; j < len(tv.colWidth); j++ {
				tv.colWidth[j] = 0
			}
		}
		if len(tv.colWidth) < len(line.Values) {
			newWidths := make([]int, len(line.Values))
			for j := 0; j < len(newWidths); j++ {
				newWidths[j] = 0
			}
			copy(newWidths, tv.colWidth)
			tv.colWidth = newWidths
		}
		for j, v := range line.Values {
			r := []rune(v)
			tv.colWidth[j] = support.MaxInt(tv.minWidth, tv.colWidth[j], len(r))
			if tv.maxWidth > 0 {
				tv.colWidth[j] = support.MinInt(tv.colWidth[j], tv.maxWidth)
			}
		}
	}

	e := lines.Front()
	for i := 0; i < lines.Len(); i++ {
		line, _ = e.Value.(*TextRecord)
		tv.writeLine(out, line, i == headerIdx)
		e = e.Next()
	}

	// var sb strings.Builder
	for err == nil {
		line, err = tv.txt.ReadLine()
		if err != nil {
			break
		}
		tv.writeLine(out, line, false)
	}

	tv.txt.Close()
}

func (tv *TextViewer) writeLine(out io.Writer, line *TextRecord, isHeader bool) {
	if line.Values == nil {
		if !tv.showComments {
			return
		}
		fmt.Fprintln(out, strings.TrimSuffix(strings.TrimSuffix(line.RawString, "\n"), "\r"))
		return
	}

	if tv.showLineNum {
		fmt.Fprintf(out, "[%d] ", line.DataLineNum)
	}

	for i, v := range line.Values {
		if i > 0 {
			fmt.Fprint(out, "| ")
		}

		r := []rune(v)

		s := fmt.Sprintf("%%-%ds", tv.colWidth[i])
		if len(r) <= tv.colWidth[i] {
			fmt.Fprintf(out, s+" ", string(r))
		} else {
			fmt.Fprintf(out, s+"$", string(r[:tv.colWidth[i]]))
		}
		//fmtFprintf(out, line.Values[i])
	}

	fmt.Fprint(out, "\n")
	if isHeader {
		for i := 0; i < len(line.Values); i++ {
			if i > 0 {
				fmt.Fprint(out, "-+-")
			}
			for j := 0; j < tv.colWidth[i]; j++ {
				fmt.Fprint(out, "-")
			}
		}
		fmt.Fprint(out, "-\n")
	}
}

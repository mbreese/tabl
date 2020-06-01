package textfile

import (
	"container/list"
	"log"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/mbreese/tabgo/support"
	tb "github.com/nsf/termbox-go"
)

const maxLines = 20000

// TextPager is a viewer for tab-delimited data, it handles formatting and showing the data on a stream
type TextPager struct {
	txt          *DelimitedTextFile
	showComments bool
	showLineNum  bool
	hasHeader    bool
	minWidth     int
	maxWidth     int
	colNames     []string
	colWidth     []int
	lines        *list.List
	topRow       *list.Element
	activeRow    int
	leftCol      int
	visibleRows  int
	visibleCols  int
}

// NewTextPager - create a new text viewer
func NewTextPager(f *DelimitedTextFile) *TextPager {
	return &TextPager{
		txt:          f,
		showComments: false,
		showLineNum:  false,
		hasHeader:    true,
		minWidth:     0,
		maxWidth:     0,
		colNames:     nil,
		colWidth:     nil,
		lines:        list.New(),
		activeRow:    1,
		leftCol:      0,
		visibleRows:  0,
		visibleCols:  0,
	}
}

// WithHasHeader - set is there is a header
func (tv *TextPager) WithHasHeader(b bool) *TextPager {
	tv.hasHeader = b
	return tv
}

// WithShowLineNum - set showing line numbers
func (tv *TextPager) WithShowLineNum(b bool) *TextPager {
	tv.showLineNum = b
	return tv
}

// WithMinWidth - set min column width
func (tv *TextPager) WithMinWidth(i int) *TextPager {
	tv.minWidth = i
	return tv
}

// WithMaxWidth - set max column width
func (tv *TextPager) WithMaxWidth(i int) *TextPager {
	tv.maxWidth = i
	return tv
}

func (tv *TextPager) load() {
	var line *TextRecord
	var err error = nil

	// headerIdx := -1
	// we will need to auto-determine the column widths

	for i := 0; i < linesForEstimation; i++ {
		line, err = tv.txt.ReadLine()
		if err != nil {
			break
		}

		if line.Values == nil {
			continue
		}

		if tv.colNames == nil {
			// if tv.hasHeader {
			// 	headerIdx = i
			// }
			tv.colNames = make([]string, len(line.Values))
			copy(tv.colNames, line.Values)
		} else if tv.colNames != nil || !tv.hasHeader {
			tv.lines.PushBack(line)
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
		for j := 0; j < len(line.Values); j++ {
			r := []rune(line.Values[j])
			tv.colWidth[j] = support.MaxInt(tv.minWidth, tv.colWidth[j], len(r))
			if tv.maxWidth > 0 {
				tv.colWidth[j] = support.MinInt(tv.colWidth[j], tv.maxWidth)
			}
		}
	}

	tv.topRow = tv.lines.Front()

	// e := lines.Front()
	// for i := 0; i < lines.Len(); i++ {
	// 	line, _ = e.Value.(*TextRecord)
	// 	tv.writeLine(out, line, i == headerIdx)
	// 	e = e.Next()
	// }

	// // var sb strings.Builder
	// for err == nil {
	// 	line, err = tv.txt.ReadLine()
	// 	if err != nil {
	// 		break
	// 	}
	// 	tv.writeLine(out, line, false)
	// }

}

// Show - format and write a delimited text file to a stream
func (tv *TextPager) Show() {

	tv.load()

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	width, height, _ := terminal.GetSize(0)

	tbl := widgets.NewTable()
	tbl.TextStyle = ui.NewStyle(ui.ColorWhite)
	tbl.SetRect(0, 0, width, height)
	tbl.RowSeparator = false
	tbl.FillRow = true
	tbl.Border = false

	tv.visibleRows = height
	tv.visibleCols = width
	tv.updateTable(tbl)
	ui.Render(tbl)

	p0 := widgets.NewParagraph()
	p0.SetRect(0, 0, width, 3)
	p0.Border = true

	inSearch := false
	query := ""

	lastMatchCol := 0

	for e := range ui.PollEvents() {
		// fmt.Printf("%v\n", e)
		if inSearch {
			switch e.ID {
			case "<C-c>", "<Escape>":
				tb.HideCursor()
				inSearch = false
				query = ""
				ui.Render(tbl)
			case "<Backspace>":
				if len(query) > 0 {
					query = query[:len(query)-1]
				}
				p0.Text = " Search: " + query
				tb.SetCursor(len(p0.Text)+1, 1)
				ui.Render(p0)
			case "<Enter>":
				tb.HideCursor()
				found := false
				origTop := tv.topRow
				origActive := tv.activeRow

				for e := tv.topRow; !found && e != nil; e = e.Next() {
					line, _ := e.Value.(*TextRecord)
					if line.Values != nil {
						for i, v := range line.Values {
							if e == tv.topRow && i <= lastMatchCol {
								continue
							}
							if strings.Contains(v, query) {
								found = true
								tv.leftCol = i
								tv.topRow = e
								tv.activeRow = 1
								lastMatchCol = i
								break
							}
						}
					}
					if !found && e.Next() == nil && !tv.txt.isEOF {
						// need to load more lines!
						// qfmt.Fprintln(os.Stderr, "Loading more lines")
						l, err := tv.txt.ReadLine()

						if err != nil {
							break
						}
						tv.lines.PushBack(l)

						// if we want to trim the buffer as we search... (but this limits what to do if the query isn't found)
						// for tv.lines.Len() > maxLines {
						// 	tv.lines.Remove(tv.lines.Front())
						// }
					}
				}
				if !found {
					p0.Text = " Not found!"
					ui.Render(p0)

					tv.topRow = origTop
					tv.activeRow = origActive
				} else {
					for tv.lines.Len() > maxLines {
						tv.lines.Remove(tv.lines.Front())
					}
					inSearch = false
					tv.updateTable(tbl)
					ui.Render(tbl)
				}

			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				tbl.SetRect(0, 0, payload.Width, payload.Height)
				tv.visibleRows = payload.Height
				tv.visibleCols = payload.Width

				tv.updateTable(tbl)
				ui.Render(tbl)

				p0.Text = " Search: " + query
				p0.SetRect(0, 0, tv.visibleCols, 3)
				tb.SetCursor(len(p0.Text)+1, 1)
				ui.Render(p0)
			default:
				if e.ID == "<Space>" || (e.ID[0:1] != "<" && e.ID[len(e.ID)-1:] != ">") {
					if e.ID == "<Space>" {
						query += " "
					} else {
						query += e.ID
					}
					p0.Text = " Search: " + query
					tb.SetCursor(len(p0.Text)+1, 1)
					ui.Render(p0)
				}
			}
		} else {
			switch e.ID {
			case "q", "<C-c>", "<Escape>":

				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				tbl.SetRect(0, 0, payload.Width, payload.Height)
				tv.visibleRows = payload.Height
				tv.visibleCols = payload.Width
				tv.updateTable(tbl)
				p0.SetRect(0, 0, tv.visibleCols, 3)
				ui.Render(tbl)
			case "<Space>":
				e := tv.topRow
				i := 0
				for i = 0; e.Next() != nil && i < tv.visibleRows-3; i++ {
					if e.Next() == nil && !tv.txt.isEOF {
						// need to load more lines!
						// qfmt.Fprintln(os.Stderr, "Loading more lines")
						l, err := tv.txt.ReadLine()
						if err != nil {
							break
						}
						tv.lines.PushBack(l)
					}
					if e.Next() != nil {
						e = e.Next()
					}
				}

				tv.topRow = e
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "b":
				e := tv.topRow
				i := 0
				for i = 0; e.Prev() != nil && i < tv.visibleRows-3; i++ {
					if e.Prev() != nil {
						e = e.Prev()
					}
				}

				tv.topRow = e
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "j", "<Down>":
				tv.activeRow++

				maxActiveRow := 0

				for last := tv.topRow; last.Next() != nil && maxActiveRow < tv.visibleRows-3; last = last.Next() {
					maxActiveRow++
				}

				if tv.activeRow > maxActiveRow {
					tv.activeRow = maxActiveRow
					if tv.topRow.Next() != nil {
						tv.topRow = tv.topRow.Next()
					}
				}

				tv.updateTable(tbl)
				ui.Render(tbl)
				for tv.lines.Len() > maxLines {
					tv.lines.Remove(tv.lines.Front())
				}
			case "k", "<Up>":
				tv.activeRow--
				if tv.activeRow < 1 {
					tv.activeRow = 1

					if tv.topRow.Prev() != nil {
						tv.topRow = tv.topRow.Prev()
					}
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "<Right>":
				tv.leftCol++
				if tv.leftCol >= len(tv.colWidth) {
					tv.leftCol = len(tv.colWidth) - 1
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "<Left>":
				tv.leftCol--
				if tv.leftCol < 0 {
					tv.leftCol = 0
				}
				tv.updateTable(tbl)
				ui.Render(tbl)

			case "/":
				p0.Text = " Search: " + query
				tb.SetCursor(len(p0.Text)+1, 1)

				ui.Render(p0)
				inSearch = true

				// default:
				// 	tbl.Rows[1][0] = fmt.Sprintf("%s", e.ID)
				// 	ui.Render(tbl)
			}
		}
		// switch e.Type {
		// case ui.KeyboardEvent: // handle all key presses
		// ui.Render(p)
		// }
	}
}

var defaultStyle ui.Style = ui.NewStyle(ui.ColorClear)
var activeStyle ui.Style = ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold|ui.ModifierReverse)
var headerStyle ui.Style = ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold|ui.ModifierUnderline)

func (tv *TextPager) updateTable(tbl *widgets.Table) {

	size := 1
	rightCol := tv.leftCol

	for rightCol < len(tv.colNames) && size < tv.visibleCols {
		size += tv.colWidth[rightCol] + 1
		rightCol++
	}

	tbl.Rows = make([][]string, tv.visibleRows-2)
	tbl.RowStyles = make(map[int]ui.Style)

	headerVals := make([]string, (rightCol - tv.leftCol))
	for i, v := range tv.colNames[tv.leftCol:rightCol] {
		headerVals[i] = v
	}

	widths := make([]int, (rightCol - tv.leftCol))
	for i, v := range tv.colWidth[tv.leftCol:rightCol] {
		widths[i] = v + 1
	}

	tbl.ColumnWidths = widths
	tbl.Rows[0] = headerVals
	tbl.RowStyles[0] = headerStyle

	e := tv.topRow
	setActive := false
	lastIdx := 1

	for i := 1; e != nil && i < len(tbl.Rows); i++ {
		lastIdx = i
		vals := make([]string, (rightCol - tv.leftCol))
		line, _ := e.Value.(*TextRecord)

		for j, v := range line.Values[tv.leftCol:rightCol] {
			vals[j] = v
		}

		tbl.Rows[i] = vals
		if i == tv.activeRow {
			tbl.RowStyles[i] = activeStyle
			setActive = true
		} else {
			tbl.RowStyles[i] = defaultStyle
		}

		if e.Next() == nil && !tv.txt.isEOF {
			// need to load more lines!
			// qfmt.Fprintln(os.Stderr, "Loading more lines")
			l, err := tv.txt.ReadLine()
			if err != nil {
				break
			}
			tv.lines.PushBack(l)
		}
		e = e.Next()
	}
	if !setActive {
		tbl.RowStyles[lastIdx] = activeStyle
		tv.activeRow = lastIdx
	}

}

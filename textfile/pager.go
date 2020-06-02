package textfile

import (
	"container/list"
	"log"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/mbreese/tabgo/support"
	tb "github.com/nsf/termbox-go"
)

const maxLines = 20000

// TextPager is a viewer for tab-delimited data, it handles formatting and showing the data on a stream
type TextPager struct {
	txt           *DelimitedTextFile
	showComments  bool
	showLineNum   bool
	hasHeader     bool
	minWidth      int
	maxWidth      int
	colNames      []string
	colWidth      []int
	lines         *list.List
	topRow        *list.Element
	activeRow     int
	leftCol       int
	visibleRows   int
	visibleCols   int
	colSticky     []bool
	colSelectMode bool
	activeCol     int
}

// NewTextPager - create a new text viewer
func NewTextPager(f *DelimitedTextFile) *TextPager {
	return &TextPager{
		txt:           f,
		showComments:  false,
		showLineNum:   false,
		hasHeader:     true,
		minWidth:      0,
		maxWidth:      0,
		colNames:      nil,
		colWidth:      nil,
		lines:         list.New(),
		activeRow:     1,
		leftCol:       0,
		visibleRows:   0,
		visibleCols:   0,
		colSticky:     nil,
		colSelectMode: false,
		activeCol:     0,
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

		if tv.colSticky == nil {
			tv.colSticky = make([]bool, len(line.Values))
		}

		for j := 0; j < len(line.Values); j++ {
			r := []rune(line.Values[j] + "   ")
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

	// width, height, err := terminal.GetSize(0)
	// if err != nil {
	// 	panic(err)
	// }

	width, height := ui.TerminalDimensions()

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

	p1 := widgets.NewParagraph()
	p1.SetRect(0, 0, 50, 20)
	p1.Border = true
	p1.Text = `[tabgo                                       help](mod:reverse)
------------------------------------------------
q,Ctrl-C,ESC      Quit the program
/                 Search
m                 Mark a line
x                 Select "sticky" columns

                  To select sticky columns, use 
                  arrow keys and hit space to 
                  toggle on/off.

[Navigation]
h,left-arrow      Move left a column  
l,right-arrow     Move right a column  
j,down-arrow      Move down a row
k,up-arrow        Move up a row  

ESC to hide help text
`

	state := "view"
	query := ""

	lastMatchCol := 0

	for e := range ui.PollEvents() {
		// fmt.Printf("%v\n", e)
		if state == "help" {
			switch e.ID {
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				tbl.SetRect(0, 0, payload.Width, payload.Height)
				tv.visibleRows = payload.Height
				tv.visibleCols = payload.Width
				tv.updateTable(tbl)
				ui.Render(tbl)

				p0.Text = " Search: " + query
				p0.SetRect(0, 0, tv.visibleCols, 3)

				ui.Render(p1)
			case "q", "<Escape>":
				state = "view"
				ui.Render(tbl)
			}
		} else if state == "search" {
			switch e.ID {
			case "<C-c>", "<Escape>":
				tb.HideCursor()
				state = "view"
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
					state = "view"
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
		} else if state == "select" {
			switch e.ID {
			case "q", "<Escape>":
				state = "view"
				tv.colSelectMode = false
				if tv.leftCol < 0 {
					tv.leftCol = 0
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				tbl.SetRect(0, 0, payload.Width, payload.Height)
				tv.visibleRows = payload.Height
				tv.visibleCols = payload.Width
				tv.updateTable(tbl)
				p0.SetRect(0, 0, tv.visibleCols, 3)
				ui.Render(tbl)
			case "x", "<Space>":
				j := -support.BoolSum(tv.colSticky)
				found := false
				for i, v := range tv.colSticky {
					if v {
						if j == tv.leftCol {
							tv.colSticky[i] = !tv.colSticky[i]
							found = true
							break
						}
						j++
					}
				}

				if !found {
					for i, v := range tv.colSticky {
						if !v {
							if j == tv.leftCol {
								tv.colSticky[i] = !tv.colSticky[i]
								found = true
								break
							}
							j++
						}
					}
				}

				tv.updateTable(tbl)
				ui.Render(tbl)
			case "j", "<Down>":
				state = "view"
				tv.colSelectMode = false
				if tv.leftCol < 0 {
					tv.leftCol = 0
				}

				tv.updateTable(tbl)
				ui.Render(tbl)
			case "l", "<Right>":
				tv.leftCol++
				if tv.leftCol >= len(tv.colWidth)-support.BoolSum(tv.colSticky) {
					tv.leftCol = len(tv.colWidth) - support.BoolSum(tv.colSticky) - 1
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "h", "<Left>":
				tv.leftCol--
				if tv.leftCol < -support.BoolSum(tv.colSticky) {
					tv.leftCol = -support.BoolSum(tv.colSticky)
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			}
		} else {
			switch e.ID {
			case "q", "<Escape>", "<C-c>":
				// quit
				return
			case "?":
				state = "help"
				ui.Render(p1)
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				tbl.SetRect(0, 0, payload.Width, payload.Height)
				tv.visibleRows = payload.Height
				tv.visibleCols = payload.Width
				tv.updateTable(tbl)
				p0.SetRect(0, 0, tv.visibleCols, 3)
				ui.Render(tbl)
			case "<Space>":
				// down a page
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
			case "m":
				// mark row
				e := tv.topRow
				for i := 0; e.Next() != nil && i < tv.activeRow-1; i++ {
					e = e.Next()
				}

				t, _ := e.Value.(*TextRecord)
				t.Flag = !t.Flag
				tv.updateTable(tbl)
				ui.Render(tbl)

			case "b":
				// back a page
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
				// down a line
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
				// up a line
				tv.activeRow--
				if tv.activeRow < 1 {
					tv.activeRow = 1

					if tv.topRow.Prev() != nil {
						tv.topRow = tv.topRow.Prev()
					}
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "l", "<Right>":
				// right a col
				tv.leftCol++
				if tv.leftCol >= len(tv.colWidth)-support.BoolSum(tv.colSticky) {
					tv.leftCol = len(tv.colWidth) - support.BoolSum(tv.colSticky) - 1
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "h", "<Left>":
				// left a col
				tv.leftCol--
				if tv.leftCol < 0 {
					tv.leftCol = 0
				}
				tv.updateTable(tbl)
				ui.Render(tbl)
			case "x":
				state = "select"
				tv.colSelectMode = true
				if support.BoolSum(tv.colSticky) > 0 {
					tv.leftCol = 0
				}

				tv.updateTable(tbl)
				ui.Render(tbl)
			case "/":
				p0.Text = " Search: " + query
				tb.SetCursor(len(p0.Text)+1, 1)

				ui.Render(p0)
				state = "search"

			}
		}
	}
}

var defaultStyle ui.Style = ui.NewStyle(ui.ColorClear)
var activeStyle ui.Style = ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold|ui.ModifierReverse)
var headerStyle ui.Style = ui.NewStyle(ui.ColorClear, ui.ColorClear, ui.ModifierBold|ui.ModifierUnderline)
var markedStyle ui.Style = ui.NewStyle(ui.ColorGreen, ui.ColorClear, ui.ModifierBold|ui.ModifierReverse)

func (tv *TextPager) updateTable(tbl *widgets.Table) {

	var showCols []int
	if support.BoolSum(tv.colSticky) > 0 {
		showCols = make([]int, len(tv.colNames)+1)
	} else {
		showCols = make([]int, len(tv.colNames))
	}
	showColCount := 0

	size := 1
	j := -support.BoolSum(tv.colSticky)

	for i, v := range tv.colWidth {
		if tv.colSticky[i] {
			size += v + 1
			showCols[showColCount] = i
			showColCount++
			j++
		}
		if size > tv.visibleCols {
			break
		}
	}

	if support.BoolSum(tv.colSticky) > 0 {
		size++
		showCols[showColCount] = -1
		showColCount++
	}

	for i, v := range tv.colWidth {
		if !tv.colSticky[i] {
			if j >= tv.leftCol {
				size += v + 1
				showCols[showColCount] = i
				showColCount++
			}
			if size > tv.visibleCols {
				break
			}
			j++
		}
	}
	// if size >= tv.visibleCols {
	// 	showColCount--
	// }

	// showCols2 := make([]int, showColCount)
	// copy(showCols2, showCols)
	// showCols = showCols2

	tbl.Rows = make([][]string, tv.visibleRows-2)
	tbl.RowStyles = make(map[int]ui.Style)

	headerVals := make([]string, showColCount)
	widths := make([]int, showColCount)

	size = 1
	j = -support.BoolSum(tv.colSticky)
	k := 0
	for i, v := range tv.colNames {
		if tv.colSticky[i] {
			for len(v) < tv.colWidth[i] {
				v += " "
			}
			headerVals[k] = v + "*"
			widths[k] = support.MaxInt(0, support.MinInt(tv.colWidth[i]+1, tv.visibleCols-size))
			size += tv.colWidth[i] + 1

			r := []rune(v)
			if len(r) > tv.colWidth[i] {
				headerVals[k] = string(r[:tv.colWidth[i]]) + "$"
			}

			if j == tv.leftCol && tv.colSelectMode {
				headerVals[k] = headerVals[k][0:len(headerVals[k])-3] + "<=*"
				// headerVals[k] = headerVals[k] + "<="
			}

			k++
			j++
		}
	}
	if support.BoolSum(tv.colSticky) > 0 {
		headerVals[k] = ""
		widths[k] = 0
		size++
		k++
	}
	for i, v := range tv.colNames {
		if !tv.colSticky[i] && k < showColCount {
			if j >= tv.leftCol {
				for len(v) < tv.colWidth[i] {
					v += " "
				}
				headerVals[k] = v + " "
				widths[k] = support.MaxInt(0, support.MinInt(tv.colWidth[i]+1, tv.visibleCols-size))
				size += tv.colWidth[i] + 1
				r := []rune(v)
				if len(r) > tv.colWidth[i] {
					headerVals[k] = string(r[:tv.colWidth[i]]) + "$"
				}

				// headerVals[k] = v //[:support.MinInt(tv.colWidth[i], len(v))]

				if j == tv.leftCol && tv.colSelectMode {
					headerVals[k] = headerVals[k][0:len(headerVals[k])-3] + "<= "
				}
				k++
			}
			j++
		}
	}

	tbl.ColumnWidths = widths
	tbl.Rows[0] = headerVals
	tbl.RowStyles[0] = headerStyle
	if tv.colSelectMode {
		tbl.RowStyles[0] = activeStyle
	}

	e := tv.topRow
	setActive := false
	lastIdx := 1

	for i := 1; e != nil && i < len(tbl.Rows); i++ {
		lastIdx = i
		vals := make([]string, showColCount)
		line, _ := e.Value.(*TextRecord)

		for j, v := range showCols[:showColCount] {
			if v >= 0 && len(line.Values) > v {
				vals[j] = line.Values[v]

				r := []rune(line.Values[v])
				if len(r) > widths[j] {
					vals[j] = string(r[:widths[j]]) + "$"
				}
			} else {
				vals[j] = ""
			}

		}

		t, _ := e.Value.(*TextRecord)

		// tbl.Rows[i] = vals[:support.MinInt(tv.colWidth[i], len(vals))]

		tbl.Rows[i] = vals
		if !tv.colSelectMode && i == tv.activeRow {
			tbl.RowStyles[i] = activeStyle
			setActive = true
		} else if t.Flag {
			tbl.RowStyles[i] = markedStyle
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
	if !tv.colSelectMode && !setActive {
		tbl.RowStyles[lastIdx] = activeStyle
		tv.activeRow = lastIdx
	}
	// if tbl.Rows[2] != nil {
	// tbl.Rows[1][0] = fmt.Sprintf("%v", showCols[:showColCount])
	// tbl.Rows[2][0] = fmt.Sprintf("%d, %d", size, tv.visibleCols)
	// }

}

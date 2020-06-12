package textfile

import "fmt"

// TextColumn - the column to export. Initially, the idx is set to -1 for named columns.
type TextColumn struct {
	name      string // the name is only used to then find the index
	idx       int    // this value is -1 when starting for a named column.
	isNum     bool   // sort as a number
	isReverse bool   // sort in reverse
}

//Name - getter for TextColumn.name
func (col *TextColumn) Name() string {
	return col.name
}

//String - write TextColumn as a string
func (col *TextColumn) String() string {
	if col.idx == -1 {
		if col.isNum {
			return fmt.Sprintf("%s,n", col.name)
		}
		return col.name
	}

	if col.isNum {
		return fmt.Sprintf("idx:%d,n", col.idx)
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

// AsNumber - sets this column to be processed as a numeric value (for sorting purposes)
func (col *TextColumn) AsNumber() *TextColumn {
	col.isNum = true
	return col
}

// AsReverse - This column should be sorted in reverse
func (col *TextColumn) AsReverse() *TextColumn {
	col.isReverse = true
	return col
}

// NewIndexColumn - the column to export. For columns defined by index, idx is the column number (0-based).
func NewIndexColumn(idx int) *TextColumn {
	return &TextColumn{
		name: "",
		idx:  idx,
	}
}

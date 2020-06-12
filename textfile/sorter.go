package textfile

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

var defaultSortBufferLen int = 10000

// TextSorter is used to export specific columns from a tab delimited file
type TextSorter struct {
	txt           *DelimitedTextFile
	cols          []*TextColumn
	showComments  bool
	sortBufferLen int
}

// NewTextSorter - create a new text sorter
func NewTextSorter(f *DelimitedTextFile, cols []*TextColumn) *TextSorter {
	return &TextSorter{
		txt:           f,
		cols:          cols,
		showComments:  false,
		sortBufferLen: defaultSortBufferLen,
	}
}

// WithShowComments - set showing comments
func (tes *TextSorter) WithShowComments(b bool) *TextSorter {
	tes.showComments = b
	return tes
}

// WriteFile - sort and write the new columns to the given stream
func (tes *TextSorter) WriteFile(out io.Writer) error {

	// for _, v := range tes.cols {
	// 	fmt.Printf("Sorting by: %s\n", v)
	// }

	files := make([]*os.File, 0)
	// fmt.Printf("tmpFiles: %v, len:%d\n", files, len(files))
	defer cleanUpTemp(&files)

	var line *TextRecord
	var err error = nil
	wroteHeader := false

	records := make(TextSortRecords, tes.sortBufferLen)
	pos := 0

	for err == nil {
		line, err = tes.txt.ReadLine()
		if err != nil {
			break
		}

		if line.Values == nil {
			// comment
			if tes.showComments {
				fmt.Fprint(out, line.RawString)
			}
			continue
		}

		if !wroteHeader {
			err := tes.populateColIndex()
			if err != nil {
				return err
			}
			if !tes.txt.noHeader {
				tes.writeHeader(out)
			}
			wroteHeader = true
		}

		records[pos] = TextSortRecord{val: line, cols: tes.cols}
		pos++

		if pos >= tes.sortBufferLen {
			sort.Sort(records)

			curTemp, fErr := ioutil.TempFile("", "tabl_sort")
			if fErr != nil {
				return fErr
			}

			for _, rec := range records {
				tes.writeLine(curTemp, rec.val)
			}
			pos = 0

			files = append(files, curTemp)
		}

	}

	if pos > 0 {
		sort.Sort(records)

		curTemp, fErr := ioutil.TempFile("", "tabl_sort")
		if fErr != nil {
			return fErr
		}

		for _, rec := range records[:pos] {
			tes.writeLine(curTemp, rec.val)
		}
		curTemp.Close()
		pos = 0

		files = append(files, curTemp)
	}

	tes.txt.Close()

	// merge the temp files

	sortReaders := make([]*DelimitedTextFile, len(files))
	sortBuffer := make(TextSortRecords, len(files))

	validReaders := 0
	for i, f := range files {
		sortReaders[i] = tes.txt.Clone(f.Name()).WithNoHeader(true)
		rec, rErr := sortReaders[i].ReadLine()
		if rErr != nil {
			return rErr
		}

		sortBuffer[i] = TextSortRecord{val: rec, cols: tes.cols, idx: i}
		validReaders++
	}

	for validReaders > 0 {
		// fmt.Printf("Sort Buffer: %v\n", sortBuffer)
		sort.Sort(sortBuffer)
		lowest := sortBuffer[0]
		// fmt.Printf("Lowest: %v\n", lowest)
		tes.writeLine(out, lowest.val)
		rec, tErr := sortReaders[lowest.idx].ReadLine()

		if tErr != nil {
			lowest.val = nil
			sortReaders[lowest.idx].Close()
			sortReaders[lowest.idx] = nil
			validReaders--
		} else {
			sortBuffer[0] = TextSortRecord{val: rec, cols: tes.cols, idx: lowest.idx}
		}
	}
	// fmt.Printf("tmpFiles: %v, len:%d\n", files, len(files))

	return nil

}

func (tes *TextSorter) populateColIndex() error {
	for _, col := range tes.cols {
		if col.idx == -1 {
			for i, header := range tes.txt.Header {
				if header == col.name {
					col.idx = i
					break
				}
			}
			if col.idx == -1 {
				return fmt.Errorf("Missing column: %s\n\nHave:%v\nHeaderComment: %v", col.name, tes.txt.Header, tes.txt.headerComment)
			}
		}
	}
	return nil
}

func (tes *TextSorter) writeHeader(out io.Writer) {
	if tes.txt.rawHeaderLine != "" {
		fmt.Fprint(out, tes.txt.rawHeaderLine)
	}
}

func (tes *TextSorter) writeLine(out io.Writer, line *TextRecord) {
	fmt.Fprint(out, line.RawString)
}

// TextSortRecord - wrapper to hold a text record (line) and the sort column definitions
type TextSortRecord struct {
	val  *TextRecord
	cols []*TextColumn
	idx  int
}

// TextSortRecords - sorting interface?
type TextSortRecords []TextSortRecord

func (a TextSortRecords) Len() int      { return len(a) }
func (a TextSortRecords) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a TextSortRecords) Less(i, j int) bool {

	if a[i].val == nil {
		return false
	}
	if a[j].val == nil {
		return true
	}

	for k := 0; k < len(a[0].cols); k++ {
		col := a[0].cols[k]
		if col.isNum {
			one, err1 := strconv.ParseFloat(a[i].val.Values[col.idx], 64)
			if err1 != nil {
				return true
			}
			two, err2 := strconv.ParseFloat(a[j].val.Values[col.idx], 64)
			if err2 != nil {
				return true
			}

			if col.isReverse {
				if two < one {
					return true
				} else if one < two {
					return false
				}
			} else {
				if one < two {
					return true
				} else if two < one {
					return false
				}
			}
		} else {
			one := a[i].val.Values[col.idx]
			two := a[j].val.Values[col.idx]

			if col.isReverse {
				if two < one {
					return true
				} else if one < two {
					return false
				}
			} else {
				if one < two {
					return true
				} else if two < one {
					return false
				}
			}
		}

	}

	return false

}

func cleanUpTemp(files *[]*os.File) {
	for _, f := range *files {
		// fmt.Printf("Removing temp file: %s\n", f.Name())
		os.Remove(f.Name())
	}
}

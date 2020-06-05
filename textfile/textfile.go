package textfile

import (
	"compress/gzip"
	"container/list"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/mbreese/tabgo/bufread"
)

var defaultBufferSize int = 64 * 1024

// DelimitedTextFile is the main delimited text file handler
type DelimitedTextFile struct {
	Filename       string
	Delim          rune
	Quote          rune
	Comment        rune
	rd             io.ReadCloser
	buf            []byte
	pos            int
	bufLen         int
	next           rune
	hasNext        bool
	isEOF          bool
	curLineNum     int
	curDataLineNum int
}

// TextRecord is a single line/record from a delimited text file
type TextRecord struct {
	Values      []string
	LineNum     int
	DataLineNum int
	RawString   string
	Flag        bool
	ByteSize    int
}

// NewDelimitedFile returns an open delimited text file
func NewDelimitedFile(fname string, delim rune, quote rune, comment rune) *DelimitedTextFile {
	return NewDelimitedFileSize(fname, delim, quote, comment, defaultBufferSize)
}

// NewDelimitedFileSize returns an open delimited text file
func NewDelimitedFileSize(fname string, delim rune, quote rune, comment rune, bufferSize int) *DelimitedTextFile {
	return &DelimitedTextFile{
		Filename:       fname,
		Delim:          delim,
		Quote:          quote,
		Comment:        comment,
		rd:             nil,
		buf:            make([]byte, bufferSize),
		next:           0,
		hasNext:        false,
		pos:            0,
		bufLen:         0,
		isEOF:          false,
		curLineNum:     0,
		curDataLineNum: 0,
	}
}

// NewTabFile returns an open tab-delimited text file
func NewTabFile(fname string) *DelimitedTextFile {
	return NewTabFileSize(fname, defaultBufferSize)
}

// NewTabFileSize returns an open tab-delimited text file
func NewTabFileSize(fname string, bufferSize int) *DelimitedTextFile {
	return NewDelimitedFileSize(fname, '\t', 0, '#', bufferSize)
}

// NewCSVFile returns an open comma-delimited text file
func NewCSVFile(fname string) *DelimitedTextFile {
	return NewCSVFileSize(fname, defaultBufferSize)
}

// NewCSVFileSize returns an open comma-delimited text file
func NewCSVFileSize(fname string, bufferSize int) *DelimitedTextFile {
	return NewDelimitedFileSize(fname, ',', '"', '#', bufferSize)
}

func (txt *DelimitedTextFile) nextRune() (rune, error) {
	if !txt.hasNext {
		err := txt.populateNext()
		if err != nil {
			return 0, err
		}
	}

	ret := txt.next
	txt.populateNext()
	return ret, nil
}

func (txt *DelimitedTextFile) peekRune() (rune, error) {
	if !txt.hasNext {
		err := txt.populateNext()
		if err != nil {
			return 0, err
		}
	}

	return txt.next, nil
}

func (txt *DelimitedTextFile) populateNext() error {
	// fmt.Println("Getting next rune")
	txt.hasNext = false

	var b rune
	var width int

	// try to pull a rune (if we are at the end of the buffer, it will return RuneError)
	if txt.bufLen == 0 {
		b = utf8.RuneError
		width = 0
	} else {
		b, width = utf8.DecodeRune(txt.buf[txt.pos:txt.bufLen])
	}

	for b == utf8.RuneError {
		// fmt.Printf(" -- error, so let's refill the buffer (pos: %d, len:%d)\n", txt.pos, txt.bufLen)
		remCount := txt.bufLen - txt.pos

		if remCount >= utf8.UTFMax {
			return errors.New("remaining buffer is too big(?!?!)")
		}
		// let's pull what's left and refill the buffer
		// fmt.Printf(" -- remaining buffer size: %d\n", remCount)

		if remCount > 0 {
			copy(txt.buf, txt.buf[txt.pos:txt.bufLen])
		}

		n, err := txt.rd.Read(txt.buf[remCount:])
		// fmt.Printf(" -- read: %d bytes\n", n)

		if err != nil {
			if n > 0 {
				txt.isEOF = true
			} else {
				// fmt.Printf(" -- !! got an error: %s\n", err)
				txt.pos = 0
				txt.bufLen = 0
				return err
			}
		}

		txt.bufLen = n + remCount
		txt.pos = 0

		b, width = utf8.DecodeRune(txt.buf)
	}

	txt.next = b
	txt.hasNext = true
	txt.pos += width

	// fmt.Printf(" -- next: %s => %c\n", txt.next, txt.next)

	return nil
}

// ReadLine read a line from the file
func (txt *DelimitedTextFile) ReadLine() (*TextRecord, error) {
	// if txt.isEOF {
	// 	return nil, io.EOF
	// }
	if txt.rd == nil {
		txt.open()
	}

	for true {

		var sb strings.Builder
		var sbRaw strings.Builder

		inQuote := false
		first := true
		isComment := false

		var err error = nil
		var b rune = 0
		byteSize := 0

		l := list.New()
		// fmt.Fprintln(os.Stderr, "\n==========\n")
		for err == nil {

			b, err = txt.nextRune()
			if err != nil {
				// fmt.Fprintf(os.Stderr, "err: %s, b:%s\n", err, string(b))
				break
			}
			// fmt.Fprintf(os.Stderr, "%s\n", b)
			sbRaw.WriteRune(b)
			byteSize += utf8.RuneLen(b)

			if first {
				first = false
				if b == txt.Comment {
					isComment = true
				}
			}

			if isComment {
				if b == '\r' {
					// do nothing...
				} else if b == '\n' {
					break
				} else {
					sb.WriteRune(b)
				}
			} else if inQuote {
				if b == txt.Quote {
					// got a new quote -- if this is a double quote (""), then replace it with ("),
					// otherwise, we should exit quote mode for the cell
					n, err2 := txt.peekRune()
					if err2 == nil && n == txt.Quote {
						txt.nextRune()
						sb.WriteRune(b)
						sbRaw.WriteRune(n)
						byteSize += utf8.RuneLen(n)
					} else {
						inQuote = false
					}
				} else {
					sb.WriteRune(b)
				}
			} else if b == txt.Quote {
				inQuote = true
			} else if b == '\r' {
				// do nothing...
			} else if b == '\n' {
				break
			} else if b == txt.Delim {
				// fmt.Printf("val: %s\n", sb.String())
				l.PushBack(sb.String())
				sb.Reset()
			} else {
				sb.WriteRune(b)
			}
		}
		if sb.Len() > 0 {
			// fmt.Printf("val: %s\n", sb.String())
			l.PushBack(sb.String())
		}

		if l.Len() > 0 {
			// TODO: Add an option to return blank lines?
			txt.curLineNum++

			if err == io.EOF {
				if l.Len() > 0 {
					txt.isEOF = true
					err = nil
				} else {
					return nil, err
				}
			}

			if isComment {
				return &TextRecord{
					Values:      nil,
					LineNum:     txt.curLineNum,
					DataLineNum: -1,
					RawString:   sbRaw.String(),
					Flag:        false,
					ByteSize:    byteSize,
				}, err
			}
			cols := make([]string, l.Len())
			e := l.Front()
			for i := 0; i < len(cols); i++ {
				s, _ := e.Value.(string)
				cols[i] = s
				e = e.Next()
			}

			txt.curDataLineNum++
			return &TextRecord{
				Values:      cols,
				LineNum:     txt.curLineNum,
				DataLineNum: txt.curDataLineNum,
				RawString:   sbRaw.String(),
				Flag:        false,
				ByteSize:    byteSize,
			}, err
		}
	}
	// This never happens
	return nil, nil
}

// Close the file
func (txt *DelimitedTextFile) Close() {
	txt.buf = nil
	txt.rd.Close()
}

// func (txt *DelimitedTextFile) fillBuffer(offset int) error {
// 	// fmt.Println("fill()")

// 	n, err := txt.rd.Read(txt.buf[offset:])

// 	if err != nil {
// 		txt.pos = 0
// 		txt.bufLen = 0
// 		return err
// 	}

// 	txt.pos = 0
// 	txt.bufLen = n
// 	return nil

// 	// fmt.Printf("out, txt.bufLen=%d, txt.pos=%d, txt.isEOF=%b\n", txt.bufLen, txt.pos, txt.isEOF)
// }

// open the file, taking into account that the file might be gzip compressed.
func (txt *DelimitedTextFile) open() {
	rd := bufread.OpenFile(txt.Filename)

	magic := make([]byte, 2)
	c, err := rd.Peek(magic)
	if err != nil {
		rd.Close()
		panic(err)
	}

	if c == 2 {
		if magic[0] == 0x1F && magic[1] == 0x8B {
			// this is gzipped
			// fmt.Println("Gzip!")
			tmp, e := gzip.NewReader(rd)
			if e != nil {
				rd.Close()
				panic(e)
			}
			txt.rd = tmp
			return
		}
	}

	// this must be a very short file... or not gzipped
	// fmt.Println("Plain!")
	txt.rd = rd
}

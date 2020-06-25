package textfile

import (
	"compress/gzip"
	"container/list"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/mbreese/tabl/bufread"
)

var defaultBufferSize int = 64 * 1024

// DelimitedTextFile is the main delimited text file handler
type DelimitedTextFile struct {
	Filename string
	Delim    rune
	Quote    rune
	Comment  rune
	IsCrLf   bool

	// This is the underlying reader
	rd             io.ReadCloser
	buf            []byte
	pos            int
	bufLen         int
	next           rune
	hasNext        bool
	isEOF          bool
	curLineNum     int
	curDataLineNum int
	Header         []string
	noHeader       bool
	headerComment  bool
	lastComment    string
	rawHeaderLine  string
}

// TextRecord is a single line/record from a delimited text file
type TextRecord struct {
	Values      []string
	LineNum     int
	DataLineNum int
	RawString   string
	Flag        bool
	ByteSize    int
	parent      *DelimitedTextFile
}

// NewDelimitedFile returns an open delimited text file
func NewDelimitedFile(fname string, delim rune, quote rune, comment rune, isCrLf bool) *DelimitedTextFile {
	return &DelimitedTextFile{
		Filename: fname,
		Delim:    delim,
		Quote:    quote,
		Comment:  comment,
		buf:      make([]byte, defaultBufferSize),
		IsCrLf:   isCrLf,
	}
}

// NewTabFile returns an open tab-delimited text file
func NewTabFile(fname string) *DelimitedTextFile {
	return NewDelimitedFile(fname, '\t', 0, '#', false)
}

// NewCSVFile returns an open comma-delimited text file
func NewCSVFile(fname string) *DelimitedTextFile {
	return NewDelimitedFile(fname, ',', '"', '#', true)
}

// Clone returns a new DelimitedTextReader just like txt, but with a new filename
func (txt *DelimitedTextFile) Clone(fname string) *DelimitedTextFile {
	return &DelimitedTextFile{
		Filename: fname,
		Delim:    txt.Delim,
		Quote:    txt.Quote,
		Comment:  txt.Comment,
		rd:       nil,
		buf:      make([]byte, defaultBufferSize),
		Header:   nil,
	}
}

// WithBufferSize - set the internal read buffer (default 64K)
func (txt *DelimitedTextFile) WithBufferSize(bufferSize int) *DelimitedTextFile {
	txt.buf = make([]byte, bufferSize)
	return txt
}

// WithNoHeader - this file doesn't have a header... so fake it with col1, col2, etc...
func (txt *DelimitedTextFile) WithNoHeader(val bool) *DelimitedTextFile {
	txt.noHeader = val
	return txt
}

// WithHeaderComment - The header is the last non-blank comment line
func (txt *DelimitedTextFile) WithHeaderComment(val bool) *DelimitedTextFile {
	txt.headerComment = val
	return txt
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
		err := txt.open()
		if err != nil {
			return nil, err
		}
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

		txt.curLineNum++

		if err == io.EOF {
			if l.Len() > 0 {
				txt.isEOF = true
				err = nil
			} else {
				return nil, err
			}
		}

		if l.Len() > 0 {
			if isComment {
				if txt.Header == nil {
					txt.lastComment = sbRaw.String()
				}

				return &TextRecord{
					Values:      nil,
					LineNum:     txt.curLineNum,
					DataLineNum: -1,
					RawString:   sbRaw.String(),
					Flag:        false,
					ByteSize:    byteSize,
					parent:      txt,
				}, err
			}
			cols := make([]string, l.Len())
			e := l.Front()
			for i := 0; i < len(cols); i++ {
				s, _ := e.Value.(string)
				cols[i] = s
				e = e.Next()
			}

			// This is the first non-comment, non-blank row. Must be the header.
			//
			// Note, we don't send the header as a line because we don't always know
			// what the header *is* when we get a line. If the header is commented, then
			// we will have already sent the commented line. We shouldn't then re-send it as
			// a header. So, we will always pull what the "header" is internally.
			//
			if txt.Header == nil {
				// fmt.Println("Need to populate header...")
				if txt.noHeader {
					// fmt.Println("nvm, no header for this file...")
					txt.Header = make([]string, len(cols))
					for i := 0; i < len(txt.Header); i++ {
						txt.Header[i] = fmt.Sprintf("col%d", (i + 1))
					}
				} else if txt.headerComment {
					// fmt.Printf("Last comment: %s\n", txt.lastComment)
					if txt.lastComment != "" {
						s2 := txt.lastComment
						b2, l := utf8.DecodeRuneInString(s2)
						for b2 == txt.Comment || b2 == ' ' {
							s2 = s2[l:]
							b2, l = utf8.DecodeRuneInString(s2)
						}
						txt.Header = txt.splitLine(s2)
					}
				} else {
					// fmt.Printf("cols used for header: %v\n", cols)
					txt.Header = cols
					txt.rawHeaderLine = sbRaw.String()
					// go around for another pass...
					continue
				}
			}

			// If we need to add a new header column...
			// the default here is to use a blank value for the header
			if len(txt.Header) < len(cols) {
				newHeader := make([]string, len(cols))
				if txt.noHeader {
					for i := 0; i < len(newHeader); i++ {
						newHeader[i] = fmt.Sprintf("col%d", (i + 1))
					}
				}
				copy(newHeader, txt.Header)
				txt.Header = newHeader
			}

			txt.curDataLineNum++

			return &TextRecord{
				Values:      cols,
				LineNum:     txt.curLineNum,
				DataLineNum: txt.curDataLineNum,
				RawString:   sbRaw.String(),
				Flag:        false,
				ByteSize:    byteSize,
				parent:      txt,
			}, err

		}
		// fmt.Fprintf(os.Stderr, "Empty line? %d\n", l.Len())
	}
	// This never happens
	return nil, nil
}

// Close the file
func (txt *DelimitedTextFile) Close() {
	txt.buf = nil
	if txt.rd != nil {
		txt.rd.Close()
	}
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
func (txt *DelimitedTextFile) open() error {
	rd := bufread.OpenFile(txt.Filename)

	magic := make([]byte, 2)
	c, err := rd.Peek(magic)
	if err != nil {
		rd.Close()
		return err
	}

	if c == 2 {
		if magic[0] == 0x1F && magic[1] == 0x8B {
			// this is gzipped
			// fmt.Println("Gzip!")
			tmp, e := gzip.NewReader(rd)
			if e != nil {
				rd.Close()
				fmt.Println("Here??")
				panic(e)
			}
			txt.rd = tmp
			return nil
		}
	}

	// this must be a very short file... or not gzipped
	// fmt.Println("Plain!")
	txt.rd = rd
	return nil
}

// Takes a string and splits it like you'd split the input stream.
func (txt *DelimitedTextFile) splitLine(buf string) []string {

	var sb strings.Builder

	inQuote := false

	var length int = 0
	var b rune = 0

	l := list.New()
	for len(buf) > 0 {

		b, length = utf8.DecodeRuneInString(buf)

		if b == utf8.RuneError {
			break
		}

		buf = buf[length:]

		if inQuote {
			if b == txt.Quote {
				// got a new quote -- if this is a double quote (""), then replace it with ("),
				// otherwise, we should exit quote mode for the cell
				b2, l2 := utf8.DecodeRuneInString(buf)
				if b2 == txt.Quote {
					buf = buf[l2:]
					sb.WriteRune(b)
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

	cols := make([]string, l.Len())
	e := l.Front()
	for i := 0; i < len(cols); i++ {
		s, _ := e.Value.(string)
		cols[i] = s
		e = e.Next()
	}

	return cols
}

// GetValue - Fetch a value from a record by column name
func (rec *TextRecord) GetValue(k string) (string, error) {
	for i, v := range rec.parent.Header {
		if v == k {
			return rec.Values[i], nil
		}
	}
	return "", fmt.Errorf("Missing column: %s", k)
}

package bufread

import (
	"errors"
	"io"
	"os"
)

const defaultBufferSize int = 16 * 1024

// BufferedReader is a buffered file reader that supports Peek
// The reader works by keeping two separate []byte's. These are both populated, but
// only the "left" buffer is read from. When that buffer has been exhausted, the right
// buffer takes over (and a new right buffer is created.)
type BufferedReader struct {
	rd          io.ReadCloser
	left        []byte
	right       []byte
	leftLength  int
	rightLength int
	curPos      int
	bufferSize  int
	isEOF       bool
}

// OpenFile opens a new buffered file (may be stdin or from a file)
func OpenFile(fname string) *BufferedReader {
	return OpenFileSize(fname, defaultBufferSize)
}

// OpenFileSize opens a new buffered file (may be stdin or from a file)
func OpenFileSize(fname string, bufferSize int) *BufferedReader {
	var r io.ReadCloser

	if fname == "-" {
		r = os.Stdin
	} else {
		var err error
		r, err = os.Open(fname)
		check(err)
	}

	return &BufferedReader{
		rd:          r,
		left:        nil,
		right:       nil,
		leftLength:  0,
		rightLength: 0,
		curPos:      0,
		bufferSize:  bufferSize,
		isEOF:       false,
	}

}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// ReadByte return a single byte from the buffer
func (br *BufferedReader) ReadByte() (byte, error) {
	var b byte

	if br.left == nil || br.curPos >= br.leftLength {
		err := br.swapAndFill()
		if err == io.EOF {
			return 0, err
		}
	}

	b = br.left[br.curPos]
	br.curPos++

	return b, nil
}

// Read bytes from the parent stream
func (br *BufferedReader) Read(p []byte) (int, error) {
	if br.left == nil || br.curPos >= br.leftLength {
		// fmt.Println("here1")
		err := br.swapAndFill()
		// fmt.Println("/here1")
		if err == io.EOF {
			return 0, err
		}
	}

	n := 0

	for i := 0; i < len(p); i++ {
		// fmt.Printf("i=%d, n=%d, curPos=%d, leftLength=%d\n", i, n, br.curPos, br.leftLength)
		if br.curPos >= br.leftLength {
			// fmt.Println("here2")
			err := br.swapAndFill()
			// fmt.Printf("/here2, err=%s\n", err)
			if err == io.EOF {
				// fmt.Printf("EOF!! n=%d\n", n)
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}
		}
		p[i] = br.left[br.curPos]
		br.curPos++
		n++
	}
	// fmt.Printf("n=%d\n", n)

	return n, nil
}

// swap the left and right buffer and fill the right if needed.
func (br *BufferedReader) swapAndFill() error {
	// fmt.Printf("swapAndFill(), br.curPos=%d, br.leftLength=%d\n", br.curPos, br.leftLength)

	for br.leftLength <= 0 || br.curPos >= br.leftLength {
		// fmt.Println("swapAndFill()")
		if br.isEOF {
			// fmt.Println("We are already EOF")
			// if the first fill resulted in an EOF, we are done here
			return io.EOF
		}
		if br.right != nil {
			// fmt.Println("Fill right")
			// the right buffer is present, so move it to the left and refill the right
			br.left = br.right
			br.leftLength = br.rightLength
			br.curPos = 0

			br.right = make([]byte, br.bufferSize)
			n, err := br.rd.Read(br.right)
			if err != nil {
				if err == io.EOF {
					br.isEOF = true
				} else {
					panic(err)
				}
			}
			br.rightLength = n

		} else {
			// fmt.Println("Fill left")
			// right buffer isn't present, so either (a) this is the first run, or (b) the entire file was read into the left buffer to start
			// fill the left
			br.left = make([]byte, br.bufferSize)

			n, err := br.rd.Read(br.left)

			if err != nil {
				if err == io.EOF {
					br.isEOF = true
				} else {
					panic(err)
				}
			}

			br.leftLength = n
			br.curPos = 0

			if !br.isEOF {
				// fill the right
				br.right = make([]byte, br.bufferSize)
				n, err2 := br.rd.Read(br.right)
				if err2 != nil {
					if err2 == io.EOF {
						br.isEOF = true
					} else {
						panic(err2)
					}
				}
				br.rightLength = n
			} else {
				br.rightLength = 0
			}
		}
	}
	return nil

}

// Peek n bytes into the future. Peek will not fill any buffer, but if necessary can read from the right buffer directly
func (br *BufferedReader) Peek(p []byte) (int, error) {
	if len(p) > br.bufferSize {
		return 0, errors.New("Bytes to Peek is larger than buffer length")
	}

	if br.left == nil || br.curPos >= br.leftLength {
		err := br.swapAndFill()
		if err == io.EOF {
			return 0, err
		}
	}

	j := 0
	n := 0

	for i := 0; i < len(p); i++ {
		if br.curPos+i < br.leftLength {
			// the peek should be from the left buffer
			p[i] = br.left[br.curPos+i]
			n++
		} else {
			// the peek should be from the right buffer
			if j < br.rightLength {
				p[i] = br.right[j]
				n++
				j++
			} else {
				// nope... we are done here
				return n, io.EOF
			}
		}
	}

	return n, nil
}

// Close the parent reader
func (br *BufferedReader) Close() error {
	e := br.rd.Close()
	return e
}

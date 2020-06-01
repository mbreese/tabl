package bufread_test

import (
	"io"
	"testing"

	"github.com/mbreese/tabgo/bufread"
)

func TestOpen(t *testing.T) {
	br := bufread.OpenFile("testdata/test.txt")
	br.Close()
}

func TestMissingFile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	br := bufread.OpenFile("testdata/missing.txt")
	if br != nil {
		t.Error("Expected a nil...")
	}

}

func TestPeekTooLong(t *testing.T) {
	br := bufread.OpenFileSize("testdata/test.txt", 4)

	buf := make([]byte, 5)
	_, err := br.Peek(buf)
	if err == nil {
		t.Error("Expected an error peeking too far into the future")
	}

	br.Close()
}
func TestPeek(t *testing.T) {
	br := bufread.OpenFile("testdata/test.txt")

	buf := make([]byte, 4)
	n, err := br.Peek(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Peek far enough into the future...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong peek value, expected \"# com\", but got: %s", buf)
	}

	n, err = br.Peek(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Peek far enough into the future...")
	}

	// Peek should return the same bytes each time
	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong peek value, expected \"# com\", but got: %s", buf)
	}

	br.Close()
}

func TestRead(t *testing.T) {
	br := bufread.OpenFile("testdata/test.txt")

	buf := make([]byte, 4)
	n, err := br.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Read far enough...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong read value, expected \"# com\", but got: %s", buf)
	}

	br.Close()
}

func TestReadFull(t *testing.T) {
	br := bufread.OpenFileSize("testdata/test.txt", 100000)

	buf := make([]byte, 4)
	n, err := br.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Read far enough...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong read value, expected \"# com\", but got: %s", buf)
	}

	br.Close()
}

func TestReadFull2(t *testing.T) {
	br := bufread.OpenFileSize("testdata/test.txt", 100000)

	buf := make([]byte, 10000)
	n, err := br.Read(buf)
	if err != nil {
		t.Errorf("Didn't expect EOF, but got %d bytes?", n)
	}

	if n != 166 {
		t.Errorf("Read %d bytes, but expected 166\n%s", n, buf)
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong read value, expected \"# com\", but got: %s", buf)
	}

	n, err = br.Read(buf)
	if err == nil || err != io.EOF {
		t.Errorf("Expected EOF, and read nothing? Read %d bytes? err=%s", n, err)
	}

	// fmt.Println("Here?")
	br.Close()
}

func TestReadSwap(t *testing.T) {
	br := bufread.OpenFileSize("testdata/test.txt", 4)

	buf := make([]byte, 10)
	n, err := br.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 10 {
		t.Error("We couldn't Read far enough...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' || buf[4] != 'm' || buf[5] != 'm' || buf[6] != 'e' || buf[7] != 'n' || buf[8] != 't' || buf[9] != '\n' {
		t.Errorf("Wrong read value, expected \"# comment\\n\", but got: %s", buf)
	}

	br.Close()
}

func TestPeekSwap(t *testing.T) {
	br := bufread.OpenFileSize("testdata/test.txt", 4)

	buf := make([]byte, 2)
	n, err := br.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 2 {
		t.Error("We couldn't Read far enough...")
	}

	if buf[0] != '#' || buf[1] != ' ' {
		t.Errorf("Wrong read value, expected \"# \", but got: %s", buf)
	}
	t.Logf("Read: %s", buf)

	buf2 := make([]byte, 4)
	n, err = br.Read(buf2)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Peek far enough...")
	}

	if buf2[0] != 'c' || buf2[1] != 'o' || buf2[2] != 'm' || buf2[3] != 'm' {
		t.Errorf("Wrong Peek value, expected \"omm\", but got: %s", buf)
	}
	t.Logf("Peek: %s", buf2)

	br.Close()
}

func TestPeekReadPeek(t *testing.T) {
	br := bufread.OpenFile("testdata/test.txt")

	buf := make([]byte, 4)
	n, err := br.Peek(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Peek far enough into the future...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong peek value, expected \"# com\", but got: %s", buf)
	}
	t.Logf("Peek: %s", buf)

	n, err = br.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Read far enough into the future...")
	}

	if buf[0] != '#' || buf[1] != ' ' || buf[2] != 'c' || buf[3] != 'o' {
		t.Errorf("Wrong read value, expected \"# com\", but got: %s", buf)
	}

	t.Logf("Read: %s", buf)

	n, err = br.Peek(buf)
	if err != nil {
		t.Error(err)
	}

	if n != 4 {
		t.Error("We couldn't Peek far enough into the future...")
	}

	if buf[0] != 'm' || buf[1] != 'm' || buf[2] != 'e' || buf[3] != 'n' {
		t.Errorf("Wrong peek value, expected \"mmen\", but got: %s", buf)
	}

	t.Logf("Peek: %s", buf)

	br.Close()
}

package textfile_test

import (
	"testing"

	"github.com/mbreese/tabgo/textfile"
)

func TestCompare(t *testing.T) {

	one := textfile.NewTabFile("../bufread/testdata/test.txt")
	two := textfile.NewTabFile("../bufread/testdata/test.txt.gz")

	if one == nil || two == nil {
		t.Error("File is nil")
	}

	for true {
		l1, e1 := one.ReadLine()
		l2, e2 := two.ReadLine()
		if e1 == nil || e2 == nil {
			if !(e1 == nil && e2 == nil) {
				t.Error("Errors out of sync")
			} else {
				break
			}
		}

		if l1 != nil && l2 != nil {
			if len(l1.Values) != len(l2.Values) {
				t.Error("Records had different lengths")
			}
		}
	}

}

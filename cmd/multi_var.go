package cmd

import (
	"fmt"

	"github.com/mbreese/tabl/textfile"
)

// MultiColumnVar allow multiple values for an arg
type MultiColumnVar struct {
	Values []*textfile.TextColumn
}

// String *pflag.Value interface
func (mv *MultiColumnVar) String() string {
	return fmt.Sprintf("%v", mv.Values)
}

//Set *pflag.Value interface
func (mv *MultiColumnVar) Set(s string) error {
	// fmt.Printf("Setting var to: %s\n", s)

	var newcols []*textfile.TextColumn
	var err error

	if len(s) > 2 && s[len(s)-2:] == ":n" {
		newcols, err = ParseColumnList(s[0 : len(s)-2])
		if err != nil {
			return err
		}
		for i, v := range newcols {
			newcols[i] = v.AsNumber()
		}
	} else if len(s) > 2 && s[len(s)-2:] == ":r" {
		newcols, err = ParseColumnList(s[0 : len(s)-2])
		if err != nil {
			return err
		}
		for i, v := range newcols {
			newcols[i] = v.AsReverse()
		}
	} else if len(s) > 3 && (s[len(s)-3:] == ":rn" || s[len(s)-3:] == ":nr") {
		newcols, err = ParseColumnList(s[0 : len(s)-3])
		if err != nil {
			return err
		}
		for i, v := range newcols {
			newcols[i] = v.AsReverse().AsNumber()
		}
	} else {
		newcols, err = ParseColumnList(s)
		if err != nil {
			return err
		}
	}

	mv.Values = append(mv.Values, newcols...)

	return nil
}

// Type *pflag.Value interface
func (mv *MultiColumnVar) Type() string {
	return "cols"
}

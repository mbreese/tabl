package textfile_test

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/compgen-io/tabl/textfile"
	"github.com/parquet-go/parquet-go"
)

type testAddr struct {
	City string `parquet:"city"`
	Zip  string `parquet:"zip"`
}

// testRow covers the type mappings that are easy to get wrong -- decimals need
// their scale applied, dates and timestamps need converting out of their
// integer representations, and unsigned ints must not be printed as signed.
type testRow struct {
	ID    int64             `parquet:"id"`
	Name  *string           `parquet:"name,optional"`
	Score float64           `parquet:"score"`
	Flag  bool              `parquet:"flag"`
	Count uint32            `parquet:"count"`
	Price int64             `parquet:"price,decimal(2:10)"`
	Day   int32             `parquet:"day,date"`
	When  int64             `parquet:"when,timestamp(microsecond)"`
	Tags  []string          `parquet:"tags,list"`
	Addr  testAddr          `parquet:"addr"`
	Attrs map[string]string `parquet:"attrs"`
}

func strptr(s string) *string { return &s }

func writeTestParquet(t *testing.T) string {
	t.Helper()

	name := strptr("abc")
	rows := []testRow{
		{
			ID:    1,
			Name:  name,
			Score: 3.5,
			Flag:  true,
			Count: 4000000000, // > MaxInt32, so a signed render would be negative
			Price: 12345,      // decimal(scale=2) -> 123.45
			Day:   19723,      // 2024-01-01
			When:  1700000000000000,
			Tags:  []string{"a", "b"},
			Addr:  testAddr{City: "Boston", Zip: "02115"},
			Attrs: map[string]string{"k": "v"},
		},
		{
			ID:    2,
			Name:  nil, // NULL
			Score: -0.25,
			Flag:  false,
			Count: 0,
			Price: -50, // -0.50
			Day:   0,   // 1970-01-01
			When:  0,
			Tags:  []string{},
			Addr:  testAddr{City: "", Zip: ""},
			Attrs: map[string]string{},
		},
	}

	path := filepath.Join(t.TempDir(), "test.parquet")
	if err := parquet.WriteFile(path, rows); err != nil {
		t.Fatalf("could not write test parquet: %v", err)
	}
	return path
}

func readAll(t *testing.T, pq *textfile.ParquetFile) [][]string {
	t.Helper()

	var out [][]string
	for {
		rec, err := pq.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadLine: %v", err)
		}
		out = append(out, rec.Values)
	}
	pq.Close()
	return out
}

func TestParquetHeader(t *testing.T) {
	path := writeTestParquet(t)

	pq := textfile.NewParquetFile(path)
	defer pq.Close()

	if err := pq.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// structs flatten to dotted names; lists and maps stay a single column
	want := []string{
		"id", "name", "score", "flag", "count", "price", "day", "when",
		"tags", "addr.city", "addr.zip", "attrs",
	}

	got := pq.GetHeader()
	if len(got) != len(want) {
		t.Fatalf("header = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParquetValues(t *testing.T) {
	path := writeTestParquet(t)

	rows := readAll(t, textfile.NewParquetFile(path))
	if len(rows) != 2 {
		t.Fatalf("read %d rows, want 2", len(rows))
	}

	tests := []struct {
		row  int
		col  int
		name string
		want string
	}{
		{0, 0, "id", "1"},
		{0, 1, "name", "abc"},
		{0, 2, "score", "3.5"},
		{0, 3, "flag", "true"},
		{0, 4, "count", "4000000000"}, // unsigned, must not wrap negative
		{0, 5, "price", "123.45"},     // decimal scale applied
		{0, 6, "day", "2024-01-01"},   // days since epoch decoded
		{0, 7, "when", "2023-11-14T22:13:20Z"},
		{0, 8, "tags", `["a","b"]`},
		{0, 9, "addr.city", "Boston"},
		{0, 10, "addr.zip", "02115"},
		{0, 11, "attrs", `{"k":"v"}`},

		{1, 0, "id", "2"},
		{1, 1, "name", ""}, // NULL -> empty by default
		{1, 2, "score", "-0.25"},
		{1, 3, "flag", "false"},
		{1, 5, "price", "-0.50"},
		{1, 6, "day", "1970-01-01"},
		{1, 7, "when", "1970-01-01T00:00:00Z"},
	}

	for _, tt := range tests {
		got := rows[tt.row][tt.col]
		if got != tt.want {
			t.Errorf("row %d %s = %q, want %q", tt.row, tt.name, got, tt.want)
		}
	}
}

func TestParquetNAString(t *testing.T) {
	path := writeTestParquet(t)

	rows := readAll(t, textfile.NewParquetFile(path).WithNAString("NA"))
	if rows[1][1] != "NA" {
		t.Errorf("NULL with --na=NA = %q, want %q", rows[1][1], "NA")
	}
	// a non-null value must be untouched
	if rows[0][1] != "abc" {
		t.Errorf("non-null value = %q, want %q", rows[0][1], "abc")
	}
}

func TestParquetStdinRejected(t *testing.T) {
	pq := textfile.NewParquetFile("-")
	if err := pq.Open(); err != textfile.ErrParquetStdin {
		t.Errorf("Open(\"-\") = %v, want ErrParquetStdin", err)
	}
}

func TestParquetMissingFile(t *testing.T) {
	pq := textfile.NewParquetFile(filepath.Join(t.TempDir(), "nope.parquet"))
	if err := pq.Open(); err == nil {
		t.Error("Open() on a missing file returned no error")
	}
}

// A parquet file is a valid RecordReader, so it can drive the viewer/pager.
func TestParquetIsRecordReader(t *testing.T) {
	path := writeTestParquet(t)

	var rd textfile.RecordReader = textfile.NewParquetFile(path)
	defer rd.Close()

	rec, err := rd.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine: %v", err)
	}
	if rd.NoHeader() {
		t.Error("NoHeader() = true, want false (the schema is the header)")
	}
	if rd.HeaderLine() == "" {
		t.Error("HeaderLine() is empty")
	}
	if v, err := rec.GetValue("addr.city"); err != nil || v != "Boston" {
		t.Errorf("GetValue(addr.city) = %q, %v; want Boston", v, err)
	}
}

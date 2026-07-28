package textfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/format"
)

// how many rows we pull from the file at a time
const parquetBatchSize int = 256

// julianEpoch is the Julian day number for 1970-01-01, used to decode the
// legacy INT96 timestamps written by older Hive/Impala versions.
const julianEpoch int64 = 2440588

// ErrParquetStdin is returned when we're asked to read parquet from a pipe.
// The file footer (which holds the schema) lives at the *end* of the file, so
// reading one requires random access -- there's nothing we can do with a stream.
var ErrParquetStdin = errors.New("parquet input cannot be read from a pipe (the file footer requires random access); please supply a file path")

// colShape describes how a display column is assembled from the underlying
// parquet leaf columns.
type colShape int

const (
	shapeScalar colShape = iota // one leaf, one value
	shapeList                   // repeated values, rendered as a JSON array
	shapeMap                    // key/value leaves, rendered as a JSON object
)

// parquetLeaf is a single physical column in the parquet file, along with the
// knowledge of how to turn its values into text.
type parquetLeaf struct {
	name    string // name relative to its display column (for structs in lists)
	format  func(parquet.Value) string
	numeric bool // formats to a bare number, so it can go into JSON unquoted
	boolean bool
}

// jsonValue renders a value for embedding in a JSON cell.
func (l *parquetLeaf) jsonValue(v parquet.Value) interface{} {
	if v.IsNull() {
		return nil
	}
	s := l.format(v)
	if l.numeric {
		return json.Number(s)
	}
	if l.boolean {
		return v.Boolean()
	}
	return s
}

// parquetColumn is a column as the user sees it. A scalar maps 1:1 onto a leaf;
// lists and maps gather several values (and possibly several leaves) into one
// JSON-encoded cell.
type parquetColumn struct {
	name   string
	shape  colShape
	first  int // index of its first leaf column
	leaves []*parquetLeaf
}

// ParquetFile reads a parquet file and presents it as tabular text records.
// It implements RecordReader, so the viewer, pager, and exporters can consume
// it exactly like a delimited text file.
type ParquetFile struct {
	Filename string
	naString string

	f  *os.File
	pf *parquet.File

	rgs   []parquet.RowGroup
	rgIdx int
	rows  parquet.Rows

	buf    []parquet.Row
	bufPos int
	bufLen int

	header []string
	cols   []*parquetColumn
	byLeaf [][]parquet.Value // scratch: values of the current row, per leaf

	curLineNum     int
	curDataLineNum int
	isEOF          bool
	openErr        error
	opened         bool
}

// NewParquetFile returns a reader for a parquet file
func NewParquetFile(fname string) *ParquetFile {
	return &ParquetFile{
		Filename: fname,
	}
}

// WithNAString - the value to write for NULL cells (default is empty)
func (pq *ParquetFile) WithNAString(s string) *ParquetFile {
	pq.naString = s
	return pq
}

// Open eagerly opens the file and reads its schema. This is optional --
// ReadLine will open on demand -- but calling it up front lets the caller
// report a bad path or a corrupt file before any output is written.
func (pq *ParquetFile) Open() error {
	return pq.open()
}

func (pq *ParquetFile) open() error {
	if pq.opened {
		return pq.openErr
	}
	pq.opened = true

	if pq.Filename == "-" {
		pq.openErr = ErrParquetStdin
		return pq.openErr
	}

	f, err := os.Open(pq.Filename)
	if err != nil {
		pq.openErr = err
		return err
	}

	st, err := f.Stat()
	if err != nil {
		f.Close()
		pq.openErr = err
		return err
	}

	pf, err := parquet.OpenFile(f, st.Size())
	if err != nil {
		f.Close()
		pq.openErr = fmt.Errorf("%s: %v", pq.Filename, err)
		return pq.openErr
	}

	pq.f = f
	pq.pf = pf
	pq.rgs = pf.RowGroups()
	pq.buf = make([]parquet.Row, parquetBatchSize)

	leafIdx := 0
	pq.buildColumns(pf.Schema(), "", &leafIdx)
	pq.byLeaf = make([][]parquet.Value, leafIdx)

	pq.header = make([]string, len(pq.cols))
	for i, c := range pq.cols {
		pq.header[i] = c.name
	}

	return nil
}

// buildColumns walks the schema depth-first, turning it into a flat list of
// display columns. Plain groups (structs) are flattened into dotted names;
// lists and maps become a single JSON column rather than being descended into.
func (pq *ParquetFile) buildColumns(node parquet.Node, prefix string, leafIdx *int) {
	for _, f := range node.Fields() {
		name := f.Name()
		if prefix != "" {
			name = prefix + "." + name
		}

		lt := logicalTypeOf(f)
		isList := lt != nil && lt.List != nil
		isMap := lt != nil && lt.Map != nil

		switch {
		case f.Leaf() && !f.Repeated():
			col := &parquetColumn{name: name, shape: shapeScalar, first: *leafIdx}
			col.leaves = []*parquetLeaf{newLeaf("", f)}
			*leafIdx++
			pq.cols = append(pq.cols, col)

		case f.Leaf():
			// a repeated primitive with no LIST annotation
			col := &parquetColumn{name: name, shape: shapeList, first: *leafIdx}
			col.leaves = []*parquetLeaf{newLeaf("", f)}
			*leafIdx++
			pq.cols = append(pq.cols, col)

		case isMap:
			col := &parquetColumn{name: name, shape: shapeMap, first: *leafIdx}
			collectLeaves(f, "", &col.leaves, leafIdx)
			pq.cols = append(pq.cols, col)

		case isList || f.Repeated():
			col := &parquetColumn{name: name, shape: shapeList, first: *leafIdx}
			collectLeaves(f, "", &col.leaves, leafIdx)
			pq.cols = append(pq.cols, col)

		default:
			// a plain struct -- flatten it into dotted column names
			pq.buildColumns(f, name, leafIdx)
		}
	}
}

// collectLeaves gathers every leaf under a list/map group in schema order,
// naming them relative to that group.
func collectLeaves(node parquet.Node, prefix string, out *[]*parquetLeaf, leafIdx *int) {
	for _, f := range node.Fields() {
		name := joinRelative(prefix, f.Name())
		if f.Leaf() {
			*out = append(*out, newLeaf(name, f))
			*leafIdx++
		} else {
			collectLeaves(f, name, out, leafIdx)
		}
	}
}

// joinRelative builds a relative field name, dropping the structural wrapper
// names that the LIST and MAP annotations require ("list", "element",
// "key_value"). Those carry no information for a text rendering.
func joinRelative(prefix, name string) string {
	switch name {
	case "list", "element", "key_value", "array", "item":
		return prefix
	}
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// logicalTypeOf safely fetches a node's logical type. Node.Type() is documented
// as panicking on some non-leaf nodes, so guard against that.
func logicalTypeOf(n parquet.Node) (lt *format.LogicalType) {
	defer func() {
		recover()
	}()
	if t := n.Type(); t != nil {
		lt = t.LogicalType()
	}
	return
}

// ReadLine reads the next row from the file
func (pq *ParquetFile) ReadLine() (*TextRecord, error) {
	if err := pq.open(); err != nil {
		return nil, err
	}

	row, err := pq.nextRow()
	if err != nil {
		return nil, err
	}

	values := pq.formatRow(row)

	pq.curLineNum++
	pq.curDataLineNum++

	// RawString is what gets written back out when saving from the pager, so
	// it needs the same escaping the tab exporter applies.
	var sb strings.Builder
	for i, v := range values {
		if i > 0 {
			sb.WriteString("\t")
		}
		sb.WriteString(quoteTab(v))
	}
	sb.WriteString("\n")
	raw := sb.String()

	return &TextRecord{
		Values:      values,
		LineNum:     pq.curLineNum,
		DataLineNum: pq.curDataLineNum,
		RawString:   raw,
		Flag:        false,
		ByteSize:    len(raw),
		parent:      pq,
	}, nil
}

// nextRow pulls one row, refilling the batch buffer and advancing across row
// groups as needed.
func (pq *ParquetFile) nextRow() (parquet.Row, error) {
	for {
		if pq.bufPos < pq.bufLen {
			row := pq.buf[pq.bufPos]
			pq.bufPos++
			return row, nil
		}

		if pq.rows == nil {
			if pq.rgIdx >= len(pq.rgs) {
				pq.isEOF = true
				return nil, io.EOF
			}
			pq.rows = pq.rgs[pq.rgIdx].Rows()
			pq.rgIdx++
		}

		n, err := pq.rows.ReadRows(pq.buf)
		pq.bufPos = 0
		pq.bufLen = n

		// Either an error or an empty read means this row group is done. We
		// still serve whatever came back in the same call before moving on.
		if err != nil || n == 0 {
			pq.rows.Close()
			pq.rows = nil
			if err != nil && err != io.EOF {
				pq.isEOF = true
				return nil, err
			}
		}
	}
}

// formatRow buckets the row's values by leaf column, then renders each display
// column.
func (pq *ParquetFile) formatRow(row parquet.Row) []string {
	for i := range pq.byLeaf {
		pq.byLeaf[i] = pq.byLeaf[i][:0]
	}
	for _, v := range row {
		c := v.Column()
		if c >= 0 && c < len(pq.byLeaf) {
			pq.byLeaf[c] = append(pq.byLeaf[c], v)
		}
	}

	out := make([]string, len(pq.cols))
	for i, col := range pq.cols {
		out[i] = pq.renderColumn(col)
	}
	return out
}

func (pq *ParquetFile) renderColumn(col *parquetColumn) string {
	switch col.shape {
	case shapeScalar:
		vals := pq.byLeaf[col.first]
		if len(vals) == 0 || vals[0].IsNull() {
			return pq.naString
		}
		return col.leaves[0].format(vals[0])

	case shapeMap:
		if len(col.leaves) < 2 {
			return pq.naString
		}
		keys := pq.byLeaf[col.first]
		vals := pq.byLeaf[col.first+1]
		if len(keys) == 0 || (len(keys) == 1 && keys[0].IsNull()) {
			return pq.naString
		}
		// A JSON object needs string keys, so the key is always formatted as
		// text even when it's numeric.
		obj := make(map[string]interface{}, len(keys))
		for i, k := range keys {
			if k.IsNull() {
				continue
			}
			key := col.leaves[0].format(k)
			if i < len(vals) {
				obj[key] = col.leaves[1].jsonValue(vals[i])
			} else {
				obj[key] = nil
			}
		}
		return marshalCell(obj, pq.naString)

	default: // shapeList
		if len(col.leaves) == 1 {
			vals := pq.byLeaf[col.first]
			if len(vals) == 0 || (len(vals) == 1 && vals[0].IsNull()) {
				return pq.naString
			}
			arr := make([]interface{}, len(vals))
			for i, v := range vals {
				arr[i] = col.leaves[0].jsonValue(v)
			}
			return marshalCell(arr, pq.naString)
		}

		// a list of structs -- zip the leaves together by position
		n := len(pq.byLeaf[col.first])
		if n == 0 {
			return pq.naString
		}
		arr := make([]interface{}, 0, n)
		for i := 0; i < n; i++ {
			obj := make(map[string]interface{}, len(col.leaves))
			for j, leaf := range col.leaves {
				vals := pq.byLeaf[col.first+j]
				if i < len(vals) {
					obj[leaf.name] = leaf.jsonValue(vals[i])
				} else {
					obj[leaf.name] = nil
				}
			}
			arr = append(arr, obj)
		}
		return marshalCell(arr, pq.naString)
	}
}

func marshalCell(v interface{}, na string) string {
	b, err := json.Marshal(v)
	if err != nil {
		return na
	}
	return string(b)
}

// Close the file
func (pq *ParquetFile) Close() {
	if pq.rows != nil {
		pq.rows.Close()
		pq.rows = nil
	}
	if pq.f != nil {
		pq.f.Close()
		pq.f = nil
	}
}

// GetHeader returns the flattened column names from the parquet schema
func (pq *ParquetFile) GetHeader() []string {
	pq.open()
	return pq.header
}

// IsEOF - have we read every row?
func (pq *ParquetFile) IsEOF() bool {
	return pq.isEOF
}

// NoHeader is always false -- the parquet schema is the header
func (pq *ParquetFile) NoHeader() bool {
	return false
}

// HeaderLine returns the column names as a tab-delimited line
func (pq *ParquetFile) HeaderLine() string {
	pq.open()
	var sb strings.Builder
	for i, v := range pq.header {
		if i > 0 {
			sb.WriteString("\t")
		}
		sb.WriteString(quoteTab(v))
	}
	sb.WriteString("\n")
	return sb.String()
}

// newLeaf builds the text formatter for a single physical column.
//
// Note that parquet.Value.String() is deliberately not used here: it renders
// the *physical* type only, so a DATE would come out as a day count and a
// DECIMAL(10,2) as its raw unscaled integer.
func newLeaf(name string, n parquet.Node) *parquetLeaf {
	leaf := &parquetLeaf{name: name}
	t := n.Type()
	lt := t.LogicalType()

	if lt != nil {
		switch {
		case lt.UTF8 != nil, lt.Enum != nil, lt.Json != nil:
			leaf.format = func(v parquet.Value) string { return string(v.ByteArray()) }
			return leaf

		case lt.Bson != nil:
			leaf.format = func(v parquet.Value) string { return fmt.Sprintf("%x", v.ByteArray()) }
			return leaf

		case lt.Decimal != nil:
			scale := lt.Decimal.Scale
			leaf.numeric = true
			leaf.format = func(v parquet.Value) string { return formatDecimal(v, scale) }
			return leaf

		case lt.Date != nil:
			leaf.format = func(v parquet.Value) string {
				return time.Unix(int64(v.Int32())*86400, 0).UTC().Format("2006-01-02")
			}
			return leaf

		case lt.Time != nil:
			unit := lt.Time.Unit
			leaf.format = func(v parquet.Value) string { return formatParquetTime(v, unit) }
			return leaf

		case lt.Timestamp != nil:
			ts := lt.Timestamp
			leaf.format = func(v parquet.Value) string { return formatTimestamp(v, ts) }
			return leaf

		case lt.UUID != nil:
			leaf.format = func(v parquet.Value) string { return formatUUID(v.ByteArray()) }
			return leaf

		case lt.Integer != nil:
			it := lt.Integer
			leaf.numeric = true
			leaf.format = func(v parquet.Value) string { return formatInteger(v, it) }
			return leaf
		}
	}

	// no logical type -- fall back to the physical representation
	switch t.Kind() {
	case parquet.Boolean:
		leaf.boolean = true
		leaf.format = func(v parquet.Value) string { return strconv.FormatBool(v.Boolean()) }
	case parquet.Int32:
		leaf.numeric = true
		leaf.format = func(v parquet.Value) string { return strconv.FormatInt(int64(v.Int32()), 10) }
	case parquet.Int64:
		leaf.numeric = true
		leaf.format = func(v parquet.Value) string { return strconv.FormatInt(v.Int64(), 10) }
	case parquet.Int96:
		leaf.format = formatInt96
	case parquet.Float:
		leaf.numeric = true
		leaf.format = func(v parquet.Value) string {
			return strconv.FormatFloat(float64(v.Float()), 'g', -1, 32)
		}
	case parquet.Double:
		leaf.numeric = true
		leaf.format = func(v parquet.Value) string {
			return strconv.FormatFloat(v.Double(), 'g', -1, 64)
		}
	case parquet.ByteArray:
		leaf.format = func(v parquet.Value) string { return string(v.ByteArray()) }
	default: // FixedLenByteArray and anything else -- show the bytes
		leaf.format = func(v parquet.Value) string { return fmt.Sprintf("%x", v.ByteArray()) }
	}

	return leaf
}

// decimalUnscaled pulls the unscaled integer out of a DECIMAL value. It can be
// backed by an int32, an int64, or a big-endian two's complement byte array.
func decimalUnscaled(v parquet.Value) *big.Int {
	switch v.Kind() {
	case parquet.Int32:
		return big.NewInt(int64(v.Int32()))
	case parquet.Int64:
		return big.NewInt(v.Int64())
	default:
		b := v.ByteArray()
		i := new(big.Int).SetBytes(b)
		if len(b) > 0 && b[0]&0x80 != 0 {
			// negative: subtract 2^(8n) to undo the two's complement
			i.Sub(i, new(big.Int).Lsh(big.NewInt(1), uint(len(b)*8)))
		}
		return i
	}
}

func formatDecimal(v parquet.Value, scale int32) string {
	i := decimalUnscaled(v)

	if scale == 0 {
		return i.String()
	}
	if scale < 0 {
		// a negative scale means the value is scaled *up*
		i.Mul(i, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scale)), nil))
		return i.String()
	}

	neg := i.Sign() < 0
	digits := new(big.Int).Abs(i).String()
	if len(digits) <= int(scale) {
		digits = strings.Repeat("0", int(scale)-len(digits)+1) + digits
	}

	cut := len(digits) - int(scale)
	out := digits[:cut] + "." + digits[cut:]
	if neg {
		out = "-" + out
	}
	return out
}

func formatParquetTime(v parquet.Value, unit format.TimeUnit) string {
	var d time.Duration
	switch {
	case unit.Millis != nil:
		d = time.Duration(v.Int32()) * time.Millisecond
	case unit.Micros != nil:
		d = time.Duration(v.Int64()) * time.Microsecond
	default:
		d = time.Duration(v.Int64())
	}
	return time.Unix(0, 0).UTC().Add(d).Format("15:04:05.999999999")
}

func formatTimestamp(v parquet.Value, ts *format.TimestampType) string {
	n := v.Int64()

	var t time.Time
	switch {
	case ts.Unit.Millis != nil:
		t = time.Unix(n/1e3, (n%1e3)*1e6)
	case ts.Unit.Micros != nil:
		t = time.Unix(n/1e6, (n%1e6)*1e3)
	default:
		t = time.Unix(n/1e9, n%1e9)
	}
	t = t.UTC()

	if ts.IsAdjustedToUTC {
		return t.Format(time.RFC3339Nano)
	}
	// a local (unzoned) timestamp -- don't imply a timezone we don't know
	return t.Format("2006-01-02T15:04:05.999999999")
}

func formatInteger(v parquet.Value, it *format.IntType) string {
	if it.IsSigned {
		if v.Kind() == parquet.Int32 {
			return strconv.FormatInt(int64(v.Int32()), 10)
		}
		return strconv.FormatInt(v.Int64(), 10)
	}

	switch it.BitWidth {
	case 8:
		return strconv.FormatUint(uint64(uint8(v.Int32())), 10)
	case 16:
		return strconv.FormatUint(uint64(uint16(v.Int32())), 10)
	case 32:
		return strconv.FormatUint(uint64(uint32(v.Int32())), 10)
	default:
		return strconv.FormatUint(uint64(v.Int64()), 10)
	}
}

func formatUUID(b []byte) string {
	if len(b) != 16 {
		return fmt.Sprintf("%x", b)
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// formatInt96 decodes the legacy INT96 timestamp: nanoseconds within the day in
// the low 64 bits, Julian day number in the high 32.
func formatInt96(v parquet.Value) string {
	i := v.Int96()
	nanos := int64(uint64(i[0]) | uint64(i[1])<<32)
	days := int64(i[2]) - julianEpoch
	return time.Unix(days*86400, nanos).UTC().Format(time.RFC3339Nano)
}

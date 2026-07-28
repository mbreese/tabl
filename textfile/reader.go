package textfile

// RecordReader is a source of tabular records. It is implemented by
// DelimitedTextFile (tab/CSV) and ParquetFile, and is what the viewer, pager,
// and CSV exporter consume -- they only ever need to pull records and ask
// about the header, so any format that can produce a *TextRecord will work.
type RecordReader interface {
	// ReadLine returns the next record, or io.EOF when there are none left.
	ReadLine() (*TextRecord, error)

	// Close releases the underlying file/stream.
	Close()

	// GetHeader returns the column names.
	GetHeader() []string

	// IsEOF is true once the end of the input has been reached.
	IsEOF() bool

	// HeaderLine returns the header formatted as a raw line (including the
	// trailing newline), suitable for writing back out to a file.
	HeaderLine() string

	// NoHeader is true when the first row should be treated as data and the
	// column names are synthesized (col1, col2, ...).
	NoHeader() bool
}

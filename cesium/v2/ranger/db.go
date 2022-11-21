package ranger

import (
	"github.com/cockroachdb/errors"
	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

var (
	// ErrRangeOverlap is returned when a range overlaps with an existing range in the DB.
	ErrRangeOverlap = errors.Wrap(validate.Error, "range overlaps with an existing range")
	// RangeNotFound is returned when a requested range is not found in the DB.
	RangeNotFound = errors.Wrap(query.NotFound, "range not found")
)

// DB provides a persistent, concurrent store for reading and writing ranges of telemetry.
type DB struct {
	idx *index
	fs  *fs
}

func Open(fs xfs.FS) (*DB, error) {
	idx, err := openIndex(fs)
	if err != nil {
		return nil, err
	}
	return &DB{idx: idx, fs: newFS(fs)}, nil
}

// NewIterator opens a new invalidated Iterator using the given options.
// A seeking call is required before the iterator can be used.
func (db *DB) NewIterator(opts *IteratorConfig) *Iterator {
	i := &Iterator{idx: db.idx, readerFactory: db.newReader}
	i.SetBounds(opts.Bounds)
	return i
}

// NewWriter opens a new Writer using the given configuration.
func (db *DB) NewWriter(cfg *WriterConfig) (*Writer, error) {
	wrapped, err := db.fs.newOffsetWriteCloser()
	if err != nil {
		return nil, err
	}
	return &Writer{wrapped: wrapped, idx: db.idx, WriterConfig: cfg}, nil
}

func (db *DB) newReader(ptr *pointer) (*Reader, error) {
	wrapped, err := db.fs.newReader(ptr.fileKey)
	if err != nil {
		return nil, err
	}
	return &Reader{
		ptr:      ptr,
		ReaderAt: xio.PartialReader(wrapped, int64(ptr.offset), int64(ptr.length)),
		Closer:   wrapped,
	}, nil
}

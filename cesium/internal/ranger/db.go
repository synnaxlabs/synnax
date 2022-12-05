package ranger

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	xio "github.com/synnaxlabs/x/io"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

var (
	// ErrRangeOverlap is returned when a range overlaps with an existing range in the DB.
	ErrRangeOverlap = errors.Wrap(validate.Error, "range overlaps with an existing range")
	// RangeNotFound is returned when a requested range is not found in the DB.
	RangeNotFound = errors.Wrap(query.NotFound, "range not found")
)

// DB provides a persistent, concurrent store for reading and writing ranges of telemetry.
type DB struct {
	idx       *index
	dataFiles *fileController
}

type Config struct {
	FS             xfs.FS
	FileSize       telem.Size
	MaxDescriptors int
	Logger         *zap.Logger
}

func (c Config) Validate() error {
	v := validate.New("ranger")
	validate.Positive(v, "fileSize", c.FileSize)
	validate.Positive(v, "maxDescriptors", c.MaxDescriptors)
	validate.NotNil(v, "fs", c.FS)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.MaxDescriptors = override.Numeric(c.MaxDescriptors, other.MaxDescriptors)
	c.FileSize = override.Numeric(c.FileSize, other.FileSize)
	c.FS = override.Nil(c.FS, other.FS)
	return c
}

var DefaultConfig = Config{
	FileSize:       1 * telem.Gigabyte,
	MaxDescriptors: 100,
}

var _ config.Config[Config] = Config{}

func Open(configs ...Config) (*DB, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	idx := &index{
		observer: observe.New[indexUpdate](),
	}
	idxPst, err := openIndexPersist(idx, cfg)
	if err != nil {
		return nil, err
	}
	idx.mu.pointers, err = idxPst.load()
	if err != nil {
		return nil, err
	}
	controller, err := openFileController(cfg)
	if err != nil {
		return nil, err
	}

	return &DB{idx: idx, dataFiles: controller}, nil
}

// NewIterator opens a new invalidated Iterator using the given options.
// A seeking call is required before the iterator can be used.
func (db *DB) NewIterator(opts IteratorConfig) *Iterator {
	i := &Iterator{idx: db.idx, readerFactory: db.newReader}
	i.SetBounds(opts.Bounds)
	return i
}

// NewWriter opens a new Writer using the given configuration.
func (db *DB) NewWriter(cfg WriterConfig) (*Writer, error) {
	key, internal, err := db.dataFiles.acquireWriter()
	if err != nil {
		return nil, err
	}
	if db.idx.overlap(cfg.Range()) {
		return nil, ErrRangeOverlap
	}
	return &Writer{fileKey: key, internal: internal, idx: db.idx, cfg: cfg}, nil
}

func (db *DB) Close() error {
	w := errutil.NewCatch(errutil.WithAggregation())
	w.Exec(db.idx.close)
	w.Exec(db.dataFiles.close)
	return w.Error()
}

func (db *DB) newReader(ptr pointer) (*Reader, error) {
	internal, err := db.dataFiles.acquireReader(ptr.fileKey)
	if err != nil {
		return nil, err
	}
	reader := xio.PartialReaderAt(internal, int64(ptr.offset), int64(ptr.length))
	return &Reader{
		ptr:      ptr,
		ReaderAt: reader,
		Closer:   internal,
	}, nil
}

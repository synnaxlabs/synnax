// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

// DB provides a persistent, concurrent store for reading and writing ranges of telemetry
// to and from an underlying file system.
//
// A DB provides two types for accessing data:
//
//   - Writer allows the caller to write a blob of telemetry occupying a particular time
//     range.
//
//   - Iterator allows the caller ot iterate over the telemetry ranges in a DB in time order,
//     and provides an io.Reader like interface for accessing the data.
//
// A DB is safe for concurrent use, and multiple writers and iterators can access the DB
// at once.
//
// It's important to note that a DB is heavily optimized for large (several megabytes
// to gigabytes), append only writes. While small, out of order writes are valid, the
// user will see a heavy performance hit.
//
//
// A DB must be closed after use to avoid leaking any underlying resources/locks.
type DB struct {
	idx       *index
	dataFiles *fileController
}

// Config is the configuration for opening a DB.
type Config struct {
	// FS is the filesystem that the DB will use to store its data. DB will write to the
	// root of the filesystem, so this should probably be a subdirectory. DB should have
	// exclusive access, and it should be empty when the DB is first opened.
	// [REQUIRED]
	FS xfs.FS
	// FileSize is the maximum size of a data file in bytes. When a data file reaches this
	// size, a new file will be created.
	// [OPTIONAL] Default: 1GB
	FileSize telem.Size
	// MaxDescriptors is the maximum number of file descriptors that the DB will use. A
	// higher value will allow more concurrent reads and writes. It's important to note
	// that the exact performance impact of changing this value is still relatively unknown.
	// [OPTIONAL] Default: 100
	MaxDescriptors int
	// Logger is the witness of it all.
	// [OPTIONAL] Default: zap.NewNop()
	Logger *zap.Logger
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		FileSize:       1 * telem.Gigabyte,
		MaxDescriptors: 100,
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ranger")
	validate.Positive(v, "fileSize", c.FileSize)
	validate.Positive(v, "maxDescriptors", c.MaxDescriptors)
	validate.NotNil(v, "fs", c.FS)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.MaxDescriptors = override.Numeric(c.MaxDescriptors, other.MaxDescriptors)
	c.FileSize = override.Numeric(c.FileSize, other.FileSize)
	c.FS = override.Nil(c.FS, other.FS)
	return c
}

// Open opens a DB using a merged view of the provided configurations (where the next
// configuration overrides the previous).
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

// NewIterator opens a new invalidated Iterator using the given configuration.
// A seeking call is required before it can be used.
func (db *DB) NewIterator(cfg IteratorConfig) *Iterator {
	i := &Iterator{idx: db.idx, readerFactory: db.newReader}
	i.SetBounds(cfg.Bounds)
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

// Close closes the DB. Close should not be called concurrently with any other DB methods.
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
	return &Reader{ptr: ptr, ReaderAt: reader, Closer: internal}, nil
}

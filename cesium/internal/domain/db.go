// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"io"
)

var (
	// ErrDomainOverlap is returned when a domain overlaps with an existing domain in the DB.
	ErrDomainOverlap = errors.Wrap(validate.Error, "domain overlaps with an existing domain")
	// RangeNotFound is returned when a requested domain is not found in the DB.
	RangeNotFound = errors.Wrap(query.NotFound, "domain not found")
)

// DB provides a persistent, concurrent store for reading and writing domains of telemetry
// to and from an underlying file system.
//
// A DB provides two types for accessing data:
//
//   - Writer allows the caller to write a blob of telemetry occupying a particular time
//     domain.
//
//   - Iterator allows the caller ot iterate over the telemetry domains in a DB in time order,
//     and provides an io.Reader like interface for accessing the data.
//
// A DB is safe for concurrent use, and multiple writers and iterators can access the DB
// at once.
//
// It's important to note that a DB is heavily optimized for large (several megabytes
// to gigabytes), append only writes. While small, out of order writes are valid, the
// user will see a heavy performance hit.
//
// A DB must be closed after use to avoid leaking any underlying resources/locks.
type DB struct {
	Config
	idx   *index
	files *fileController
}

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
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
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		FileSize:       1 * telem.Gigabyte,
		MaxDescriptors: 100,
	}
)

// Validate implements config.GateConfig.
func (c Config) Validate() error {
	v := validate.New("domain")
	validate.Positive(v, "fileSize", c.FileSize)
	validate.Positive(v, "maxDescriptors", c.MaxDescriptors)
	validate.NotNil(v, "fs", c.FS)
	return v.Error()
}

// Override implements config.GateConfig.
func (c Config) Override(other Config) Config {
	c.MaxDescriptors = override.Numeric(c.MaxDescriptors, other.MaxDescriptors)
	c.FileSize = override.Numeric(c.FileSize, other.FileSize)
	c.FS = override.Nil(c.FS, other.FS)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Open opens a DB using a merged view of the provided configurations (where the next
// configuration overrides the previous).
func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	idx := &index{Observer: observe.New[indexUpdate]()}
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

	return &DB{Config: cfg, idx: idx, files: controller}, nil
}

// NewIterator opens a new invalidated Iterator using the given configuration.
// A seeking call is required before it can be used.
func (db *DB) NewIterator(cfg IteratorConfig) *Iterator {
	i := &Iterator{
		Instrumentation: db.Instrumentation.Child("iterator"),
		idx:             db.idx,
		readerFactory:   db.newReader,
	}
	i.SetBounds(cfg.Bounds)
	return i
}

func (db *DB) Delete(ctx context.Context, tr telem.TimeRange, startOffset int, endOffset int) bool {
	db.idx.mu.RLock()
	deleteStart, ok := db.idx.unprotectedSearch(tr.Start.SpanRange(0))
	if !ok {
		db.idx.mu.RUnlock()
		return false
	}
	deleteEnd, ok := db.idx.unprotectedSearch(tr.End.SpanRange(0))
	if !ok {
		db.idx.mu.RUnlock()
		return false
	}
	db.idx.mu.RUnlock()

	db.idx.mu.Lock()
	defer db.idx.mu.Lock()

	start, _ := db.idx.get(deleteStart)
	end, _ := db.idx.get(deleteEnd)

	// add to tombstones
	for _, p := range db.idx.mu.pointers[deleteStart+1 : deleteEnd] {
		db.idx.insertTombstone(ctx, p)
	}
	db.idx.insertTombstone(ctx, pointer{
		TimeRange: telem.TimeRange{
			Start: tr.Start,
			End:   start.End,
		},
		fileKey: start.fileKey,
		offset:  start.offset,
		length:  uint32(startOffset), // length of {tr.Start, start.End}
	})

	db.idx.insertTombstone(ctx, pointer{
		TimeRange: telem.TimeRange{
			Start: end.Start,
			End:   tr.End,
		},
		fileKey: end.fileKey,
		offset:  uint32(endOffset),
		length:  end.length - uint32(endOffset), // length of {end.Start, tr.End}
	})

	// remove whole pointers
	db.idx.mu.pointers = append(db.idx.mu.pointers[:deleteStart], db.idx.mu.pointers[deleteEnd+1:]...)

	// make a new pointer for the first half of start pointer
	db.idx.mu.pointers = append(append(db.idx.mu.pointers[:deleteStart], pointer{
		TimeRange: telem.TimeRange{
			Start: start.Start,
			End:   tr.Start,
		},
		fileKey: start.fileKey,
		offset:  start.offset,
		length:  uint32(startOffset), // length from start.Start to tr.Start
	}), db.idx.mu.pointers[deleteStart+1:]...)
	// make a new pointer for the second half of end pointer
	db.idx.mu.pointers = append(append(db.idx.mu.pointers[:deleteStart+1], pointer{
		TimeRange: telem.TimeRange{
			Start: tr.End,
			End:   end.End,
		},
		fileKey: end.fileKey,
		offset:  end.offset + uint32(endOffset), // end.offset + length of from tr.End to tr.End
		length:  end.length - uint32(endOffset), // length from tr.End to tr.Start
	}), db.idx.mu.pointers[deleteStart+2:]...)
	return true

}

// Close closes the DB. Close should not be called concurrently with any other DB methods.
func (db *DB) Close() error {
	w := errutil.NewCatch(errutil.WithAggregation())
	w.Exec(db.idx.close)
	w.Exec(db.files.close)
	return w.Error()
}

func (db *DB) newReader(ctx context.Context, ptr pointer) (*Reader, error) {
	internal, err := db.files.acquireReader(ctx, ptr.fileKey)
	if err != nil {
		return nil, err
	}
	reader := io.NewSectionReader(internal, int64(ptr.offset), int64(ptr.length))
	return &Reader{ptr: ptr, ReaderAt: reader}, nil
}

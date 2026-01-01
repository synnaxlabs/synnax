// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package domain implements a database for reading and writing chunks of data (called
// domains) to and from an underlying file system. For information on how to use
// a database, see the DB struct and Open function.
package domain

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// DB provides a persistent, concurrent store for reading and writing domains of
// telemetry to and from an underlying file system.
//
// A DB provides two types for accessing data:
//
//   - Writer allows the caller to write a blob of telemetry occupying a particular time
//     domain.
//
//   - Iterator allows the caller to iterate over the telemetry domains in a DB in time order,
//     and provides an io.Reader like interface for accessing the data.
//
// A DB is safe for concurrent use, and multiple writers and iterators can access the DB
// at once.
//
// It's important to note that a DB is heavily optimized for large (several megabytes to
// gigabytes), append only writes. While small, out-of-order writes are valid, the user
// will see a heavy performance hit.
//
// A DB must be closed after use to avoid leaking any underlying resources/locks.
type DB struct {
	cfg           Config
	idx           *index
	fc            *fileController
	closed        *atomic.Bool
	resourceCount *atomic.Int64
}

// Config is the configuration for opening a DB.
type Config struct {
	alamos.Instrumentation
	// FS is the filesystem that the DB will use to store its data. DB will write to the
	// root of the filesystem, so this should probably be a subdirectory. DB should have
	// exclusive access, and it should be empty when the DB is first opened.
	// [REQUIRED]
	FS fs.FS
	// FileSize is the maximum size, in bytes, for a writer to be created on a file.
	// Note while that a file's size may still exceed this value, it is not likely to
	// exceed by much with frequent commits.
	// [OPTIONAL] Default: 800 MB
	FileSize telem.Size
	// GCThreshold is the minimum tombstone proportion of the FileSize to trigger a GC.
	// Must be in (0, 1].
	// Note: Setting this value to 0 will have NO EFFECT as it is the default value.
	// instead, set it to a very small number greater than 0.
	// [OPTIONAL] Default: 0.2
	GCThreshold float32
	// MaxDescriptors is the maximum number of file descriptors that the DB will use. A
	// higher value will allow more concurrent reads and writes. It's important to note
	// that the exact performance impact of changing this value is still relatively
	// unknown.
	// [OPTIONAL] Default: 100
	MaxDescriptors int
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a DB.
	DefaultConfig = Config{
		FileSize:       800 * telem.Megabyte,
		GCThreshold:    0.2,
		MaxDescriptors: 100,
	}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("domain")
	validate.Positive(v, "file_size", c.FileSize)
	validate.Positive(v, "max_descriptors", c.MaxDescriptors)
	validate.NotNil(v, "fs", c.FS)
	validate.GreaterThanEq(v, "gc_threshold", c.GCThreshold, 0)
	validate.LessThanEq(v, "gc_threshold", c.GCThreshold, 1)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.MaxDescriptors = override.Numeric(c.MaxDescriptors, other.MaxDescriptors)
	c.FileSize = override.Numeric(c.FileSize, other.FileSize)
	c.FS = override.Nil(c.FS, other.FS)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.GCThreshold = override.Numeric(c.GCThreshold, other.GCThreshold)
	// Store 80% of the desired maximum file size as file size since we must leave some
	// buffer for when we stop acquiring a new writer on a file.
	c.FileSize = telem.Size(math.Round(0.8 * float64(c.FileSize)))
	return c
}

// Open opens a DB using a merged view of the provided configurations (where the next
// configuration overrides the previous).
func Open(configs ...Config) (*DB, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	idx := &index{}
	idxPst, err := openIndexPersist(idx, cfg.FS)
	if err != nil {
		return nil, err
	}
	idx.indexPersist = idxPst
	idx.mu.pointers, err = idxPst.load()
	if err != nil {
		return nil, err
	}
	controller, err := openFileController(cfg)
	if err != nil {
		return nil, err
	}
	return &DB{
		cfg:           cfg,
		idx:           idx,
		fc:            controller,
		closed:        &atomic.Bool{},
		resourceCount: &atomic.Int64{},
	}, nil
}

// HasDataFor returns whether any time stamp in the time range tr exists in the
// database.
func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	if db.closed.Load() {
		return false, ErrDBClosed
	}
	i := db.OpenIterator(IterRange(telem.TimeRangeMax))
	if i.SeekGE(ctx, tr.Start) && i.TimeRange().OverlapsWith(tr) {
		return true, i.Close()
	}
	return i.SeekLE(ctx, tr.End) && i.TimeRange().OverlapsWith(tr), i.Close()
}

// Size returns the total size of all data stored in the database by summing the sizes
// of all pointers in the index.
func (db *DB) Size() telem.Size {
	db.idx.mu.RLock()
	defer db.idx.mu.RUnlock()
	var total telem.Size
	for _, p := range db.idx.mu.pointers {
		total += telem.Size(p.size)
	}
	return total
}

// Close closes the DB. Close should not be called concurrently with any other DB
// methods. If close fails for a reason other than unclosed writers/readers, the
// database will still be marked closed and no read/write operations are allowed on it
// to protect data integrity.
func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	count := db.resourceCount.Load()
	if count > 0 {
		err := errors.Wrapf(
			resource.ErrOpen,
			"there are %d unclosed writers/iterators accessing it",
			count,
		)
		db.closed.Store(false)
		return err
	}
	w := errors.NewCatcher(errors.WithAggregation())
	w.Exec(db.fc.close)
	w.Exec(db.idx.close)
	return w.Error()
}

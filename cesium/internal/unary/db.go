// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/meta"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// DB is a database for a single channel. It executes reads (via iterators) and writes
// (via writers) against an underlying domain.DB. It also manages the channel's control
// state, allowing for dynamic handoff between multiple writers.
type DB struct {
	// Config contains validated configuration parameters for the DB.
	cfg Config
	// domain is the underlying domain database on which writes will be executed.
	domain     *domain.DB
	controller *control.Controller[*controlledWriter]
	// _idx is the index used for resolving timestamp positions on this channel.
	_idx             *index.Domain
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
}

// ErrDBClosed is returned when an operation is attempted on a closed unary database.
var ErrDBClosed = core.NewErrResourceClosed("unary.db")

// Channel returns the channel for this unary database.
func (db *DB) Channel() core.Channel { return db.cfg.Channel }

// Index returns the index for the unary database IF AND ONLY IF the channel is an index
// channel. Otherwise, this method will panic.
func (db *DB) Index() *index.Domain {
	if !db.cfg.Channel.IsIndex {
		// inconceivable state
		panic(fmt.Sprintf("channel %v is not an index channel", db.cfg.Channel))
	}
	return db.index()
}

func (db *DB) index() *index.Domain {
	if db._idx == nil {
		// inconceivable state
		panic(fmt.Sprintf("channel <%v> index is not set", db.cfg.Channel))
	}
	return db._idx
}

func (db *DB) SetIndex(idx *index.Domain) { db._idx = idx }

// LeadingControlState returns the first chronological gate in this unary database.
func (db *DB) LeadingControlState() *control.State {
	return db.controller.LeadingState()
}

// HasDataFor check whether there is a time range in the unary DB's underlying domain
// that overlaps with the given time range. Note that this function will return false if
// there is an open writer that could write into the requested time range
func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	if db.closed.Load() {
		return false, ErrDBClosed
	}
	release, err := db.lockControllerForNonWriteOp(tr, "has_data_for")
	if err != nil {
		return true, errors.Skip(err, xcontrol.ErrUnauthorized)
	}
	defer release()
	hasData, err := db.domain.HasDataFor(ctx, tr)
	return hasData, db.wrapError(err)
}

// Read reads a Time Range of data at the unary level.
func (db *DB) Read(ctx context.Context, tr telem.TimeRange) (frame core.Frame, err error) {
	defer func() { err = db.wrapError(err) }()
	var iter *Iterator
	if iter, err = db.OpenIterator(IterRange(tr)); err != nil {
		return frame, err
	}
	defer func() { err = db.wrapError(iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		return frame, err
	}
	for iter.Next(ctx, telem.TimeSpanMax) {
		frame = frame.Extend(iter.Value())
	}
	return frame, err
}

// Size returns the total size of all data stored in the database.
func (db *DB) Size() telem.Size { return db.domain.Size() }

// Close closes the unary database, releasing all resources associated with it. Close
// will return an error if there are any unclosed writers, iterators, or delete
// operations being executed on the database. Close is idempotent, and will return nil
// if the database is already closed.
//
// If close fails for a reason other than unclosed writers/readers, the database will
// still be marked closed and no read/write operations are allowed on it to protect data
// integrity.
func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	if err := db.domain.Close(); err != nil {
		if errors.Is(err, core.ErrOpenResource) {
			// If the close failed because of an open entity, the database should not be
			// marked as closed and can still serve reads/writes.
			db.closed.Store(false)
		}
		return db.wrapError(err)
	}

	return nil
}

// RenameChannelInMeta renames the channel to the given name, and persists the change to
// the underlying file system.
func (db *DB) RenameChannelInMeta(ctx context.Context, newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.Name == newName {
		return nil
	}
	db.cfg.Channel.Name = newName
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetIndexKeyInMeta changes the channel's index to the channel with the given key, and
// persists the change to the underlying file system.
func (db *DB) SetIndexKeyInMeta(ctx context.Context, key core.ChannelKey) error {
	if db.closed.Load() {
		return db.wrapError(ErrDBClosed)
	}
	db.cfg.Channel.Index = key
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetChannelKeyInMeta changes the channel's key to the channel with the given key, and
// persists the change to the underlying file system.
func (db *DB) SetChannelKeyInMeta(ctx context.Context, key core.ChannelKey) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.IsIndex {
		db.cfg.Channel.Index = key
	}
	db.cfg.Channel.Key = key
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

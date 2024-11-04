// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
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
	controller *controller.Controller[*controlledWriter]
	// _idx is the index used for resolving timestamp positions on this channel.
	_idx             index.Index
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
}

// ErrDBClosed is returned when an operation is attempted on a closed unary database.
var ErrDBClosed = core.EntityClosed("unary.db")

// Channel returns the channel for this unary database.
func (db *DB) Channel() core.Channel { return db.cfg.Channel }

// Index returns the index for the unary database IF AND ONLY IF the channel is an index
// channel. Otherwise, this method will panic.
func (db *DB) Index() index.Index {
	if !db.cfg.Channel.IsIndex {
		// inconceivable state
		panic(fmt.Sprintf("channel %v is not an index channel", db.cfg.Channel))
	}
	return db.index()
}

func (db *DB) index() index.Index {
	if db._idx == nil {
		// inconceivable state
		panic(fmt.Sprintf("channel <%v> index is not set", db.cfg.Channel))
	}
	return db._idx
}

func (db *DB) SetIndex(idx index.Index) { db._idx = idx }

func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds.Start = override.Numeric(i.Bounds.Start, other.Bounds.Start)
	i.Bounds.End = override.Numeric(i.Bounds.End, other.Bounds.End)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	return i
}

func (i IteratorConfig) domainIteratorConfig() domain.IteratorConfig {
	return domain.IteratorConfig{Bounds: i.Bounds}
}

// LeadingControlState returns the first chronological gate in this unary database.
func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) OpenIterator(cfg IteratorConfig) *Iterator {
	cfg = DefaultIteratorConfig.Override(cfg)
	iter := db.domain.OpenIterator(cfg.domainIteratorConfig())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.cfg.Channel,
		internal:       iter,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i
}

// HasDataFor check whether there is a time range in the unary DB's underlying domain that
// overlaps with the given time range. Note that this function will return false if there
// is an open writer that could write into the requested time range
func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	if db.closed.Load() {
		return false, ErrDBClosed
	}
	g, _, err := db.controller.OpenAbsoluteGateIfUncontrolled(
		tr,
		control.Subject{Key: "has_data_for"},
		func() (*controlledWriter, error) {
			return &controlledWriter{Writer: nil, channelKey: db.cfg.Channel.Key}, nil
		})

	if err != nil {
		if errors.Is(err, control.Unauthorized) {
			return true, nil
		}
		return true, err
	}

	_, ok := g.Authorized()
	if !ok {
		return true, nil
	}
	defer g.Release()

	ok, err = db.domain.HasDataFor(ctx, tr)
	return ok, db.wrapError(err)
}

// Read reads a Time Range of data at the unary level.
func (db *DB) Read(ctx context.Context, tr telem.TimeRange) (frame core.Frame, err error) {
	defer func() { err = db.wrapError(err) }()
	if db.closed.Load() {
		return frame, ErrDBClosed
	}
	iter := db.OpenIterator(IterRange(tr))
	if err != nil {
		return
	}
	defer func() { err = iter.Close() }()
	if !iter.SeekFirst(ctx) {
		return
	}
	for iter.Next(ctx, telem.TimeSpanMax) {
		frame = frame.AppendFrame(iter.Value())
	}
	return
}

// Close closes the unary database, releasing all resources associated with it. Close
// will return an error if there are any unclosed writers, iterators, or delete
// operations being executed on the database. Close is idempotent, and will return nil
// if the database is already closed.
func (db *DB) Close() error {
	if db.closed.Load() {
		return nil
	}
	err := db.domain.Close()
	if err != nil {
		return db.wrapError(err)
	}

	db.closed.Store(true)
	return nil
}

// RenameChannelInMeta renames the channel to the given name, and persists the change to the
// underlying file system.
func (db *DB) RenameChannelInMeta(newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.Name == newName {
		return nil
	}
	db.cfg.Channel.Name = newName
	return meta.Create(db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetIndexKeyInMeta changes the channel's index to the channel with the given key,
// and persists the change to the underlying file system.
func (db *DB) SetIndexKeyInMeta(key core.ChannelKey) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.cfg.Channel.Index = key
	return meta.Create(db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

// SetChannelKeyInMeta changes the channel's key to the channel with the given key,
// and persists the change to the underlying file system.
func (db *DB) SetChannelKeyInMeta(key core.ChannelKey) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.IsIndex {
		db.cfg.Channel.Index = key
	}
	db.cfg.Channel.Key = key
	return meta.Create(db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

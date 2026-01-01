// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"io"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
	Frame      = channel.Frame
)

var (
	errDBClosed          = channel.NewErrResourceClosed("cesium.db")
	ErrChannelNotFound   = channel.ErrNotFound
	ZeroLeadingAlignment = alignment.ZeroLeading
)

// Metrics contains statistics about the cesium database.
type Metrics struct {
	// DiskSize is the total disk space used by all channel data.
	DiskSize telem.Size
	// ChannelCount is the number of channels in the database.
	ChannelCount int
}

// LeadingAlignment returns an Alignment whose array index is the maximum possible value
// and whose sample index is the provided value.
func LeadingAlignment(domainIdx, sampleIdx uint32) telem.Alignment {
	return alignment.Leading(domainIdx, sampleIdx)
}

type DB struct {
	*options
	relay *relay
	mu    struct {
		sync.RWMutex
		unaryDBs   map[ChannelKey]unary.DB
		virtualDBs map[ChannelKey]virtual.DB
		digests    struct {
			key      ChannelKey
			shutdown io.Closer
			inlet    confluence.Inlet[WriterRequest]
			outlet   confluence.Outlet[WriterResponse]
		}
	}
	closed   *atomic.Bool
	shutdown io.Closer
}

// Write writes the frame to database at the specified start time.
func (db *DB) Write(ctx context.Context, start telem.TimeStamp, frame Frame) error {
	if db.closed.Load() {
		return errDBClosed
	}
	_, span := db.T.Bench(ctx, "write")
	defer span.End()
	w, err := db.OpenWriter(ctx, WriterConfig{Start: start, Channels: frame.KeysSlice()})
	if err != nil {
		return span.Error(err)
	}
	if _, err = w.Write(frame); err != nil {
		return err
	}
	if _, err = w.Commit(); err != nil {
		return err
	}
	return span.Error(w.Close())
}

// WriteSeries writes a series into the specified channel at the specified start time.
func (db *DB) WriteSeries(ctx context.Context, key channel.Key, start telem.TimeStamp, series telem.Series) error {
	if db.closed.Load() {
		return errDBClosed
	}
	return db.Write(ctx, start, telem.UnaryFrame(key, series))
}

// Read reads from the database at the specified time range and outputs a frame.
func (db *DB) Read(ctx context.Context, tr telem.TimeRange, keys ...channel.Key) (frame Frame, err error) {
	if db.closed.Load() {
		return frame, errDBClosed
	}
	_, span := db.T.Bench(ctx, "read")
	defer func() { err = span.EndWith(err) }()
	iter, err := db.OpenIterator(IteratorConfig{Channels: keys, Bounds: tr})
	if err != nil {
		return frame, err
	}
	defer func() { err = iter.Close() }()
	if !iter.SeekFirst() {
		return frame, err
	}
	for iter.Next(telem.TimeSpanMax) {
		frame = frame.Extend(iter.Value())
	}
	return frame, err
}

// Metrics returns current metrics for the database.
func (db *DB) Metrics() Metrics {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var size telem.Size
	for _, u := range db.mu.unaryDBs {
		size += u.Size()
	}
	return Metrics{
		DiskSize:     size,
		ChannelCount: len(db.mu.unaryDBs) + len(db.mu.virtualDBs),
	}
}

// Close closes the database.
//
// Close is not safe to call with any other DB methods concurrently.
//
// Note that if this method is called while writers are still open on channels in the
// database, a deadlock is caused since the signal context is closed while the writers
// attempt to send to relay.
//
// If there is an error in closing the cesium database, the database will be marked as
// closed regardless of whether an error occurred.
func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}

	c := errors.NewCatcher(errors.WithAggregation())
	// Crucial to close control digests here before closing the signal context so writes
	// can still use the signal context to send frames to relay.
	//
	// This function acquires the mutex lock internally, so there's no need to lock it
	// here.
	c.Exec(db.closeControlDigests)
	// Shut down without locking mutex to allow existing goroutines (e.g. GC) that
	// require a mutex lock to exit.
	c.Exec(db.shutdown.Close)
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, u := range db.mu.unaryDBs {
		c.Exec(u.Close)
	}
	return c.Error()
}

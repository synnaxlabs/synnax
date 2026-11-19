// Copyright 2026 Synnax Labs, Inc.
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
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
	Frame      = channel.Frame
)

var (
	ErrDBClosed        = resource.NewClosedError("cesium.db")
	ErrChannelNotFound = channel.ErrNotFound
)

// Metrics contains statistics about the cesium database.
type Metrics struct {
	// DiskSize is the total disk space used by all channel data.
	DiskSize telem.Size
}

type DB struct {
	// removals tracks the background removals of deleted channel data.
	removals sync.WaitGroup
	*options
	// controlUpdates notifies subscribers of every control transfer the DB arbitrates.
	controlUpdates controlObserver
	historyChanges observe.Observer[HistoryChange]
	closed         *atomic.Bool
	// lifecycle orders the start of Run against Close.
	lifecycle struct {
		sync.Mutex
		state runState
	}
	mu struct {
		dbs struct {
			unary   map[ChannelKey]unary.DB
			virtual map[ChannelKey]virtual.DB
		}
		sync.RWMutex
	}
}

// Write writes the frame to database at the specified start time. It returns
// control.ErrUnauthorized if another writer controls a channel in the frame, and then
// it may have committed only some of the channels.
func (db *DB) Write(ctx context.Context, start telem.TimeStamp, frame Frame) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	_, span := db.T.Bench(ctx, "write")
	defer span.End()
	w, err := db.OpenWriter(
		ctx,
		WriterConfig{Start: start, Channels: frame.KeysSlice()},
	)
	if err != nil {
		return span.Error(err)
	}
	// A write refused for no control still writes the channels the writer controls,
	// so they commit.
	if _, err = w.Write(
		ctx,
		frame,
	); err == nil ||
		errors.Is(err, xcontrol.ErrUnauthorized) {
		_, commitErr := w.Commit(ctx)
		err = errors.Combine(err, commitErr)
	}
	if errors.Is(err, xcontrol.ErrUnauthorized) {
		err = errors.Wrap(err, "another writer controls the frame")
	}
	return span.Error(errors.Combine(err, w.Close()))
}

// WriteSeries writes a series into the specified channel at the specified start time.
func (db *DB) WriteSeries(
	ctx context.Context,
	key channel.Key,
	start telem.TimeStamp,
	series telem.Series,
) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	return db.Write(ctx, start, telem.UnaryFrame(key, series))
}

// Read reads from the database at the specified time range and outputs a frame.
func (db *DB) Read(
	ctx context.Context,
	tr telem.TimeRange,
	keys ...channel.Key,
) (frame Frame, err error) {
	if db.closed.Load() {
		return frame, ErrDBClosed
	}
	_, span := db.T.Bench(ctx, "read")
	defer func() { err = span.EndWith(err) }()
	iter, err := db.OpenIterator(IteratorConfig{Channels: keys, Bounds: tr})
	if err != nil {
		return frame, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
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
	for _, u := range db.mu.dbs.unary {
		size += u.Size()
	}
	return Metrics{DiskSize: size}
}

// runState is the stage of DB.Run.
type runState uint8

const (
	runIdle runState = iota
	runRunning
	runStopped
)

// Run runs the garbage collector until ctx ends. It then waits for it to stop and
// returns its error. Call Run once, after Open and before Close. A second Run returns
// an error, and a Run after Close returns ErrDBClosed.
func (db *DB) Run(ctx context.Context) error {
	db.lifecycle.Lock()
	if db.closed.Load() {
		db.lifecycle.Unlock()
		return ErrDBClosed
	}
	if db.lifecycle.state != runIdle {
		db.lifecycle.Unlock()
		return errors.New("run called twice")
	}
	db.lifecycle.state = runRunning
	db.lifecycle.Unlock()
	defer func() {
		db.lifecycle.Lock()
		db.lifecycle.state = runStopped
		db.lifecycle.Unlock()
	}()
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithInstrumentation(db.Instrumentation),
	)
	defer cancel()
	db.startGC(sCtx, db.options)
	return errors.Skip(sCtx.Wait(), context.Canceled)
}

// Close closes the database. Call it only after Run returns: while Run runs, Close
// returns an error wrapping resource.ErrOpen and closes nothing. Close is not safe to
// call with any other DB methods concurrently.
//
// If there is an error in closing the cesium database, the database will be marked as
// closed regardless of whether an error occurred.
func (db *DB) Close() error {
	db.lifecycle.Lock()
	if db.lifecycle.state == runRunning {
		db.lifecycle.Unlock()
		return errors.Wrap(resource.ErrOpen, "close called while Run runs")
	}
	closing := db.closed.CompareAndSwap(false, true)
	db.lifecycle.Unlock()
	if !closing {
		return nil
	}
	var err error
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, u := range db.mu.dbs.unary {
		err = errors.Join(err, u.Close())
	}
	db.removals.Wait()
	return err
}

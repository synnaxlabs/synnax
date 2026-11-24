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
	"github.com/synnaxlabs/cesium/internal/claim"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lifetime"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
	Frame      = channel.Frame
	// Concurrency is the concurrency of a channel's writers, kept in its metadata.
	Concurrency = channel.Concurrency
	// ChannelError is an error of one channel. It carries the key of the channel, never
	// its name, so a caller that holds channel records can name it.
	ChannelError = channel.Error
)

// ClaimError is the refusal of a writer, a commit, or a delete that reaches the time
// range another writer claims. It names the time range, never the writer.
type ClaimError = claim.Error

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
	*options
	historyChanges observe.Observer[HistoryChange]
	closed         *atomic.Bool
	// intents makes the delete intents of the DB durable.
	intents *intents
	// corruptIntents is the number of intent records that Open found bad before a good
	// one.
	corruptIntents int
	// engine is the pointer log layout of the DB, and nil on the old layout.
	engine *engine
	// run orders the start of Run against Close, and started is true once Run began.
	run     lifetime.Guard
	started atomic.Bool
	mu      struct {
		dbs struct {
			unary map[ChannelKey]unary.DB
			// hidden holds the channels that Sweep keeps: they hold samples but have no
			// record and no intent. Writers and iterators cannot open on them.
			hidden map[ChannelKey]unary.DB
		}
		// channels is the channel list of the pointer log layout, and hiddenChannels
		// holds its channels that Sweep keeps.
		channels       map[ChannelKey]Channel
		hiddenChannels map[ChannelKey]Channel
		// torn holds the keys of the pointer log layout whose channel file a cut-short
		// create left, which Sweep removes.
		torn set.Set[ChannelKey]
		// virtualDirs holds the keys of the directories whose metadata names a virtual
		// channel. A Core older than this Cesium wrote them, and an undo to it reads
		// them, so Open leaves them in place.
		virtualDirs set.Set[ChannelKey]
		// skipped holds the keys whose directories Open skipped for their intent.
		skipped set.Set[ChannelKey]
		// swept is true once Sweep has run. kept holds the keys that it kept, and
		// sweepErr its error.
		swept    bool
		kept     set.Set[ChannelKey]
		sweepErr error
		sync.RWMutex
	}
}

// lockRoot takes db.mu for a call that removes files or directories of the root. On the
// pointer log, it first waits for the orphan pass of Run, which lists and removes files
// with no lock. It returns the error of ctx, and takes no lock, when ctx ends first.
func (db *DB) lockRoot(ctx context.Context) error {
	if db.engine != nil {
		select {
		case <-db.engine.ready:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	db.mu.Lock()
	return nil
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

// CorruptIntents returns the number of batches of delete intents that Open found
// damaged, on two reads, before an intact one. Each Intend appends one batch, and a
// crash tears only the last, so such a batch is damage on disk. The data of a damaged
// intent's channel stays, and Sweep keeps it when no record holds its key.
func (db *DB) CorruptIntents() int { return db.corruptIntents }

// Metrics returns current metrics for the database.
func (db *DB) Metrics() Metrics {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var size telem.Size
	for _, u := range db.mu.dbs.unary {
		size += u.Size()
	}
	for _, u := range db.mu.dbs.hidden {
		size += u.Size()
	}
	return Metrics{DiskSize: size}
}

// Run runs the garbage collector until ctx ends. It then waits for it to stop and
// returns its error. Call Run once, after Open and before Close. A second Run returns
// an error, and a Run after Close returns ErrDBClosed.
func (db *DB) Run(ctx context.Context) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if err := db.run.Begin(); err != nil {
		return err
	}
	defer db.run.End()
	db.started.Store(true)
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithInstrumentation(db.Instrumentation),
	)
	defer cancel()
	if db.engine != nil {
		return db.engine.run(ctx)
	}
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
	first, err := db.run.Close()
	if err != nil {
		return errors.Wrap(resource.ErrOpen, err.Error())
	}
	if !first {
		return nil
	}
	db.closed.Store(true)
	if db.engine != nil {
		if db.started.Load() {
			return db.engine.close()
		}
		// The owners close only while they run, so a DB that never ran runs them with a
		// done context, which closes them in order.
		done, cancel := context.WithCancel(context.Background())
		cancel()
		return db.engine.run(done)
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, u := range db.mu.dbs.unary {
		err = errors.Join(err, u.Close())
	}
	for _, u := range db.mu.dbs.hidden {
		err = errors.Join(err, u.Close())
	}
	return errors.Join(err, db.intents.Close())
}

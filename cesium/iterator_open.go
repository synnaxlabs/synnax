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
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

func (db *DB) OpenIterator(cfg IteratorConfig) (*iterator, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	internal, err := db.newStreamIterator(cfg)
	if err != nil {
		// return early to prevent panic in wrapStreamIterator
		return nil, err
	}
	return wrapStreamIterator(internal), nil
}

// NewStreamIterator opens a segment that reads the DB in time order, as OpenIterator
// does. Each IteratorRequest sent to its inlet gets zero or more responses with frames,
// then one response of the IteratorResponseVariantAck variant with the command, the
// next sequence number, and whether the command succeeded. Close the inlet to close the
// segment: it serves the requests in progress, closes its outlet, and returns any error
// through the context passed to Flow.
func (db *DB) NewStreamIterator(
	cfg IteratorConfig,
) (confluence.Segment[IteratorRequest, IteratorResponse], error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.newStreamIterator(cfg)
}

func (db *DB) newStreamIterator(cfg IteratorConfig) (si *streamIterator, err error) {
	var (
		internal []*unary.Iterator
		cache    = index.NewCache()
		clocks   map[ChannelKey]*logClock
	)
	if db.engine != nil {
		clocks = make(map[ChannelKey]*logClock, 1)
	}
	defer func() {
		if err == nil {
			return
		}
		for _, iter := range internal {
			if iter != nil {
				err = errors.Combine(err, iter.Close())
			}
		}
	}()
	for _, key := range cfg.Channels {
		ucfg := unary.IteratorConfig{
			Bounds:        cfg.Bounds,
			AutoChunkSize: cfg.AutoChunkSize,
			Reduction:     cfg.Reduction,
			Cache:         cache,
		}
		if db.engine != nil {
			iter, iterErr := db.openLogIterator(key, ucfg, clocks)
			if iterErr != nil {
				return nil, iterErr
			}
			internal = append(internal, iter)
			continue
		}
		if uDB, ok := db.mu.dbs.unary[key]; ok {
			iter, iterErr := uDB.OpenIterator(ucfg)
			if iterErr != nil {
				return nil, iterErr
			}
			internal = append(internal, iter)
			continue
		}
		return nil, channel.NewNotFoundError(key)
	}
	_, limited := cfg.Reduction.Variant.(telem.LimitReduction)
	views := make([]*logView, 0, len(clocks))
	for _, c := range clocks {
		views = append(views, c.view)
	}
	return &streamIterator{
		internal: internal,
		validateCommand: validate.NewInclusiveBoundsChecker(
			IteratorCommandNext,
			IteratorCommandSetBounds,
		),
		cache:     cache,
		views:     views,
		limited:   limited,
		chunkSize: db.chunkSize,
	}, nil
}

// logClock is the view and the index of a clock, which the keys of a read on it share.
type logClock struct {
	view *logView
	idx  *index.Domain
}

// openLogIterator opens the iterator of key on the pointer log layout, which takes the
// stamps of a data channel from its clock. The keys on one clock share its view and its
// index in clocks, so they read one state in each step and share the lookups of
// cfg.Cache. The caller holds db.mu.
func (db *DB) openLogIterator(
	key ChannelKey,
	cfg unary.IteratorConfig,
	clocks map[ChannelKey]*logClock,
) (*unary.Iterator, error) {
	ch, ok := db.mu.channels[key]
	if !ok {
		return nil, channel.NewNotFoundError(key)
	}
	clock := ch.Clock()
	c, ok := clocks[clock]
	if !ok {
		view := newLogView(db.engine, clock)
		c = &logClock{view: view, idx: &index.Domain{
			Instrumentation: db.Instrumentation,
			DB:              &logSource{view: view, clock: true},
			Channel:         db.mu.channels[clock],
		}}
		clocks[clock] = c
	}
	data := &logSource{view: c.view, slot: c.view.add(key)}
	return unary.NewIterator(db.Instrumentation, ch, data, c.idx, cfg)
}

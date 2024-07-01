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
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"sync"
	"sync/atomic"
)

type controlledWriter struct {
	*domain.Writer
	channelKey core.ChannelKey
}

func (w controlledWriter) ChannelKey() core.ChannelKey { return w.channelKey }

type entityCount struct {
	sync.RWMutex
	openIteratorWriters int
}

func (c *entityCount) add(delta int) {
	c.Lock()
	c.openIteratorWriters += delta
	c.Unlock()
}

type DB struct {
	Config
	Domain      *domain.DB
	Controller  *controller.Controller[controlledWriter]
	_idx        index.Index
	entityCount *entityCount
	wrapError   func(error) error
	closed      *atomic.Bool
}

var errDBClosed = core.EntityClosed("unary.db")

func (db *DB) Index() index.Index {
	if !db.Channel.IsIndex {
		// inconceivable state
		panic(fmt.Sprintf("channel %v is not an index channel", db.Channel))
	}
	return db.index()
}

func (db *DB) index() index.Index {
	if db._idx == nil {
		// inconceivable state
		panic(fmt.Sprintf("channel <%v> index is not set", db.Channel))
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
	return db.Controller.LeadingState()
}

func (db *DB) OpenIterator(cfg IteratorConfig) *Iterator {
	cfg = DefaultIteratorConfig.Override(cfg)
	iter := db.Domain.NewIterator(cfg.domainIteratorConfig())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.Channel,
		internal:       iter,
		IteratorConfig: cfg,
		onClose: func() {
			db.entityCount.add(-1)
		},
	}
	i.SetBounds(cfg.Bounds)

	db.entityCount.add(1)
	return i
}

// HasDataFor check whether there is a timerange in the unary DB's underlying domain that
// overlaps with the given timerange. Note that this function will return false if there
// is an open writer that could write into the requested timerange
func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	if db.closed.Load() {
		return false, errDBClosed
	}
	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(tr, control.Subject{Key: "has_data_for"},
		func() (controlledWriter, error) {
			return controlledWriter{
				Writer:     nil,
				channelKey: db.Channel.Key,
			}, nil
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

	ok, err = db.Domain.HasDataFor(ctx, tr)
	return ok, db.wrapError(err)
}

// Read reads a timerange of data at the unary level.
func (db *DB) Read(ctx context.Context, tr telem.TimeRange) (frame core.Frame, err error) {
	defer func() { err = db.wrapError(err) }()

	if db.closed.Load() {
		return frame, errDBClosed
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

func (db *DB) Close() error {
	if db.closed.Load() {
		return nil
	}

	db.entityCount.RLock()
	defer db.entityCount.RUnlock()
	if db.entityCount.openIteratorWriters > 0 {
		return db.wrapError(errors.Newf("cannot close channel because there are %d unclosed writers/iterators accessing it", db.entityCount.openIteratorWriters))
	} else {
		db.closed.Store(true)
		return db.wrapError(db.Domain.Close())
	}
}

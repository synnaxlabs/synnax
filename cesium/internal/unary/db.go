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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type controlledWriter struct {
	*domain.Writer
	channelKey core.ChannelKey
}

func (w controlledWriter) ChannelKey() core.ChannelKey { return w.channelKey }

type openEntityCount struct {
	sync.RWMutex
	openIteratorWriters int
}

func (c *openEntityCount) Add(delta int) {
	c.Lock()
	c.openIteratorWriters += delta
	c.Unlock()
}

type DB struct {
	Config
	Domain     *domain.DB
	Controller *controller.Controller[controlledWriter]
	_idx       index.Index
	mu         *openEntityCount
}

func (db *DB) Index() index.Index {
	if !db.Channel.IsIndex {
		panic(fmt.Sprintf("[control.unary] - database %v does not support indexing", db.Channel.Key))
	}
	return db.index()
}

func (db *DB) index() index.Index {
	if db._idx == nil {
		panic("[ranger.unary] - index is not set")
	}
	return db._idx
}

func (db *DB) SetIndex(idx index.Index) { db._idx = idx }

type IteratorConfig struct {
	Bounds telem.TimeRange
	// AutoChunkSize sets the maximum size of a chunk that will be returned by the
	// iterator when using AutoSpan in calls ot Next or Prev.
	AutoChunkSize int64
}

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: domain.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var (
	DefaultIteratorConfig = IteratorConfig{AutoChunkSize: 5e5}
)

func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds.Start = override.Numeric(i.Bounds.Start, other.Bounds.Start)
	i.Bounds.End = override.Numeric(i.Bounds.End, other.Bounds.End)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	return i
}

func (i IteratorConfig) ranger() domain.IteratorConfig {
	return domain.IteratorConfig{Bounds: i.Bounds}
}

func (db *DB) LeadingControlState() *controller.State {
	return db.Controller.LeadingState()
}

func (db *DB) OpenIterator(cfg IteratorConfig) *Iterator {
	cfg = DefaultIteratorConfig.Override(cfg)
	iter := db.Domain.NewIterator(cfg.ranger())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.Channel,
		internal:       iter,
		IteratorConfig: cfg,
		onClose: func() {
			db.mu.Add(-1)
		},
	}
	i.SetBounds(cfg.Bounds)

	db.mu.Add(1)
	return i
}

// HasDataFor check whether there is a timerange in the unary DB's underlying domain that
// overlaps with the given timerange. Note that this function will return false if there
// is an open writer that could write into the requested timerange
func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	g, _, err := db.Controller.OpenAbsoluteGateIfUncontrolled(tr, control.Subject{Key: "Delete Writer"}, func() (controlledWriter, error) {
		return controlledWriter{
			Writer:     nil,
			channelKey: db.Channel.Key,
		}, nil
	})

	if err != nil {
		return true, err
	}

	_, ok := g.Authorize()
	if !ok {
		g.Release()
		return true, nil
	}

	return db.Domain.HasDataFor(ctx, tr)
}

// Read reads a timerange of data at the unary level.
func (db *DB) Read(ctx context.Context, tr telem.TimeRange) (frame core.Frame, err error) {
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

func (db *DB) TryClose() error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.mu.openIteratorWriters > 0 {
		return errors.Newf("[cesium] - cannot close channel because there are currently %d unclosed writers/iterators accessing it", db.mu.openIteratorWriters)
	} else {
		return db.Close()
	}
}

func (db *DB) Close() error { return db.Domain.Close() }

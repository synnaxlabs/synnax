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
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

type writeControlState struct {
	counter int
	writers []*Writer
}

type DB struct {
	Config
	Domain  *domain.DB
	writers map[*domain.Writer]*writeControlState
	mu      sync.RWMutex
	_idx    index.Index
}

func (db *DB) Index() index.Index {
	if !db.Channel.IsIndex {
		panic(fmt.Sprintf("[domain.unary] - database %v does not support indexing", db.Channel.Key))
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

func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (w *Writer, err error) {
	w = &Writer{WriterConfig: cfg, Channel: db.Channel, idx: db.index()}
	db.mu.Lock()
	for d, e := range db.writers {
		if d.Domain().OverlapsWith(cfg.domain().Domain()) {
			w.internal = d
			e.counter++
			w.pos = e.counter
		}
	}
	if w.internal == nil {
		w.internal, err = db.Domain.NewWriter(ctx, cfg.domain())
		if err != nil {
			db.mu.Unlock()
			return nil, err
		}
		w.pos = 1
		db.writers[w.internal].writers = append(db.writers[w.internal].writers, w)
	}
	db.mu.Unlock()
	return
}

func (db *DB) removeWriter(w *Writer) error {
	db.mu.Lock()
	d, ok := db.writers[w.internal]
	if !ok {
		panic(fmt.Sprintf("[domain.unary] - writer %v not found in database %v", w.internal, db.Channel.Key))
	}
	if len(d.writers) == 1 {
		delete(db.writers, w.internal)
		db.mu.Unlock()
		return w.internal.Close()
	}
	db.writers[w.internal].writers = append(d.writers[:w.pos], d.writers[w.pos+1:]...)
	db.mu.Unlock()
	return nil
}

func (db *DB) authorize(w *Writer) bool {
	db.mu.RLock()
	d, ok := db.writers[w.internal]
	if !ok {
		panic(fmt.Sprintf("[domain.unary] - writer %v not found in database %v", w.internal, db.Channel.Key))
	}
	for i, ow := range d.writers {
		if ow == w {
			continue
		}
		if ow.Authority > w.Authority || (ow.Authority == w.Authority && i < w.pos) {
			db.mu.RUnlock()
			return false
		}
	}
	db.mu.RUnlock()
	return true
}

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

func (db *DB) OpenIterator(cfg IteratorConfig) *Iterator {
	cfg = DefaultIteratorConfig.Override(cfg)
	iter := db.Domain.NewIterator(cfg.ranger())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.Channel,
		internal:       iter,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i
}

func (db *DB) Close() error { return db.Domain.Close() }

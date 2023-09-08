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
)

type DB struct {
	Config
	Ranger *domain.DB
	_idx   index.Index
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

func (db *DB) OpenWriter(ctx context.Context, cfg domain.WriterConfig) (*Writer, error) {
	w, err := db.Ranger.NewWriter(ctx, cfg)
	return &Writer{start: cfg.Start, Channel: db.Channel, internal: w, idx: db.index()}, err
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
	iter := db.Ranger.NewIterator(cfg.ranger())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.Channel,
		internal:       iter,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i
}

func (db *DB) Close() error { return db.Ranger.Close() }

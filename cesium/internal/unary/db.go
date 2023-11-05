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
	"fmt"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

type controlledWriter struct {
	*domain.Writer
	channelKey core.ChannelKey
}

func (w controlledWriter) ChannelKey() core.ChannelKey { return w.channelKey }

type DB struct {
	Config
	Domain       *domain.DB
	Controller   *controller.Controller[controlledWriter]
	_idx         index.Index
	Open_writers int64
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
	}
	i.SetBounds(cfg.Bounds)
	db.Open_writers += 1
	return i
}

func (db *DB) IsWrittenTo() bool {
	return db.Open_writers == 0
}

func (db *DB) Close() error { return db.Domain.Close() }

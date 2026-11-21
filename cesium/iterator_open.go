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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

func (db *DB) OpenIterator(cfg IteratorConfig) (*Iterator, error) {
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

func (db *DB) NewStreamIterator(cfg IteratorConfig) (StreamIterator, error) {
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
	)
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
			iter, iterErr := db.openLogIterator(key, ucfg)
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
	return &streamIterator{
		internal: internal,
		validateCommand: validate.NewInclusiveBoundsChecker(
			IteratorCommandNext,
			IteratorCommandSetBounds,
		),
		cache:     cache,
		limited:   limited,
		chunkSize: db.chunkSize,
	}, nil
}

// openLogIterator opens the iterator of key on the pointer log layout, which takes the
// stamps of a data channel from its clock. The caller holds db.mu.
func (db *DB) openLogIterator(
	key ChannelKey,
	cfg unary.IteratorConfig,
) (*unary.Iterator, error) {
	ch, ok := db.mu.channels[key]
	if !ok {
		return nil, channel.NewNotFoundError(key)
	}
	clock := ch.Clock()
	idx := &index.Domain{
		Instrumentation: db.Instrumentation,
		DB:              logSource{e: db.engine, key: clock},
		Channel:         db.mu.channels[clock],
	}
	return unary.NewIterator(
		db.Instrumentation,
		ch,
		logSource{e: db.engine, key: key},
		idx,
		cfg,
	)
}

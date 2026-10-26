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
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
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
	var internal []channelIterator
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
		if uDB, ok := db.mu.dbs.unary[key]; ok {
			iter, iterErr := uDB.OpenIterator(unary.IteratorConfig{
				Bounds:        cfg.Bounds,
				AutoChunkSize: cfg.AutoChunkSize,
				Reduction:     cfg.Reduction,
			})
			if iterErr != nil {
				return nil, iterErr
			}
			internal = append(internal, iter)
			continue
		}
		if vDB, ok := db.mu.dbs.virtual[key]; ok {
			iter, iterErr := vDB.OpenIterator(
				virtual.IteratorConfig{Bounds: cfg.Bounds},
			)
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
		internal:  internal,
		limited:   limited,
		chunkSize: db.chunkSize,
	}, nil
}

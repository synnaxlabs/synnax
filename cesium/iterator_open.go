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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// OpenIterator opens an iterator over the channels of cfg. It returns an error for a
// channel that db does not hold, and for a virtual channel, which stores no data.
func (db *DB) OpenIterator(cfg IteratorConfig) (iter *Iterator, err error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	var (
		internal []*unary.Iterator
		cache    = index.NewCache()
	)
	defer func() {
		if err == nil {
			return
		}
		for _, u := range internal {
			err = errors.Combine(err, u.Close())
		}
	}()
	for _, key := range cfg.Channels {
		if uDB, ok := db.mu.dbs.unary[key]; ok {
			u, err := uDB.OpenIterator(unary.IteratorConfig{
				Bounds:        cfg.Bounds,
				AutoChunkSize: cfg.AutoChunkSize,
				Reduction:     cfg.Reduction,
				Cache:         cache,
			})
			if err != nil {
				return nil, err
			}
			internal = append(internal, u)
			continue
		}
		if vdb, ok := db.mu.dbs.virtual[key]; ok {
			return nil, errors.Wrapf(
				query.ErrUnsupported,
				"cannot open iterator on virtual channel %v",
				vdb.Channel,
			)
		}
		return nil, channel.NewNotFoundError(key)
	}
	_, limited := cfg.Reduction.Variant.(telem.LimitReduction)
	return &Iterator{
		internal:  internal,
		cache:     cache,
		limited:   limited,
		chunkSize: db.chunkSize,
	}, nil
}

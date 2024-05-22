// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errors"
)

func (db *DB) OpenIterator(cfg IteratorConfig) (*Iterator, error) {
	internal, err := db.newStreamIterator(cfg)
	if err != nil {
		// return early to prevent panic in wrapStreamIterator
		return nil, err
	}
	return wrapStreamIterator(internal), nil
}

func (db *DB) NewStreamIterator(cfg IteratorConfig) (StreamIterator, error) {
	return db.newStreamIterator(cfg)
}

func (db *DB) newStreamIterator(cfg IteratorConfig) (*streamIterator, error) {
	internal := make([]*unary.Iterator, len(cfg.Channels))
	db.mu.RLock()
	defer db.mu.RUnlock()
	for i, key := range cfg.Channels {
		uDB, ok := db.unaryDBs[key]
		if !ok {
			_, vOk := db.virtualDBs[key]
			if vOk {
				return nil, errors.Newf("cannot open iterator on virtual channel %d", key)
			}
			return nil, core.ChannelNotFound(key)
		}
		internal[i] = uDB.OpenIterator(unary.IteratorConfig{Bounds: cfg.Bounds})
	}

	return &streamIterator{internal: internal}, nil
}

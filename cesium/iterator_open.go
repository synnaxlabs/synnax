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
	"github.com/synnaxlabs/cesium/internal/unary"
)

func (db *DB) OpenIterator(cfg IteratorConfig) (*Iterator, error) {
	internal, err := db.newStreamIterator(cfg)
	return wrapStreamIterator(internal), err
}

func (db *DB) NewStreamIterator(cfg IteratorConfig) (StreamIterator, error) {
	return db.newStreamIterator(cfg)
}

func (db *DB) newStreamIterator(cfg IteratorConfig) (*streamIterator, error) {
	internal := make([]*unary.Iterator, len(cfg.Channels))
	for i, key := range cfg.Channels {
		uDB, err := db.getUnary(key)
		if err != nil {
			return nil, err
		}
		internal[i] = uDB.OpenIterator(unary.IteratorConfig{Bounds: cfg.Bounds})
	}

	return &streamIterator{internal: internal}, nil
}

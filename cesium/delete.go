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
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
	"strconv"
)

func (db *DB) DeleteChannel(ch ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if uok {
		if udb.Config.Channel.IsIndex {
			for otherDBKey := range db.unaryDBs {
				if otherDBKey == ch {
					continue
				}
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel.Index == udb.Config.Channel.Key {
					return errors.New("Could not delete index channel with other channels depending on it")
				}
			}
		}

		if err := udb.TryClose(); err != nil {
			return err
		}
		delete(db.unaryDBs, ch)
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if db.digests.key == ch {
			return errors.New("[cesium] - cannot delete update digest channel")
		}
		if err := vdb.TryClose(); err != nil {
			return err
		}
		delete(db.virtualDBs, ch)
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}

	return ChannelNotFound
}

func (db *DB) DeleteTimeRange(ctx context.Context, ch ChannelKey, tr telem.TimeRange) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if !uok {
		return ChannelNotFound
	}

	// cannot delete an index channel that other channels rely on
	if udb.Config.Channel.IsIndex {
		for otherDBKey := range db.unaryDBs {
			if otherDBKey == ch {
				continue
			}
			otherDB := db.unaryDBs[otherDBKey]
			if otherDB.Channel.Index == udb.Config.Channel.Key && otherDB.Domain.GetBounds().OverlapsWith(tr) {
				return errors.New("Could not delete index channel with other channels depending on it")
			}
		}
	}

	return udb.Delete(ctx, tr)
}

func (db *DB) GCTombstone(ctx context.Context, ch ChannelKey, maxsizeRead uint32) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if !uok {
		return ChannelNotFound
	}

	return udb.Domain.CollectTombstone(ctx, maxsizeRead)
}

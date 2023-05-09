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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"strconv"
)

// openUnary opens the unary database for the given channel. If the database already exists,
// the only property that needs to be set on the channel is its key (as the existing database
// is assumed to live in a subdirectory named by the key). If the database does not
// exist, the channel must be fully populated and the database will be created.
func (db *DB) openUnary(ch Channel) error {
	fs, err := db.fs.Sub(strconv.Itoa(int(ch.Key)))
	if err != nil {
		return err
	}
	u, err := unary.Open(unary.Config{FS: fs, Channel: ch, Instrumentation: db.Instrumentation})
	if err != nil {
		return err
	}

	// In the case where we index the data using a separate index database, we
	// need to set the index on the unary database. Otherwise, we assume the database
	// is self-indexing.
	if u.Channel.Index != 0 && !u.Channel.IsIndex {
		idxDB, err := db.getUnary(u.Channel.Index)
		if errors.Is(err, ChannelNotFound) {
			err = db.openUnary(Channel{Key: u.Channel.Index})
			if err != nil {
				return err
			}
			idxDB, err = db.getUnary(u.Channel.Index)
			if err != nil {
				return err
			}
		}
		u.SetIndex((&idxDB).Index())
	}

	db.mu.Lock()
	db.dbs[ch.Key] = *u
	db.mu.Unlock()
	return nil
}

func (db *DB) getUnary(key core.ChannelKey) (unary.DB, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	u, ok := db.dbs[key]
	if !ok {
		return unary.DB{}, errors.Wrapf(ChannelNotFound, "channel: %s", key)
	}
	return u, nil
}

func (db *DB) unaryIsOpen(key core.ChannelKey) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, ok := db.dbs[key]
	return ok
}

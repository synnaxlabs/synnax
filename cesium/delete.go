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
	"strconv"
)

func (db *DB) DeleteChannel(ch ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if uok {
		if err := udb.TryClose(); err == nil {
			delete(db.unaryDBs, ch)
			return db.fs.Remove(strconv.Itoa(int(ch)))
		} else {
			return err
		}
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if db.GetDigestKey() == ch {
			return errors.New("[cesium] cannot delete update digest channel")
		}
		if err := vdb.TryClose(); err == nil {
			delete(db.virtualDBs, ch)
			return db.fs.Remove(strconv.Itoa(int(ch)))
		} else {
			return errors.New("[cesium] channel being written to")
		}
	}

	return ChannelNotFound
}

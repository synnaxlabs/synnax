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

func (db *DB) DeleteChannel(channel ChannelKey) error {
	// need to also check for virtual
	if db.unaryDBs[channel].Open_writers == 0 {
		return db.fs.Remove(strconv.Itoa(int(channel)))
	}
	return errors.New("Channel being written to")
}

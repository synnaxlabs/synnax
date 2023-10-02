// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

func (db *DB) DeleteChannel(channel_to_remove string) (bool, error) {
	if db.controlStates().Transfers == nil {
		// Someone is writing!
		return false, nil
	}

	if err := db.options.fs.Remove(channel_to_remove); err != nil {
		return false, err
	}

	return true, nil
}

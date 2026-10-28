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
	"context"

	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
)

// HistoryChange names a time range of a channel's stored data that a delete or a write
// before the channel's newest data changed.
type HistoryChange struct {
	// Key is the channel whose data changed.
	Key ChannelKey `json:"key"`
	// TimeRange holds every sample the change added or removed.
	TimeRange telem.TimeRange `json:"time_range"`
}

// OnHistoryChange registers handler to receive every HistoryChange. Handlers run inline
// while the DB holds its locks, so a handler must hand the change off: blocking on it
// stalls writers and deletes, and calling back into the DB deadlocks it. The returned
// function unregisters the handler.
func (db *DB) OnHistoryChange(
	handler func(context.Context, HistoryChange),
) observe.Disconnect {
	return db.historyChanges.OnChange(handler)
}

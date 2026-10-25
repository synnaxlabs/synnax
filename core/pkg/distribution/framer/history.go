// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
)

// HistoryChange names a time range of a channel's stored data that a delete or a write
// before the channel's newest data changed.
type HistoryChange struct {
	// Key is the channel whose stored data changed.
	Key channel.Key `json:"key" msgpack:"key"`
	// TimeRange holds every sample that the change added or removed.
	TimeRange telem.TimeRange `json:"time_range" msgpack:"time_range"`
}

// OnHistoryChange registers handler to receive every HistoryChange to channels stored
// on this Core. Changes to channels stored on other Cores never arrive. Handlers run
// inline under storage locks, so a handler must hand the change off. The returned
// function unregisters the handler.
func (s *Service) OnHistoryChange(
	handler func(context.Context, HistoryChange),
) observe.Disconnect {
	return s.cfg.TS.OnHistoryChange(func(ctx context.Context, c ts.HistoryChange) {
		handler(ctx, HistoryChange{Key: channel.Key(c.Key), TimeRange: c.TimeRange})
	})
}

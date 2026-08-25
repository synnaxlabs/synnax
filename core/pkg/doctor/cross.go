// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/cesium/inspect"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/set"
)

// checkCross reconciles the channels the Core stored against the channels Cesium
// holds. Only channels leased to the host node have storage here, so channels leased
// elsewhere are not judged.
func checkCross(s *state, ts *inspect.Report, host aspen.NodeKey) {
	stored := make(set.Set[channel.Key], len(ts.Channels))
	for _, ch := range ts.Channels {
		stored.Add(channel.Key(ch.Key))
	}
	for key := range s.channels {
		if key.Lease() != host || stored.Contains(key) {
			continue
		}
		s.note(CheckChannelDir, "channel has no Cesium directory", key.String())
	}
	for _, ch := range ts.Channels {
		key := channel.Key(ch.Key)
		c, found := s.channels[key]
		if !found {
			s.note(
				CheckChannelDir,
				"Cesium directory has no channel entry",
				key.String(),
			)
			continue
		}
		checkMeta(s, c, ch)
	}
}

// checkMeta reports disagreement between a channel entry and its Cesium metadata. A
// channel whose metadata could not be decoded is left to the time-series findings.
func checkMeta(s *state, c channel.Channel, report inspect.ChannelReport) {
	meta := report.Channel
	if meta.Key == 0 {
		return
	}
	key := c.Key().String()
	if meta.DataType != c.DataType {
		s.note(CheckChannelMeta, "data type disagrees with Cesium", key)
	}
	if meta.Virtual != c.Virtual {
		s.note(CheckChannelMeta, "virtual flag disagrees with Cesium", key)
	}
	if meta.IsIndex != c.IsIndex {
		s.note(CheckChannelMeta, "index flag disagrees with Cesium", key)
	}
	index := channel.NewKey(c.Leaseholder, c.LocalIndex)
	if c.LocalIndex != 0 && channel.Key(meta.Index) != index {
		s.note(CheckChannelMeta, "index channel disagrees with Cesium", key)
	}
}

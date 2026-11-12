// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9

import (
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

var yAxisKeys = []YAxisKey{YAxisKeyY1, YAxisKeyY2, YAxisKeyY3, YAxisKeyY4}

// ReconcileLines returns the plot's line set: one line per y-channel bound to a
// y-axis, plotted against the x1 channel. A line styles its channel across every
// range, so the set does not depend on the plot's ranges. An existing line with the
// same y-axis and channels keeps its styling and is re-keyed to the plot, so styling
// survives a plot getting a new key. Missing lines get default styling, with label
// and color nil so they resolve from the channel name and palette at render time.
func ReconcileLines(plot Key, channels Channels, existing []Line) []Line {
	type identity struct {
		yAxis              YAxisKey
		yChannel, xChannel channel.Key
	}
	byIdentity := make(map[identity]Line, len(existing))
	for _, l := range existing {
		id := identity{l.YAxis, l.YChannel, l.XChannel}
		if _, ok := byIdentity[id]; !ok {
			byIdentity[id] = l
		}
	}
	kept := set.New[identity]()
	lines := make([]Line, 0, len(existing))
	for _, yAxis := range yAxisKeys {
		for _, yChannel := range yAxisChannels(channels, yAxis) {
			id := identity{yAxis, yChannel, channels.X1}
			if kept.Contains(id) {
				continue
			}
			kept.Add(id)
			l, ok := byIdentity[id]
			if !ok {
				l = Line{}
				l.SetDefaults()
				l.YAxis, l.YChannel, l.XChannel = yAxis, yChannel, channels.X1
			}
			l.Key = v8.LineKey(plot, yAxis, yChannel, channels.X1)
			lines = append(lines, l)
		}
	}
	return lines
}

func yAxisChannels(c Channels, k YAxisKey) []channel.Key {
	switch k {
	case YAxisKeyY1:
		return c.Y1
	case YAxisKeyY2:
		return c.Y2
	case YAxisKeyY3:
		return c.Y3
	case YAxisKeyY4:
		return c.Y4
	}
	panic(errors.Newf("lineplot: unknown y-axis %q", k))
}

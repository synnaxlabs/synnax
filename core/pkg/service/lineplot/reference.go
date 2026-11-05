// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

const channelTable = string(resource.TypeChannel)

// channelsReference returns the config of the reference from a line plot to the
// channels on its axes. Deleting a channel removes it from every axis and drops its
// lines. Deleting the x channel plots the lines against time and keeps their styles.
func channelsReference() gorp.ReferenceConfig[Key, LinePlot] {
	return gorp.ReferenceConfig[Key, LinePlot]{
		Name:     "channels",
		Tables:   []string{channelTable},
		OnDelete: gorp.Detach,
		Targets: func(p *LinePlot) []gorp.Target {
			c := p.Channels
			keys := slices.Concat([]channel.Key{c.X1, c.X2}, c.Y1, c.Y2, c.Y3, c.Y4)
			targets := make([]gorp.Target, 0, len(keys))
			for _, k := range keys {
				if k != 0 {
					targets = append(targets, gorp.To(channelTable, k))
				}
			}
			return targets
		},
		Detach: func(p *LinePlot, t gorp.Target) {
			held := func(k channel.Key) bool { return gorp.To(channelTable, k) == t }
			c := &p.Channels
			if held(c.X1) {
				c.X1 = 0
				for i := range p.Lines {
					if held(p.Lines[i].XChannel) {
						p.Lines[i].XChannel = 0
					}
				}
			}
			if held(c.X2) {
				c.X2 = 0
			}
			for _, y := range []*[]channel.Key{&c.Y1, &c.Y2, &c.Y3, &c.Y4} {
				*y = slices.DeleteFunc(*y, held)
			}
			p.Lines = v8.ReconcileLines(p.Key, p.Channels, p.Lines)
		},
	}
}

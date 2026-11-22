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
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

const channelTable = string(resource.TypeChannel)

// channelsReference returns the reference from a line plot to the channels on its axes.
// Deleting a channel removes it from every axis and drops its lines. Deleting the x
// channel plots the lines against time and keeps their styles. A retarget moves each
// line with its style, and an axis keeps the first of two entries for one channel.
func channelsReference() *gorp.KeyReference[Key, LinePlot, channel.Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, LinePlot, channel.Key]{
		Name:     "channels",
		Table:    channelTable,
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Tag:      channel.ReadTag,
		Targets: func(p *LinePlot) []channel.Key {
			c := p.Channels
			keys := slices.Concat([]channel.Key{c.X1, c.X2}, c.Y1, c.Y2, c.Y3, c.Y4)
			return slices.DeleteFunc(keys, func(k channel.Key) bool { return k == 0 })
		},
		Detach: func(p *LinePlot, t channel.Key) {
			held := func(k channel.Key) bool { return k == t }
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
			p.Lines = v10.ReconcileLines(p.Key, p.Channels, p.Lines)
		},
		Retarget: func(p *LinePlot, to map[channel.Key]channel.Key) {
			move := func(k channel.Key) channel.Key {
				if t, ok := to[k]; ok {
					return t
				}
				return k
			}
			c := &p.Channels
			c.X1, c.X2 = move(c.X1), move(c.X2)
			for _, y := range []*[]channel.Key{&c.Y1, &c.Y2, &c.Y3, &c.Y4} {
				moved := make([]channel.Key, 0, len(*y))
				for _, k := range *y {
					if k = move(k); !slices.Contains(moved, k) {
						moved = append(moved, k)
					}
				}
				*y = moved
			}
			for i := range p.Lines {
				l := &p.Lines[i]
				l.YChannel, l.XChannel = move(l.YChannel), move(l.XChannel)
			}
			p.Lines = v10.ReconcileLines(p.Key, p.Channels, p.Lines)
		},
	})
}

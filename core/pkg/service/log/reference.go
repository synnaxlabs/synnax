// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// channelsReference returns the reference from a log to the channel of each of its
// entries. Deleting a channel drops its entry. A retarget moves each entry with its
// style, and the log keeps the first of two entries for one channel.
func channelsReference() *gorp.KeyReference[Key, Log, channel.Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Log, channel.Key]{
		Name:     "channels.channel",
		Table:    string(resource.TypeChannel),
		OnDelete: gorp.Detach,
		Kind:     gorp.Use,
		Tag:      channel.ReadTag,
		Targets: func(l *Log) []channel.Key {
			if len(l.Channels) == 0 {
				return nil
			}
			keys := make([]channel.Key, len(l.Channels))
			for i, e := range l.Channels {
				keys[i] = e.Channel
			}
			return keys
		},
		Detach: func(l *Log, t channel.Key) {
			l.Channels = slices.DeleteFunc(l.Channels, func(e ChannelEntry) bool {
				return e.Channel == t
			})
		},
		Retarget: func(l *Log, to map[channel.Key]channel.Key) {
			moved := make([]ChannelEntry, 0, len(l.Channels))
			for _, e := range l.Channels {
				if t, ok := to[e.Channel]; ok {
					e.Channel = t
				}
				if !slices.ContainsFunc(moved, func(m ChannelEntry) bool {
					return m.Channel == e.Channel
				}) {
					moved = append(moved, e)
				}
			}
			l.Channels = moved
		},
	})
}

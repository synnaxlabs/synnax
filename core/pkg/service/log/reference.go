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

const channelTable = string(resource.TypeChannel)

// channelsReference returns the reference from a log to the channels of its entries.
// Deleting a channel removes its entry.
func channelsReference() *gorp.KeyReference[Key, Log, channel.Key] {
	return gorp.NewKeyReference(gorp.KeyReferenceConfig[Key, Log, channel.Key]{
		Name:     "channels.channel",
		Table:    channelTable,
		OnDelete: gorp.Detach,
		Targets: func(l *Log) []channel.Key {
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
	})
}

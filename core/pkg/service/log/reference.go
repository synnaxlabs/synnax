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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

const channelTable = string(resource.TypeChannel)

// channelsReference returns the config of the reference from a log to the channels of
// its entries. Deleting a channel removes its entry.
func channelsReference() gorp.ReferenceConfig[Key, Log] {
	return gorp.ReferenceConfig[Key, Log]{
		Name:     "channels.channel",
		Tables:   []string{channelTable},
		OnDelete: gorp.Detach,
		Targets: func(l *Log) []gorp.Target {
			targets := make([]gorp.Target, len(l.Channels))
			for i, e := range l.Channels {
				targets[i] = gorp.To(channelTable, e.Channel)
			}
			return targets
		},
		Detach: func(l *Log, t gorp.Target) {
			l.Channels = slices.DeleteFunc(l.Channels, func(e ChannelEntry) bool {
				return gorp.To(channelTable, e.Channel) == t
			})
		},
	}
}

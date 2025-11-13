// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/set"
)

type ExtendedStateConfig struct {
	State         state.Config
	Reads, Writes set.Set[channel.Key]
}

func NewStateConfig(
	ctx context.Context,
	channelSvc channel.Readable,
	module arc.Module,
) (ExtendedStateConfig, error) {
	var (
		reads  = make(set.Set[channel.Key])
		writes = make(set.Set[channel.Key])
	)
	for _, n := range module.Nodes {
		isWrite := n.Type == "write"
		for rawChanKey := range n.Channels.Read {
			chanKey := channel.Key(rawChanKey)
			if isWrite {
				writes.Add(chanKey)
			} else {
				reads.Add(chanKey)
			}
		}
		for chanKey := range n.Channels.Write {
			writes.Add(channel.Key(chanKey))
		}
	}
	channels, err := retrieveChannels(ctx, channelSvc, slices.Concat(reads.Keys(), writes.Keys()))
	if err != nil {
		return ExtendedStateConfig{}, err
	}
	channelDigests := make([]state.ChannelDigest, 0, len(channels))
	for _, ch := range channels {
		channelDigests = append(channelDigests, state.ChannelDigest{
			Key:      uint32(ch.Key()),
			DataType: ch.DataType,
			Index:    uint32(ch.Index()),
		})
		if reads.Contains(ch.Key()) && ch.Index() != 0 {
			reads.Add(ch.Index())
		}
		if writes.Contains(ch.Key()) && ch.Index() != 0 {
			writes.Add(ch.Index())
		}
	}
	return ExtendedStateConfig{
		Reads:  reads,
		Writes: writes,
		State: state.Config{
			ChannelDigests: lo.Uniq(channelDigests),
			IR:             module.IR,
		},
	}, nil
}

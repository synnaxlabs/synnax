// Copyright 2026 Synnax Labs, Inc.
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
	Reads  set.Set[channel.Key]
	Writes set.Set[channel.Key]
	State  state.Config
}

func retrieveChannels(
	ctx context.Context,
	channelSvc *channel.Service,
	keys []channel.Key,
) ([]channel.Channel, error) {
	channels := make([]channel.Channel, 0, len(keys))
	if err := channelSvc.NewRetrieve().
		WhereKeys(keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	indexes := lo.FilterMap(channels, func(item channel.Channel, index int) (channel.Key, bool) {
		return item.Index(), !item.Virtual
	})
	indexChannels := make([]channel.Channel, 0, len(indexes))
	if err := channelSvc.NewRetrieve().
		WhereKeys(indexes...).
		Entries(&indexChannels).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return slices.Concat(channels, indexChannels), nil
}

func NewStateConfig(
	ctx context.Context,
	channelSvc *channel.Service,
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

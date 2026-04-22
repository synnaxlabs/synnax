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
	"github.com/synnaxlabs/arc/ir"
	stlchannel "github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/set"
)

type ExtendedStateConfig struct {
	Reads          set.Set[channel.Key]
	Writes         set.Set[channel.Key]
	ChannelDigests []stlchannel.Digest
	IR             ir.IR
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
	prog arc.Program,
) (ExtendedStateConfig, error) {
	var (
		reads  = make(set.Set[channel.Key])
		writes = make(set.Set[channel.Key])
	)
	for _, n := range prog.Nodes {
		for rawChanKey := range n.Channels.Read {
			reads.Add(channel.Key(rawChanKey))
		}
		for chanKey := range n.Channels.Write {
			writes.Add(channel.Key(chanKey))
		}
	}
	for key := range prog.Authorities.Channels {
		writes.Add(channel.Key(key))
	}
	channels, err := retrieveChannels(ctx, channelSvc, slices.Concat(reads.Slice(), writes.Slice()))
	if err != nil {
		return ExtendedStateConfig{}, err
	}
	channelDigests := make([]stlchannel.Digest, 0, len(channels))
	for _, ch := range channels {
		channelDigests = append(channelDigests, stlchannel.Digest{
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
		Reads:          reads,
		Writes:         writes,
		ChannelDigests: lo.Uniq(channelDigests),
		IR:             prog.IR,
	}, nil
}

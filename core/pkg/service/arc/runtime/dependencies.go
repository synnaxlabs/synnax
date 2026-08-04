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
	stlchannels "github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/set"
)

// Dependencies describes the channels a compiled Arc program reads from and writes to,
// along with the channel digests needed to execute it.
type Dependencies struct {
	// Reads is the set of channel keys the program reads from, including the index
	// channels of any non-virtual channels it reads.
	Reads set.Set[channel.Key]
	// Writes is the set of channel keys the program writes to, including the index
	// channels of any non-virtual channels it writes.
	Writes set.Set[channel.Key]
	// ChannelDigests holds the key, data type, and index for every channel the program
	// touches.
	ChannelDigests []stlchannels.Digest
}

func retrieveChannels(
	ctx context.Context,
	channelSvc *channel.Service,
	keys []channel.Key,
) ([]channel.Channel, error) {
	channels := make([]channel.Channel, 0, len(keys))
	if err := channelSvc.NewRetrieve().
		Where(channel.MatchKeys(keys...)).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	indexes := lo.FilterMap(channels, func(
		ch channel.Channel,
		_ int,
	) (channel.Key, bool) {
		return ch.Index(), !ch.Virtual
	})
	indexChannels := make([]channel.Channel, 0, len(indexes))
	if err := channelSvc.NewRetrieve().
		Where(channel.MatchKeys(indexes...)).
		Entries(&indexChannels).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return slices.Concat(channels, indexChannels), nil
}

// NewDependencies derives the read/write channel sets and digests for a compiled
// program. Non-virtual channels pull in their index channels so the executor can align
// samples. It returns an error if any referenced channel cannot be retrieved.
func NewDependencies(
	ctx context.Context,
	channelSvc *channel.Service,
	prog arc.Program,
) (Dependencies, error) {
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
	channels, err := retrieveChannels(
		ctx,
		channelSvc,
		slices.Concat(reads.Slice(), writes.Slice()),
	)
	if err != nil {
		return Dependencies{}, err
	}
	channelDigests := make([]stlchannels.Digest, 0, len(channels))
	for _, ch := range channels {
		channelDigests = append(channelDigests, stlchannels.Digest{
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
	return Dependencies{
		Reads:          reads,
		Writes:         writes,
		ChannelDigests: lo.Uniq(channelDigests),
	}, nil
}

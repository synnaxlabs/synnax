// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/testutil"
)

// ChannelStore is an in-memory channel.Retriever shared across the nodes of a mock
// cluster. It lets distribution-layer tests create channels (through the allocator) and
// resolve their metadata without depending on the service layer, which is where the real
// channel table and retrieval live.
type ChannelStore struct {
	mu sync.RWMutex
	m  map[channel.Key]channel.Channel
}

func newChannelStore() *ChannelStore {
	return &ChannelStore{m: make(map[channel.Key]channel.Channel)}
}

func (s *ChannelStore) add(channels ...channel.Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range channels {
		s.m[ch.Key()] = ch
	}
}

// RetrieveByKeys implements channel.Retriever. It returns query.ErrNotFound if any
// requested key has no matching channel, mirroring the gorp retrieve semantics of the
// real service-layer retriever the framer depends on.
func (s *ChannelStore) RetrieveByKeys(_ context.Context, keys ...channel.Key) ([]channel.Channel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]channel.Channel, 0, len(keys))
	for _, k := range keys {
		ch, ok := s.m[k]
		if !ok {
			return nil, errors.Wrapf(query.ErrNotFound, "Channel %v not found", k)
		}
		out = append(out, ch)
	}
	return out, nil
}

// ContainsKeys implements channel.Retriever.
func (s *ChannelStore) ContainsKeys(_ context.Context, keys ...channel.Key) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range keys {
		if _, ok := s.m[k]; !ok {
			return false, nil
		}
	}
	return true, nil
}

var _ channel.Retriever = (*ChannelStore)(nil)

// CreateChannel allocates a single channel — assigning its local key and creating its
// storage on the leaseholder — and records its metadata in the cluster's shared channel
// store so the framer can resolve it. The channel is updated in place with its key.
func (n Node) CreateChannel(ctx context.Context, ch *channel.Channel) error {
	out, err := n.Channel.Create(ctx, []channel.Channel{*ch})
	if err != nil {
		return err
	}
	*ch = out[0]
	n.channels.add(out...)
	return nil
}

// CreateChannels allocates a batch of channels and records their metadata. The slice is
// updated in place with the assigned keys.
func (n Node) CreateChannels(ctx context.Context, channels *[]channel.Channel) error {
	out, err := n.Channel.Create(ctx, *channels)
	if err != nil {
		return err
	}
	*channels = out
	n.channels.add(out...)
	return nil
}

// RetrieveChannels returns the channels matching the provided keys from the shared store.
func (n Node) RetrieveChannels(ctx context.Context, keys ...channel.Key) []channel.Channel {
	return testutil.MustSucceed(n.channels.RetrieveByKeys(ctx, keys...))
}

// RetrieveChannelsInto populates chs with the channels matching the provided keys. It
// always returns a nil error, mirroring the shape of a retrieve query so tests can keep
// asserting Succeed().
func (n Node) RetrieveChannelsInto(ctx context.Context, chs *[]channel.Channel, keys ...channel.Key) error {
	out, err := n.channels.RetrieveByKeys(ctx, keys...)
	*chs = out
	return err
}

// OpenWriter opens a distribution-layer writer against the node, resolving the channel
// metadata for cfg.Keys through the node's bound channel retriever and attaching it to
// the config. The distribution writer requires resolved channels on its config; in
// production the service layer supplies them. Resolving through the bound retriever lets
// this work whether the test created channels through the mock store (pure
// distribution-layer tests) or a real service-layer channel service.
func (n Node) OpenWriter(ctx context.Context, cfg writer.Config) (*writer.Writer, error) {
	channels, err := n.ChannelRetriever.RetrieveByKeys(ctx, cfg.Keys...)
	if err != nil {
		return nil, err
	}
	cfg.Channels = channels
	return n.Framer.OpenWriter(ctx, cfg)
}

// NewStreamWriter opens a distribution-layer stream writer against the node, resolving
// channel metadata for cfg.Keys exactly as OpenWriter does.
func (n Node) NewStreamWriter(ctx context.Context, cfg writer.Config) (writer.StreamWriter, error) {
	channels, err := n.ChannelRetriever.RetrieveByKeys(ctx, cfg.Keys...)
	if err != nil {
		return nil, err
	}
	cfg.Channels = channels
	return n.Framer.NewStreamWriter(ctx, cfg)
}

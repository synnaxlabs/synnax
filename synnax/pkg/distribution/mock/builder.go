// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/aspen"
	aspentransmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type Builder struct {
	cfg         distribution.Config
	storage     *mock.Builder
	Nodes       map[distribution.NodeKey]*distribution.Layer
	writerNet   *tmock.FramerWriterNetwork
	iterNet     *tmock.FramerIteratorNetwork
	channelNet  *tmock.ChannelNetwork
	relayNet    *tmock.FramerRelayNetwork
	deleteNet   *tmock.FramerDeleterNetwork
	aspenNet    *aspentransmock.Network
	addrFactory *address.Factory
}

func NewBuilder(cfgs ...distribution.Config) *Builder {
	cfg := MustSucceed(config.New(distribution.DefaultConfig, cfgs...))
	return &Builder{
		cfg:         cfg,
		storage:     mock.NewBuilder(),
		writerNet:   tmock.NewWriterNetwork(),
		iterNet:     tmock.NewIteratorNetwork(),
		channelNet:  tmock.NewChannelNetwork(),
		relayNet:    tmock.NewRelayNetwork(),
		deleteNet:   tmock.NewDeleterNetwork(),
		aspenNet:    aspentransmock.NewNetwork(),
		addrFactory: address.NewLocalFactory(0),
		Nodes:       make(map[distribution.NodeKey]*distribution.Layer),
	}
}

func (b *Builder) New(ctx context.Context) *distribution.Layer {
	peers := b.addrFactory.Generated()
	addr := b.addrFactory.Next()
	dist := MustSucceed(distribution.Open(ctx, distribution.Config{
		Storage: b.storage.New(ctx),
		FrameTransport: mockFramerTransport{
			iter:    b.iterNet.New(addr, 1),
			writer:  b.writerNet.New(addr, 1),
			relay:   b.relayNet.New(addr, 1),
			deleter: b.deleteNet.New(addr),
		},
		ChannelTransport: b.channelNet.New(addr),
		AspenTransport:   b.aspenNet.NewTransport(),
		AdvertiseAddress: b.cfg.AdvertiseAddress,
		PeerAddresses:    peers,
		AspenOptions: []aspen.Option{
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		},
	}, b.cfg))
	b.Nodes[dist.Cluster.HostKey()] = dist
	return dist
}

func (b *Builder) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, node := range b.Nodes {
		c.Exec(node.Close)
	}
	return c.Error()
}

func (b *Builder) Cleanup() error {
	return b.storage.Cleanup()
}

type mockFramerTransport struct {
	iter    iterator.Transport
	writer  writer.Transport
	relay   relay.Transport
	deleter deleter.Transport
}

var _ framer.Transport = (*mockFramerTransport)(nil)

func (m mockFramerTransport) Iterator() iterator.Transport {
	return m.iter
}

func (m mockFramerTransport) Writer() writer.Transport {
	return m.writer
}

func (m mockFramerTransport) Relay() relay.Transport {
	return m.relay
}

func (m mockFramerTransport) Deleter() deleter.Transport {
	return m.deleter
}

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

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	aspentransmock "github.com/synnaxlabs/aspen/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tmock "github.com/synnaxlabs/synnax/pkg/distribution/transport/mock"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/storage/mock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

type Node struct {
	*distribution.Layer
	Storage *storage.Layer
}

type Cluster struct {
	cfg         distribution.Config
	storage     *mock.Cluster
	Nodes       map[cluster.NodeKey]Node
	writerNet   *tmock.FramerWriterNetwork
	iterNet     *tmock.FramerIteratorNetwork
	channelNet  *tmock.ChannelNetwork
	relayNet    *tmock.FramerRelayNetwork
	deleteNet   *tmock.FramerDeleterNetwork
	aspenNet    *aspentransmock.Network
	addrFactory *address.Factory
}

func ProvisionCluster(ctx context.Context, n int, cfgs ...distribution.Config) *Cluster {
	b := NewCluster(cfgs...)
	for range n {
		b.Provision(ctx)
	}
	return b
}

func NewCluster(cfgs ...distribution.Config) *Cluster {
	cfg, _ := config.New(distribution.Config{}, cfgs...)
	return &Cluster{
		cfg:         cfg,
		storage:     mock.NewCluster(),
		writerNet:   tmock.NewWriterNetwork(),
		iterNet:     tmock.NewIteratorNetwork(),
		channelNet:  tmock.NewChannelNetwork(),
		relayNet:    tmock.NewRelayNetwork(),
		deleteNet:   tmock.NewDeleterNetwork(),
		aspenNet:    aspentransmock.NewNetwork(),
		addrFactory: address.NewLocalFactory(0),
		Nodes:       make(map[cluster.NodeKey]Node),
	}
}

func (b *Cluster) Provision(
	ctx context.Context,
	cfgs ...distribution.Config,
) Node {
	var (
		peers             = b.addrFactory.Generated()
		addr              = b.addrFactory.Next()
		storageLayer      = b.storage.Provision(ctx)
		distributionLayer = testutil.MustSucceed(distribution.Open(ctx, append([]distribution.Config{{
			Storage: storageLayer,
			FrameTransport: mockFramerTransport{
				iter:    b.iterNet.New(addr, 1),
				writer:  b.writerNet.New(addr, 1),
				relay:   b.relayNet.New(addr, 1),
				deleter: b.deleteNet.New(addr),
			},
			ChannelTransport: b.channelNet.New(addr),
			AspenTransport:   b.aspenNet.NewTransport(),
			AdvertiseAddress: addr,
			PeerAddresses:    peers,
			AspenOptions: []aspen.Option{
				aspen.WithPropagationConfig(aspen.FastPropagationConfig),
			},
			GorpCodec:            &binary.JSONCodec{},
			EnableServiceSignals: config.False(),
		}, b.cfg}, cfgs...)...))
	)
	node := Node{Layer: distributionLayer, Storage: storageLayer}
	b.Nodes[distributionLayer.Cluster.HostKey()] = node
	b.WaitForTopologyToStabilize()
	return node
}

// WaitForTopologyToStabilize waits for all nodes in the cluster to be aware of each
// other.
func (b *Cluster) WaitForTopologyToStabilize() {
	for _, node := range b.Nodes {
		gomega.Eventually(func() int {
			return len(node.Cluster.Nodes())
		}).Should(gomega.Equal(len(b.Nodes)))
	}
}

func (b *Cluster) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, node := range b.Nodes {
		c.Exec(node.Close)
	}
	c.Exec(b.storage.Close)
	return c.Error()
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

type StaticHostProvider struct {
	Node cluster.Node
}

var _ cluster.HostProvider = StaticHostProvider{}

func StaticHostKeyProvider(key cluster.NodeKey) StaticHostProvider {
	return StaticHostProvider{Node: cluster.Node{Key: key}}
}

func (s StaticHostProvider) Host() cluster.Node { return s.Node }

func (s StaticHostProvider) HostKey() cluster.NodeKey { return s.Node.Key }

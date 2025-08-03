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
	. "github.com/synnaxlabs/x/testutil"
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
	c := NewCluster(cfgs...)
	for range n {
		c.Provision(ctx)
	}
	return c
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

func (c *Cluster) Provision(
	ctx context.Context,
	cfgs ...distribution.Config,
) Node {
	var (
		peers             = c.addrFactory.Generated()
		addr              = c.addrFactory.Next()
		storageLayer      = c.storage.Provision(ctx)
		distributionLayer = MustSucceed(distribution.Open(ctx, append([]distribution.Config{{
			Storage: storageLayer,
			FrameTransport: mockFramerTransport{
				iter:    c.iterNet.New(addr, 1),
				writer:  c.writerNet.New(addr, 1),
				relay:   c.relayNet.New(addr, 1),
				deleter: c.deleteNet.New(addr),
			},
			ChannelTransport: c.channelNet.New(addr),
			AspenTransport:   c.aspenNet.NewTransport(),
			AdvertiseAddress: addr,
			PeerAddresses:    peers,
			AspenOptions: []aspen.Option{
				aspen.WithPropagationConfig(aspen.FastPropagationConfig),
			},
			GorpCodec:            binary.JSONCodec,
			EnableChannelSignals: config.False(),
		}, c.cfg}, cfgs...)...))
	)
	node := Node{Layer: distributionLayer, Storage: storageLayer}
	c.Nodes[distributionLayer.Cluster.HostKey()] = node
	c.WaitForTopologyToStabilize()
	return node
}

// WaitForTopologyToStabilize waits for all nodes in the cluster to be aware of each
// other.
func (c *Cluster) WaitForTopologyToStabilize() {
	for _, node := range c.Nodes {
		gomega.Eventually(func() int {
			return len(node.Cluster.Nodes())
		}).Should(gomega.Equal(len(c.Nodes)))
	}
}

func (c *Cluster) Close() error {
	catcher := errors.NewCatcher(errors.WithAggregation())
	for _, node := range c.Nodes {
		catcher.Exec(node.Close)
	}
	catcher.Exec(c.storage.Close)
	return catcher.Error()
}

type mockFramerTransport struct {
	iter    iterator.Transport
	writer  writer.Transport
	relay   relay.Transport
	deleter deleter.Transport
}

var _ framer.Transport = (*mockFramerTransport)(nil)

func (mft mockFramerTransport) Iterator() iterator.Transport { return mft.iter }

func (mft mockFramerTransport) Writer() writer.Transport { return mft.writer }

func (mft mockFramerTransport) Relay() relay.Transport { return mft.relay }

func (mft mockFramerTransport) Deleter() deleter.Transport { return mft.deleter }

type StaticHostProvider struct{ Node cluster.Node }

var _ cluster.HostProvider = StaticHostProvider{}

func NewStaticHostProvider(key cluster.NodeKey) StaticHostProvider {
	return StaticHostProvider{Node: cluster.Node{Key: key}}
}

func (shp StaticHostProvider) Host() cluster.Node { return shp.Node }

func (shp StaticHostProvider) HostKey() cluster.NodeKey { return shp.Node.Key }

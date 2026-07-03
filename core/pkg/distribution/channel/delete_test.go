// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete", Ordered, func() {
	var n mock.Node

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = mock.NewNode(ctx)
	})

	It("Should delete the storage channel for a gateway key", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "to-delete", DataType: telem.TimeStampT, IsIndex: true},
		}))
		key := out[0].Key()
		Expect(n.Channel.Delete(ctx, channel.Keys{key})).To(Succeed())
		Expect(n.Storage.TS.RetrieveChannel(ctx, key.StorageKey())).Error().To(
			MatchError(query.ErrNotFound),
		)
	})
	It("Should skip free-virtual channels without error", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "free-delete", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
		}))
		Expect(n.Channel.Delete(ctx, channel.Keys{out[0].Key()})).To(Succeed())
	})
	It("Should return an error when a key's leaseholder is not in the cluster", func(ctx SpecContext) {
		Expect(n.Channel.Delete(ctx, channel.Keys{channel.NewKey(node.Key(99), 1)})).To(
			MatchError(query.ErrNotFound),
		)
	})

	Context("Multi Node", Ordered, func() {
		var (
			gateway mock.Node
			peer    mock.Node
		)
		BeforeAll(func(ctx SpecContext) {
			cluster := mock.NewCluster(ctx, 2)
			gateway = cluster.Nodes[node.KeyBootstrapper]
			peer = cluster.Nodes[node.Key(2)]
		})

		It("Should route deletion to the leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-delete", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			key := out[0].Key()
			MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
			Expect(gateway.Channel.Delete(ctx, channel.Keys{key})).To(Succeed())
			Expect(peer.Storage.TS.RetrieveChannel(ctx, key.StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
		It("Should delete both gateway and peer channels", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "gateway-delete", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "peer-delete", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			key := out[0].Key()
			Expect(gateway.Channel.Delete(ctx, channel.Keys{key})).To(Succeed())
			Expect(peer.Channel.Delete(ctx, channel.Keys{key})).To(Succeed())
		})
		It("Should route a mixed batch to each key's leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "mixed-gateway-delete", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "mixed-peer-delete", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			gatewayKey := out[0].Key()
			peerKey := out[1].Key()
			Expect(gateway.Channel.Delete(ctx, channel.Keys{gatewayKey, peerKey})).To(Succeed())
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, gatewayKey.StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
			Expect(peer.Storage.TS.RetrieveChannel(ctx, peerKey.StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
	})
})

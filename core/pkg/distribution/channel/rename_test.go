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

var _ = Describe("Rename", Ordered, func() {
	var n mock.Node

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = mock.NewNode(ctx)
	})

	It("Should rename the storage channel for a gateway key", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "old-name", DataType: telem.TimeStampT, IsIndex: true},
		}))
		key := out[0].Key()
		Expect(n.Channel.Rename(ctx, map[channel.Key]string{key: "new-name"})).To(Succeed())
		stored := MustSucceed(n.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
		Expect(stored.Name).To(Equal("new-name"))
	})
	It("Should skip free-virtual channels without error", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "free-rename", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
		}))
		Expect(n.Channel.Rename(
			ctx, map[channel.Key]string{out[0].Key(): "ignored"},
		)).To(Succeed())
	})
	It("Should return an error when a key's leaseholder is not in the cluster", func(ctx SpecContext) {
		Expect(n.Channel.Rename(
			ctx, map[channel.Key]string{channel.NewKey(node.Key(99), 1): "unresolvable"},
		)).To(MatchError(query.ErrNotFound))
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

		It("Should route the rename to the leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-old", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			key := out[0].Key()
			Expect(gateway.Channel.Rename(
				ctx, map[channel.Key]string{key: "remote-new"},
			)).To(Succeed())
			stored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
			Expect(stored.Name).To(Equal("remote-new"))
		})
		It("Should rename both gateway and peer channels", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "gateway-old", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "peer-old", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			gatewayKey := out[0].Key()
			peerKey := out[1].Key()
			Expect(gateway.Channel.Rename(
				ctx, map[channel.Key]string{gatewayKey: "gateway-new"},
			)).To(Succeed())
			Expect(peer.Channel.Rename(
				ctx, map[channel.Key]string{peerKey: "peer-new"},
			)).To(Succeed())
			stored := MustSucceed(gateway.Storage.TS.RetrieveChannel(ctx, gatewayKey.StorageKey()))
			Expect(stored.Name).To(Equal("gateway-new"))
			stored = MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, peerKey.StorageKey()))
			Expect(stored.Name).To(Equal("peer-new"))
		})
		It("Should route a mixed-batch rename to each key's leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "mb-gateway-old", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "mb-peer-old", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			gatewayKey := out[0].Key()
			peerKey := out[1].Key()
			Expect(gateway.Channel.Rename(
				ctx,
				map[channel.Key]string{
					gatewayKey: "mb-gateway-new",
					peerKey:    "mb-peer-new",
				},
			)).To(Succeed())
			Expect(MustSucceed(gateway.Storage.TS.RetrieveChannel(ctx, gatewayKey.StorageKey())).Name).
				To(Equal("mb-gateway-new"))
			Expect(MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, peerKey.StorageKey())).Name).
				To(Equal("mb-peer-new"))
		})
	})
})

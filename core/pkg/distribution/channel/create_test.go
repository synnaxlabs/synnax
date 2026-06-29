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

var _ = Describe("Create", Ordered, func() {
	var (
		n    mock.Node
		host node.Key
	)
	BeforeAll(func(ctx SpecContext) {
		n = mock.NewNode(ctx)
		host = n.Cluster.HostKey()
	})

	It("Should assign a local key and create storage for a gateway channel", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "gateway", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: n.Cluster.HostKey()},
		}))
		Expect(out[0].LocalKey).ToNot(BeZero())
		Expect(out[0].Key().Lease()).To(Equal(host))
		stored := MustSucceed(n.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
		Expect(stored.Name).To(Equal("gateway"))
	})
	It("Should default an unspecified leaseholder to the host node", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "defaulted", DataType: telem.TimeStampT, IsIndex: true},
		}))
		Expect(out[0].Leaseholder).To(Equal(host))
	})
	It("Should assign sequential local keys across a batch", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "batch-a", DataType: telem.TimeStampT, IsIndex: true},
			{Name: "batch-b", DataType: telem.TimeStampT, IsIndex: true},
			{Name: "batch-c", DataType: telem.TimeStampT, IsIndex: true},
		}))
		Expect(out[1].LocalKey).To(Equal(out[0].LocalKey + 1))
		Expect(out[2].LocalKey).To(Equal(out[1].LocalKey + 1))
	})
	It("Should set the local index of an index channel to its own local key", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "index", DataType: telem.TimeStampT, IsIndex: true},
		}))
		Expect(out[0].LocalIndex).To(Equal(out[0].LocalKey))
	})
	It("Should allocate a free virtual channel without creating storage", func(ctx SpecContext) {
		out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
			{Name: "free", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
		}))
		Expect(out[0].LocalKey).ToNot(BeZero())
		Expect(out[0].Key().Lease()).To(Equal(node.KeyFree))
		Expect(n.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
			MatchError(query.ErrNotFound),
		)
	})
	It("Should return an error when the leaseholder is not in the cluster", func(ctx SpecContext) {
		Expect(n.Channel.Create(ctx, []channel.Channel{
			{Name: "unresolvable", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: node.Key(99)},
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should surface an error when storage creation fails", func(ctx SpecContext) {
		Expect(n.Channel.Create(ctx, []channel.Channel{
			{Name: "no-index", DataType: telem.Float32T},
		})).Error().To(MatchError(ContainSubstring("non-indexed channel must have an index")))
	})

	Context("Multi Node", Ordered, func() {
		var (
			gateway mock.Node
			peer    mock.Node
		)
		BeforeAll(func(ctx SpecContext) {
			cluster := mock.NewCluster(ctx, 0)
			gateway = cluster.Provision(ctx)
			peer = cluster.Provision(ctx)
		})

		It("Should route allocation to the leaseholder and create storage there", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			Expect(out[0].LocalKey).ToNot(BeZero())
			Expect(out[0].Key().Lease()).To(Equal(peer.Cluster.HostKey()))
			stored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
			Expect(stored.Name).To(Equal("remote"))
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
		It("Should set the local index to its own key for a remote index channel", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-index", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			Expect(out[0].LocalIndex).To(Equal(out[0].LocalKey))
			stored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
			Expect(stored.IsIndex).To(BeTrue())
		})
		It("Should create a virtual channel on the leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-virtual", DataType: telem.JSONT, Leaseholder: peer.Cluster.HostKey(), Virtual: true},
			}))
			Expect(out[0].Key().Lease()).To(Equal(peer.Cluster.HostKey()))
			stored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
			Expect(stored.Virtual).To(BeTrue())
			Expect(stored.DataType).To(Equal(telem.JSONT))
		})
		It("Should allocate a free virtual channel through a non-bootstrapper node", func(ctx SpecContext) {
			out := MustSucceed(peer.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-free", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
			}))
			Expect(out[0].LocalKey).ToNot(BeZero())
			Expect(out[0].Key().Lease()).To(Equal(node.KeyFree))
			Expect(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
		It("Should route a mixed batch to each channel's leaseholder", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "mixed-gateway", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "mixed-peer", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			Expect(out[0].Key().Lease()).To(Equal(gateway.Cluster.HostKey()))
			Expect(out[1].Key().Lease()).To(Equal(peer.Cluster.HostKey()))
			gatewayStored := MustSucceed(gateway.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
			Expect(gatewayStored.Name).To(Equal("mixed-gateway"))
			peerStored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[1].Key().StorageKey()))
			Expect(peerStored.Name).To(Equal("mixed-peer"))
			Expect(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, out[1].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
		It("Should route a mixed batch of local, remote, and free channels from a peer", func(ctx SpecContext) {
			out := MustSucceed(peer.Channel.Create(ctx, []channel.Channel{
				{Name: "mb-local", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
				{Name: "mb-remote", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: gateway.Cluster.HostKey()},
				{Name: "mb-free", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
			}))
			Expect(out[0].Key().Lease()).To(Equal(peer.Cluster.HostKey()))
			Expect(MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Name).
				To(Equal("mb-local"))
			Expect(out[1].Key().Lease()).To(Equal(gateway.Cluster.HostKey()))
			Expect(MustSucceed(gateway.Storage.TS.RetrieveChannel(ctx, out[1].Key().StorageKey())).Name).
				To(Equal("mb-remote"))
			Expect(out[2].Key().Lease()).To(Equal(node.KeyFree))
			Expect(out[2].LocalKey).ToNot(BeZero())
			Expect(peer.Storage.TS.RetrieveChannel(ctx, out[2].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, out[2].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
		It("Should return an error when the leaseholder is not in the cluster", func(ctx SpecContext) {
			Expect(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-unresolvable", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: node.Key(99)},
			})).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should surface an error when storage creation fails on the leaseholder", func(ctx SpecContext) {
			Expect(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote-no-index", DataType: telem.Float32T, Leaseholder: peer.Cluster.HostKey()},
			})).Error().To(MatchError(ContainSubstring("non-indexed channel must have an index")))
		})
	})
})

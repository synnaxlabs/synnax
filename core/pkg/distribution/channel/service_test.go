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

var _ = Describe("Service", func() {
	Describe("NewService", func() {
		It("Should return an error when required configuration is missing", func(ctx SpecContext) {
			Expect(channel.NewService(ctx, channel.ServiceConfig{})).Error().To(
				MatchError(ContainSubstring("host_resolver")),
			)
		})
	})

	Describe("Single Node", Ordered, func() {
		var (
			builder *mock.Cluster
			n       mock.Node
			host    node.Key
		)
		BeforeAll(func(ctx SpecContext) {
			builder = mock.ProvisionCluster(ctx, 1)
			n = builder.Nodes[node.KeyBootstrapper]
			host = n.Cluster.HostKey()
		})
		AfterAll(func() { Expect(builder.Close()).To(Succeed()) })

		Describe("Create", func() {
			It("Should assign a local key and create storage for a gateway channel", func(ctx SpecContext) {
				out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
					{Name: "gateway", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: host},
				}))
				Expect(out[0].LocalKey).ToNot(BeZero())
				Expect(out[0].Key().Leaseholder()).To(Equal(host))
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
				Expect(out[0].Key().Leaseholder()).To(Equal(node.KeyFree))
				Expect(n.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
					MatchError(query.ErrNotFound),
				)
			})
		})

		Describe("Delete", func() {
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
		})

		Describe("Rename", func() {
			It("Should rename the storage channel for a gateway key", func(ctx SpecContext) {
				out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
					{Name: "old-name", DataType: telem.TimeStampT, IsIndex: true},
				}))
				key := out[0].Key()
				Expect(n.Channel.Rename(ctx, channel.Keys{key}, []string{"new-name"})).To(Succeed())
				stored := MustSucceed(n.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
				Expect(stored.Name).To(Equal("new-name"))
			})
			It("Should skip free-virtual channels without error", func(ctx SpecContext) {
				out := MustSucceed(n.Channel.Create(ctx, []channel.Channel{
					{Name: "free-rename", DataType: telem.Float32T, Leaseholder: node.KeyFree, Virtual: true},
				}))
				Expect(n.Channel.Rename(ctx, channel.Keys{out[0].Key()}, []string{"ignored"})).To(Succeed())
			})
		})
	})

	Describe("Multi Node", Ordered, func() {
		var (
			builder *mock.Cluster
			gateway mock.Node
			peer    mock.Node
		)
		BeforeAll(func(ctx SpecContext) {
			builder = mock.ProvisionCluster(ctx, 2)
			gateway = builder.Nodes[node.KeyBootstrapper]
			peer = builder.Nodes[node.Key(2)]
		})
		AfterAll(func() { Expect(builder.Close()).To(Succeed()) })

		It("Should route allocation to the leaseholder and create storage there", func(ctx SpecContext) {
			out := MustSucceed(gateway.Channel.Create(ctx, []channel.Channel{
				{Name: "remote", DataType: telem.TimeStampT, IsIndex: true, Leaseholder: peer.Cluster.HostKey()},
			}))
			Expect(out[0].LocalKey).ToNot(BeZero())
			Expect(out[0].Key().Leaseholder()).To(Equal(peer.Cluster.HostKey()))
			stored := MustSucceed(peer.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey()))
			Expect(stored.Name).To(Equal("remote"))
			Expect(gateway.Storage.TS.RetrieveChannel(ctx, out[0].Key().StorageKey())).Error().To(
				MatchError(query.ErrNotFound),
			)
		})
	})
})

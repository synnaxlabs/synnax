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
	var (
		builder *mock.Cluster
		n       mock.Node
	)
	BeforeAll(func(ctx SpecContext) {
		builder = mock.MustOpenCluster(ctx, 1)
		n = builder.Nodes[node.KeyBootstrapper]
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

	Context("Multi Node", Ordered, func() {
		var (
			multiBuilder *mock.Cluster
			gateway      mock.Node
			peer         mock.Node
		)
		BeforeAll(func(ctx SpecContext) {
			multiBuilder = mock.MustOpenCluster(ctx, 2)
			gateway = multiBuilder.Nodes[node.KeyBootstrapper]
			peer = multiBuilder.Nodes[node.Key(2)]
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
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Multi Node", Ordered, func() {
	var (
		node1, node2 mock.Node
		svc1, svc2   *channel.Service
		writer1      channel.Writer
		writer2      channel.Writer
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		cluster := mock.NewCluster(ctx, 0)
		node1 = cluster.Provision(ctx)
		node2 = cluster.Provision(ctx)
		svc1, _ = openService(ctx, node1)
		svc2, _ = openService(ctx, node2)
		writer1 = svc1.NewWriter(nil)
		writer2 = svc2.NewWriter(nil)
	})

	It("Should make a channel created on node 1 retrievable from node 2", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:        "mn_gossiped",
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Leaseholder: node1.Cluster.HostKey(),
		}
		Expect(writer1.Create(ctx, &ch)).To(Succeed())
		Expect(ch.LocalKey).ToNot(BeZero())
		Eventually(func(g Gomega) {
			var res channel.Channel
			g.Expect(svc2.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
			g.Expect(res.Key()).To(Equal(ch.Key()))
			g.Expect(res.Name).To(Equal("mn_gossiped"))
			g.Expect(res.DataType).To(Equal(telem.TimeStampT))
			g.Expect(res.Leaseholder).To(Equal(node1.Cluster.HostKey()))
			g.Expect(res.IsIndex).To(BeTrue())
		}).Should(Succeed())
	})

	It("Should allocate a remote-leased channel on its leaseholder", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:        "mn_remote",
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Leaseholder: node2.Cluster.HostKey(),
		}
		Expect(writer1.Create(ctx, &ch)).To(Succeed())
		Expect(ch.LocalKey).ToNot(BeZero())
		Expect(ch.Leaseholder).To(Equal(node2.Cluster.HostKey()))
		Expect(ch.LocalIndex).To(Equal(ch.LocalKey))
		stored := MustSucceed(node2.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()))
		Expect(stored.Name).To(Equal("mn_remote"))
		Expect(node1.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey())).Error().To(
			MatchError(query.ErrNotFound),
		)
		for _, svc := range []*channel.Service{svc1, svc2} {
			Eventually(func(g Gomega) {
				var res channel.Channel
				g.Expect(svc.NewRetrieve().
					Where(channel.MatchKeys(ch.Key())).
					Entry(&res).
					Exec(ctx, nil)).To(Succeed())
				g.Expect(res.Name).To(Equal("mn_remote"))
				g.Expect(res.Leaseholder).To(Equal(node2.Cluster.HostKey()))
				g.Expect(res.LocalKey).To(Equal(ch.LocalKey))
			}).Should(Succeed())
		}
	})

	It("Should propagate a rename performed on the leaseholder to the other node", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:        "mn_rename_old",
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Leaseholder: node2.Cluster.HostKey(),
		}
		Expect(writer1.Create(ctx, &ch)).To(Succeed())
		Eventually(func(g Gomega) {
			var res channel.Channel
			g.Expect(svc2.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
		}).Should(Succeed())
		Expect(writer2.Rename(ctx, ch.Key(), "mn_rename_new", false)).To(Succeed())
		Expect(MustSucceed(
			node2.Storage.TS.RetrieveChannel(ctx, ch.Key().StorageKey()),
		).Name).To(Equal("mn_rename_new"))
		Eventually(func(g Gomega) {
			var res channel.Channel
			g.Expect(svc1.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
			g.Expect(res.Name).To(Equal("mn_rename_new"))
		}).Should(Succeed())
	})

	It("Should propagate a delete performed on the leaseholder to the other node", func(ctx SpecContext) {
		ch := channel.Channel{
			Name:        "mn_deleted",
			DataType:    telem.TimeStampT,
			IsIndex:     true,
			Leaseholder: node2.Cluster.HostKey(),
		}
		Expect(writer1.Create(ctx, &ch)).To(Succeed())
		key := ch.Key()
		MustSucceed(node2.Storage.TS.RetrieveChannel(ctx, key.StorageKey()))
		Eventually(func(g Gomega) {
			var res channel.Channel
			g.Expect(svc1.NewRetrieve().
				Where(channel.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, nil)).To(Succeed())
		}).Should(Succeed())
		Expect(writer2.Delete(ctx, key, false)).To(Succeed())
		Expect(node2.Storage.TS.RetrieveChannel(ctx, key.StorageKey())).Error().To(
			MatchError(query.ErrNotFound),
		)
		Eventually(func() error {
			var res channel.Channel
			return svc1.NewRetrieve().
				Where(channel.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, nil)
		}).Should(MatchError(query.ErrNotFound))
	})
})

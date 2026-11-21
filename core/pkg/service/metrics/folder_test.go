// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/metrics"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stored records", func() {
	var (
		cfg      metrics.ServiceConfig
		groups   *group.Service
		channels *channel.Service
		db       *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: node.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		graph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      node.DB,
			Channel: channels,
			Status:  statuses,
		}))
		framers := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			DB:           node.DB,
			Framer:       node.Framer,
			Channel:      channels,
			ChannelGraph: graph,
		}))
		cfg = metrics.ServiceConfig{
			Channel:            channels,
			Group:              groups,
			Framer:             framers,
			DB:                 node.DB,
			HostKey:            node.Identity.HostKey,
			Storage:            node.Storage,
			CollectionInterval: 5 * time.Second,
		}
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})

	named := func(ctx SpecContext, name string) []channel.Channel {
		GinkgoHelper()
		var chs []channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entries(&chs).
			Exec(ctx, db)).To(Succeed())
		return chs
	}
	const indexName = "sy_node_1_metrics_time"

	It("Should keep a metrics channel that a user renamed", func(ctx SpecContext) {
		idx := named(ctx, indexName)[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Rename(ctx, idx.Key(), "renamed_time", false)
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		Expect(named(ctx, indexName)).To(BeEmpty())
		Expect(
			named(ctx, "renamed_time"),
		).To(ConsistOf(HaveField("Name", "renamed_time")))
	})

	It("Should take the channels of its folder by name when no key is stored", func(
		ctx SpecContext,
	) {
		idx := named(ctx, indexName)[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return tx.Delete(ctx, []byte("sy_metrics_channel/"+indexName))
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		Expect(named(ctx, indexName)).To(ConsistOf(HaveField("LocalKey", idx.LocalKey)))
	})

	It("Should take the one channel of its name in another folder when no key is "+
		"stored", func(ctx SpecContext) {
		idx := named(ctx, indexName)[0]
		plant := MustSucceed(
			groups.CreateOrRetrieve(ctx, "Plant", resource.TypeChannel),
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := channels.NewWriter(tx).
				SetGroup(ctx, &plant.Key, idx.Key()); err != nil {
				return err
			}
			return tx.Delete(ctx, []byte("sy_metrics_channel/"+indexName))
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		Expect(named(ctx, indexName)).To(ConsistOf(SatisfyAll(
			HaveField("LocalKey", idx.LocalKey),
			HaveField("Group", HaveValue(Equal(plant.Key))),
		)))
	})

	It("Should open after a user renames the folder and names a root channel Metrics",
		func(ctx SpecContext) {
			folder := MustSucceed(
				groups.CreateOrRetrieve(ctx, "Metrics", resource.TypeChannel),
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := groups.NewWriter(tx).
					Rename(ctx, folder.Key, "Host metrics"); err != nil {
					return err
				}
				ch := channel.Channel{
					Name:     "Metrics",
					DataType: telem.Float32T,
					Virtual:  true,
				}
				return channels.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		},
	)

	It("Should make a new folder when a user deletes the stored one", func(
		ctx SpecContext,
	) {
		folder := MustSucceed(
			groups.CreateOrRetrieve(ctx, "Metrics", resource.TypeChannel),
		)
		var members []channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchGroups(folder.Key)).
			Entries(&members).
			Exec(ctx, db)).To(Succeed())
		Expect(members).ToNot(BeEmpty())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			keys := channel.KeysFromChannels(members)
			if err := channels.NewWriter(tx).SetGroup(ctx, nil, keys...); err != nil {
				return err
			}
			return groups.NewWriter(tx).Delete(ctx, folder.Key)
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		var again group.Group
		Expect(groups.NewRetrieve().
			Where(group.MatchNames("Metrics")).
			Where(group.MatchHasParent(false)).
			Entry(&again).
			Exec(ctx, db)).To(Succeed())
		Expect(again.Key).ToNot(Equal(folder.Key))
	})
})

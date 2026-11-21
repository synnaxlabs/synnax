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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
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

// D-242: after the first start, metrics never looks up its channels by name, and a
// name never fails a start.
var _ = Describe("Stored key breaker 2", func() {
	var (
		cfg      metrics.ServiceConfig
		groups   *group.Service
		channels *channel.Service
		statuses *status.Service
		db       *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		n := testutil.NewNode(ctx)
		db = n.DB
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: n.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: n.DB}))
		statuses = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: n.DB}))
		channels = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: n.Channel,
			DB:      n.DB,
			HostKey: n.Identity.HostKey,
			Status:  statuses,
		}))
		graph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
			DB:      n.DB,
			Channel: channels,
			Status:  statuses,
		}))
		framers := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
			DB:           n.DB,
			Framer:       n.Framer,
			Channel:      channels,
			ChannelGraph: graph,
		}))
		cfg = metrics.ServiceConfig{
			Status:             statuses,
			Channel:            channels,
			Group:              groups,
			Framer:             framers,
			DB:                 n.DB,
			HostKey:            n.Identity.HostKey,
			Storage:            n.Storage,
			CollectionInterval: 5 * time.Second,
		}
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	})
	named := func(ctx context.Context, name string) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	storedKey := func(ctx context.Context, name string) string {
		GinkgoHelper()
		var key string
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			b, closer, err := tx.Get([]byte("sy_metrics_channel/" + name))
			if err != nil {
				return err
			}
			key = string(b)
			return closer.Close()
		})).To(Succeed())
		return key
	}
	metricsFolder := func(ctx context.Context) group.Group {
		GinkgoHelper()
		return MustSucceed(
			groups.CreateOrRetrieve(ctx, "Metrics", resource.TypeChannel),
		)
	}
	plant := func(ctx context.Context) group.Group {
		GinkgoHelper()
		return MustSucceed(groups.CreateOrRetrieve(ctx, "Plant", resource.TypeChannel))
	}
	create := func(ctx context.Context, ch *channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, ch)
		})).To(Succeed())
	}
	deleteChannels := func(ctx context.Context, keys ...channel.Key) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).DeleteMany(ctx, keys, false)
		})).To(Succeed())
	}
	warning := func(ctx context.Context) []status.Status[any] {
		GinkgoHelper()
		var sts []status.Status[any]
		Expect(statuses.NewRetrieve[any]().Entries(&sts).Exec(ctx, db)).To(Succeed())
		return lo.Filter(sts, func(st status.Status[any], _ int) bool {
			return st.Key == "sy_node_1_metrics"
		})
	}

	reopen := func(ctx context.Context) {
		GinkgoHelper()
		svc := MustSucceed(metrics.OpenService(ctx, cfg))
		DeferCleanup(svc.Close)
	}

	It("Should not take a user's channel by name when the channel of a stored key "+
		"is gone", func(ctx SpecContext) {
		const memName = "sy_node_1_metrics_mem_percentage"
		deleteChannels(ctx, named(ctx, memName).Key())
		p := plant(ctx)
		user := channel.Channel{
			Name:     memName,
			DataType: telem.Int64T,
			Virtual:  true,
			Group:    &p.Key,
		}
		create(ctx, &user)
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		Expect(storedKey(ctx, memName)).ToNot(Equal(user.Key().String()))
	})

	It("Should open when the channels of the stored keys are gone and a user's "+
		"channel holds the index name", func(ctx SpecContext) {
		folder := metricsFolder(ctx)
		var members []channel.Channel
		Expect(channels.NewRetrieve().
			Where(channel.MatchGroups(folder.Key)).
			Entries(&members).
			Exec(ctx, db)).To(Succeed())
		Expect(members).ToNot(BeEmpty())
		deleteChannels(ctx, channel.KeysFromChannels(members)...)
		p := plant(ctx)
		user := channel.Channel{
			Name:     "sy_node_1_metrics_time",
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &p.Key,
		}
		create(ctx, &user)
		reopen(ctx)
	})

	It("Should name the folder that holds a metric's name in the metrics folder", func(
		ctx SpecContext,
	) {
		const memName = "sy_node_1_metrics_mem_percentage"
		deleteChannels(ctx, named(ctx, memName).Key())
		parent := metricsFolder(ctx).ResourceID()
		blocker := group.Group{
			Name:   memName,
			Parent: &parent,
			Scope:  resource.TypeChannel,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groups.NewWriter(tx).Create(ctx, &blocker)
		})).To(Succeed())
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
		Expect(warning(ctx)).To(ConsistOf(HaveField(
			"Description",
			SatisfyAll(
				ContainSubstring(memName),
				Not(ContainSubstring("0 channels are named")),
			),
		)))
	})

	It("Should open when a user renamed an input of a deleted calculated metric", func(
		ctx SpecContext,
	) {
		ts := named(ctx, "sy_node_1_metrics_ts_size_gb")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Rename(ctx, ts.Key(), "disk_ts_gb", false)
		})).To(Succeed())
		deleteChannels(ctx, named(ctx, "sy_node_1_metrics_total_size_gb").Key())
		reopen(ctx)
	})
})

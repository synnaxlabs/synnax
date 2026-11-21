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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	distnode "github.com/synnaxlabs/synnax/pkg/distribution/node"
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
		statuses *status.Service
		db       *gorp.DB
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: node.DB}))
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: node.DB}))
		statuses = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
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
			Status:             statuses,
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

	// storeDuplicate writes a channel named name in g straight to the table, as a store
	// from before the sibling rule holds it.
	storeDuplicate := func(ctx SpecContext, name string, g group.Key) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:        name,
			Leaseholder: distnode.KeyFree,
			LocalKey:    1_000_501,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       &g,
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, ch)).
			To(Succeed())
		return ch
	}
	warning := func(ctx SpecContext) []status.Status[any] {
		GinkgoHelper()
		var sts []status.Status[any]
		Expect(statuses.NewRetrieve[any]().Entries(&sts).Exec(ctx, db)).To(Succeed())
		return lo.Filter(sts, func(st status.Status[any], _ int) bool {
			return st.Key == "sy_node_1_metrics"
		})
	}
	reopen := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(MustSucceed(metrics.OpenService(ctx, cfg)).Close()).To(Succeed())
	}

	It("Should leave a metric off with a warning when several channels outside its "+
		"folder hold its name", func(ctx SpecContext) {
		const cpuName = "sy_node_1_metrics_cpu_percentage"
		cpu := named(ctx, cpuName)[0]
		plant := MustSucceed(
			groups.CreateOrRetrieve(ctx, "Plant", resource.TypeChannel),
		)
		lab := MustSucceed(groups.CreateOrRetrieve(ctx, "Lab", resource.TypeChannel))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := channels.NewWriter(tx).
				SetGroup(ctx, &plant.Key, cpu.Key()); err != nil {
				return err
			}
			return tx.Delete(ctx, []byte("sy_metrics_channel/"+cpuName))
		})).To(Succeed())
		duplicate := storeDuplicate(ctx, cpuName, lab.Key)
		reopen(ctx)
		Expect(named(ctx, cpuName)).To(HaveLen(2))
		Expect(warning(ctx)).To(ConsistOf(SatisfyAll(
			HaveField("Variant", status.VariantWarning),
			HaveField("Message", "Metrics left off"),
			HaveField("Description", ContainSubstring(fmt.Sprintf(
				"2 channels are named '%s' (keys ",
				cpuName,
			))),
			HaveField("Description", SatisfyAll(
				ContainSubstring(cpu.Key().String()),
				ContainSubstring(duplicate.Key().String()),
			)),
		)))
	})

	It("Should leave a deleted calculated metric off with a warning when its input "+
		"name is ambiguous, and clear the warning once it opens", func(ctx SpecContext) {
		const (
			totalName = "sy_node_1_metrics_total_size_gb"
			inputName = "sy_node_1_metrics_ts_size_gb"
		)
		total := named(ctx, totalName)[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Delete(ctx, total.Key(), false)
		})).To(Succeed())
		plant := MustSucceed(
			groups.CreateOrRetrieve(ctx, "Plant", resource.TypeChannel),
		)
		duplicate := storeDuplicate(ctx, inputName, plant.Key)
		reopen(ctx)
		Expect(named(ctx, totalName)).To(BeEmpty())
		Expect(warning(ctx)).To(ConsistOf(HaveField(
			"Description",
			ContainSubstring(
				fmt.Sprintf("2 channels are named '%s' (keys ", inputName),
			),
		)))
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
			Delete(ctx, duplicate.Key())).To(Succeed())
		reopen(ctx)
		Expect(named(ctx, totalName)).To(HaveLen(1))
		Expect(warning(ctx)).To(BeEmpty())
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"encoding/json"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A 0.62 store has not applied v63_clear_unissued_channel_links. The first start of
// this Core on it runs the step, but the services that open before it can create
// channels first.
var _ = Describe("ClearUnissuedChannelLinks on the first start", func() {
	It("Should clear a never-issued key that a channel created at start takes", func(
		ctx SpecContext,
	) {
		n := testutil.NewNode(ctx)
		first := MustSucceed(svctestutil.OpenLayer(ctx, n))
		firstOpen := true
		DeferCleanup(func() {
			if firstOpen {
				Expect(first.Close()).To(Succeed())
			}
		})
		const metricName = "sy_node_1_metrics_total_size_gb"
		var metric channel.Channel
		Expect(first.Channel.NewRetrieve().
			Where(channel.MatchNames(metricName)).
			Entry(&metric).
			Exec(ctx, n.DB)).To(Succeed())
		// The newest free key that the 0.62 Core issued.
		last := channel.Channel{
			Name:        "last_0_62_channel",
			DataType:    telem.Float32T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return first.Channel.NewWriter(tx).Create(ctx, &last)
		})).To(Succeed())
		// A user deleted the calculated metrics channel, which the metrics service
		// creates again on every start.
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return first.Channel.NewWriter(tx).Delete(ctx, metric.Key(), false)
		})).To(Succeed())
		// A 0.62 import kept the raw key of another Core: one this Core never issued.
		never := distchannel.NewKey(last.Leaseholder, last.LocalKey+1)
		proj := project.Project{Name: "Upgraded"}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return first.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = never
		// A second key no channel takes shows that the step runs on the next start.
		far := value
		far.Channel = distchannel.NewKey(node.KeyFree, last.LocalKey+1000)
		sch := schematic.Schematic{
			Key:     uuid.New(),
			Name:    "imported",
			Project: &proj.Key,
			Nodes: []schematic.Node{
				{Key: "value", Config: schematic.NodeConfig{Variant: value}},
				{Key: "far", Config: schematic.NodeConfig{Variant: far}},
			},
		}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[schematic.Key, schematic.Schematic](tx).Set(ctx, sch)
		})).To(Succeed())
		firstOpen = false
		Expect(first.Close()).To(Succeed())
		// The store is now as 0.62 left it: the step has not run.
		record := []byte("gorp.migration.Composition")
		b, closer := MustSucceed2(n.Storage.KV.Get(record))
		var applied []string
		Expect(json.Unmarshal(b, &applied)).To(Succeed())
		Expect(closer.Close()).To(Succeed())
		applied = slices.DeleteFunc(applied, func(k string) bool {
			return k == "v63_clear_unissued_channel_links"
		})
		b = MustSucceed(json.Marshal(applied))
		Expect(n.Storage.KV.Set(ctx, record, b)).To(Succeed())

		second := MustOpen(svctestutil.OpenLayer(ctx, n))
		var taker channel.Channel
		Expect(second.Channel.NewRetrieve().
			Where(channel.MatchKeys(never)).
			Entry(&taker).
			Exec(ctx, n.DB)).To(Succeed())
		GinkgoWriter.Printf("key %d now belongs to %q\n", never, taker.Name)
		var stored schematic.Schematic
		Expect(second.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, n.DB)).To(Succeed())
		held, ok := stored.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		far, ok := stored.Nodes[1].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(far.Channel).To(BeZero())
		// The imported symbol never named the metrics channel.
		Expect(held.Channel).To(BeZero())
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/distribution"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// The composition migrations commit in one transaction at the end of the start, after
// the services that create channels at start have committed them. A Core killed in
// between leaves those channels and no applied step.
var _ = Describe("ClearUnissuedChannelLinks after a start that did not finish", func() {
	It("Should clear a never-issued key that the unfinished start gave a channel", func(
		ctx SpecContext,
	) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		type core struct {
			node  testutil.Node
			layer *service.Layer
		}
		start := func() core {
			GinkgoHelper()
			n := testutil.OpenNode(ctx, distribution.LayerConfig{Storage: store.Layer})
			return core{node: n, layer: MustSucceed(svctestutil.OpenLayer(ctx, n))}
		}
		stop := func(c core) {
			GinkgoHelper()
			Expect(c.layer.Close()).To(Succeed())
			Expect(c.node.Close()).To(Succeed())
		}
		unapply := func() {
			GinkgoHelper()
			record := []byte("gorp.migration.Composition")
			b, closer := MustSucceed2(store.KV.Get(record))
			var applied []string
			Expect(json.Unmarshal(b, &applied)).To(Succeed())
			Expect(closer.Close()).To(Succeed())
			applied = slices.DeleteFunc(applied, func(k string) bool {
				return k == "v63_clear_unissued_channel_links"
			})
			b = MustSucceed(json.Marshal(applied))
			Expect(store.KV.Set(ctx, record, b)).To(Succeed())
		}

		// The 0.62 Core.
		old := start()
		const metricName = "sy_node_1_metrics_total_size_gb"
		var metric channel.Channel
		Expect(old.layer.Channel.NewRetrieve().
			Where(channel.MatchNames(metricName)).
			Entry(&metric).
			Exec(ctx, old.node.DB)).To(Succeed())
		last := channel.Channel{
			Name:        "last_0_62_channel",
			DataType:    telem.Float32T,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Channel.NewWriter(tx).Create(ctx, &last)
		})).To(Succeed())
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Channel.NewWriter(tx).Delete(ctx, metric.Key(), false)
		})).To(Succeed())
		never := distchannel.NewKey(last.Leaseholder, last.LocalKey+1)
		proj := project.Project{Name: "Upgraded"}
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return old.layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = never
		sch := schematic.Schematic{
			Key:     uuid.New(),
			Name:    "imported",
			Project: &proj.Key,
			Nodes: []schematic.Node{
				{Key: "value", Config: schematic.NodeConfig{Variant: value}},
			},
		}
		Expect(old.node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return gorp.WrapWriter[schematic.Key, schematic.Schematic](tx).Set(ctx, sch)
		})).To(Succeed())
		stop(old)

		// The first start of this Core on the 0.62 store: the metrics service creates
		// its channel again, then the Core dies before the composition commits.
		unfinished := start()
		var taker channel.Channel
		Expect(unfinished.layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(never)).
			Entry(&taker).
			Exec(ctx, unfinished.node.DB)).To(Succeed())
		GinkgoWriter.Printf("key %d now belongs to %q\n", never, taker.Name)
		stop(unfinished)
		unapply()

		// The next start runs the step.
		next := start()
		DeferCleanup(func() { stop(next) })
		var stored schematic.Schematic
		Expect(next.layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, next.node.DB)).To(Succeed())
		held, ok := stored.Nodes[0].Config.Variant.(schematic.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		// The imported symbol never named the channel that the start created.
		Expect(held.Channel).To(BeZero())
	})
})

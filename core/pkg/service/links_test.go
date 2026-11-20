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
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// A 0.62 import kept the raw channel keys of another Core in schematics and tables.
var _ = Describe("ClearUnissuedChannelLinks", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		dist  testutil.Node
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = testutil.NewNode(ctx)
		db = dist.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, dist))
		proj = project.Project{Name: "Upgraded"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createChannel := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}

	// storeUpgraded writes a schematic and a table that hold keys, as a 0.62 import
	// left
	// them: with no reference check.
	storeUpgraded := func(
		ctx SpecContext,
		keys ...channel.Key,
	) (schematic.Schematic, table.Table) {
		GinkgoHelper()
		sch := schematic.Schematic{
			Key:     uuid.New(),
			Name:    "imported",
			Project: &proj.Key,
		}
		tbl := table.Table{Key: uuid.New(), Name: "imported", Project: &proj.Key}
		row := table.Row{Key: "r0", Size: 30}
		for i, k := range keys {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = k
			sch.Nodes = append(sch.Nodes, schematic.Node{
				Key:    fmt.Sprintf("value_%d", i),
				Config: schematic.NodeConfig{Variant: value},
			})
			cfg := defaults.Fill(table.CellConfig{
				Variant: table.ValueCellConfig{Channel: k},
			})
			tbl.Columns = append(tbl.Columns, table.Column{
				Key:  fmt.Sprintf("c%d", i),
				Size: 80,
			})
			row.Cells = append(row.Cells, table.Cell{
				Key:    fmt.Sprintf("cell_%d", i),
				Config: cfg,
			})
		}
		tbl.Rows = []table.Row{row}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := gorp.WrapWriter[schematic.Key, schematic.Schematic](tx).
				Set(ctx, sch); err != nil {
				return err
			}
			return gorp.WrapWriter[table.Key, table.Table](tx).Set(ctx, tbl)
		})).To(Succeed())
		return sch, tbl
	}

	// clearLinks runs the step as a Core that opened after every channel so far.
	clearLinks := func(ctx SpecContext) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: uuid.New().String(),
			Migrations: []gorp.Migration{
				service.ClearUnissuedChannelLinks(
					layer.Schematic,
					layer.Table,
					dist.Channel.Issued,
				),
			},
		})).To(Succeed())
	}

	schematicKeys := func(ctx SpecContext, k schematic.Key) []channel.Key {
		GinkgoHelper()
		var sch schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(k)).
			Entry(&sch).
			Exec(ctx, db)).To(Succeed())
		keys := make([]channel.Key, len(sch.Nodes))
		for i, n := range sch.Nodes {
			value, ok := n.Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			keys[i] = value.Channel
		}
		return keys
	}

	tableKeys := func(ctx SpecContext, k table.Key) []channel.Key {
		GinkgoHelper()
		var tbl table.Table
		Expect(layer.Table.NewRetrieve().
			Where(table.MatchKeys(k)).
			Entry(&tbl).
			Exec(ctx, db)).To(Succeed())
		keys := make([]channel.Key, len(tbl.Rows[0].Cells))
		for i, c := range tbl.Rows[0].Cells {
			value, ok := c.Config.Variant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			keys[i] = value.Channel
		}
		return keys
	}

	It("Should keep its key, which the applied set holds forever", func() {
		step := service.ClearUnissuedChannelLinks(
			layer.Schematic,
			layer.Table,
			dist.Channel.Issued,
		)
		Expect(step.Key()).To(Equal("v63_clear_unissued_channel_links"))
	})

	It("Should clear a never-issued key and keep a deleted and a live key", func(
		ctx SpecContext,
	) {
		live := createChannel(ctx)
		dead := createChannel(ctx)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		never := distchannel.NewKey(live.Leaseholder, live.LocalKey+1000)
		sch, tbl := storeUpgraded(ctx, never, dead.Key(), live.Key())
		clearLinks(ctx)
		Expect(schematicKeys(ctx, sch.Key)).To(Equal(
			[]channel.Key{0, dead.Key(), live.Key()},
		))
		Expect(tableKeys(ctx, tbl.Key)).To(Equal(
			[]channel.Key{0, dead.Key(), live.Key()},
		))
	})

	It("Should keep a key leased to another node", func(ctx SpecContext) {
		live := createChannel(ctx)
		other := distchannel.NewKey(node.Key(2), live.LocalKey+500)
		sch, tbl := storeUpgraded(ctx, other)
		clearLinks(ctx)
		Expect(schematicKeys(ctx, sch.Key)).To(Equal([]channel.Key{other}))
		Expect(tableKeys(ctx, tbl.Key)).To(Equal([]channel.Key{other}))
	})

	It("Should let a write hold a key leased to another node", func(ctx SpecContext) {
		live := createChannel(ctx)
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = distchannel.NewKey(node.Key(2), live.LocalKey+600)
		sch := schematic.Schematic{
			Name:    "other node",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
	})

	It("Should let an upgraded schematic be copied", func(ctx SpecContext) {
		live := createChannel(ctx)
		sch, _ := storeUpgraded(
			ctx,
			distchannel.NewKey(live.Leaseholder, live.LocalKey+2000),
		)
		clearLinks(ctx)
		var cp schematic.Schematic
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Copy(ctx, sch.Key, "copy", &cp)
		})).To(Succeed())
	})

	It("Should let the removal of an upgraded symbol be undone", func(ctx SpecContext) {
		live := createChannel(ctx)
		sch, _ := storeUpgraded(
			ctx,
			distchannel.NewKey(live.Leaseholder, live.LocalKey+3000),
		)
		clearLinks(ctx)
		apply := func(act schematic.Action) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := layer.Schematic.NewWriter(tx).Apply(ctx, sch.Key, act)
				return err
			})
		}
		var stored schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(apply(schematic.NewRemoveNodeAction(
			schematic.RemoveNodePayload{Key: "value_0"},
		))).To(Succeed())
		Expect(apply(schematic.NewSetNodeAction(
			schematic.SetNodePayload{Node: stored.Nodes[0]},
		))).To(Succeed())
	})

	It("Should not bind an upgraded symbol to the next channel the Core creates", func(
		ctx SpecContext,
	) {
		last := createChannel(ctx)
		never := distchannel.NewKey(last.Leaseholder, last.LocalKey+1)
		sch, _ := storeUpgraded(ctx, never)
		clearLinks(ctx)
		next := createChannel(ctx)
		Expect(next.Key()).To(Equal(never))
		uses := MustSucceed(layer.Channel.Uses(ctx, db, []channel.Key{next.Key()}))
		Expect(uses[0]).ToNot(ContainElement(HaveField(
			"Holder", gorp.NewTarget("schematic", sch.Key),
		)))
	})
})

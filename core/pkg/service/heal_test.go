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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Heal", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Heal"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	pair := func(ctx SpecContext) (dead, live channel.Channel) {
		GinkgoHelper()
		chs := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		return chs[0], chs[1]
	}

	// takeOver deletes dead and renames live onto its path, so the heal moves each link
	// of dead onto live.
	takeOver := func(ctx SpecContext, dead, live channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Rename(ctx, live.Key(), dead.Name, false)
		})).To(Succeed())
	}

	It("Should keep both nodes of a schematic that holds the dead and the healing "+
		"channel", func(ctx SpecContext) {
		dead, live := pair(ctx)
		node := func(key string, ch channel.Key) schematic.Node {
			var value schematic.ValueNodeConfig
			value.SetDefaults()
			value.Channel = ch
			return schematic.Node{
				Key:    key,
				Config: schematic.NodeConfig{Variant: value},
			}
		}
		sch := schematic.Schematic{
			Name:    "heal",
			Project: &proj.Key,
			Nodes:   []schematic.Node{node("a", dead.Key()), node("b", live.Key())},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		takeOver(ctx, dead, live)
		var stored schematic.Schematic
		Expect(layer.Schematic.NewRetrieve().
			Where(schematic.MatchKeys(sch.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		held := make([]channel.Key, len(stored.Nodes))
		for i, n := range stored.Nodes {
			value, ok := n.Config.Variant.(schematic.ValueNodeConfig)
			Expect(ok).To(BeTrue())
			held[i] = value.Channel
		}
		Expect(held).To(Equal([]channel.Key{live.Key(), live.Key()}))
	})

	It("Should keep both cells of a table that holds the dead and the healing channel",
		func(ctx SpecContext) {
			dead, live := pair(ctx)
			cell := func(key string, ch channel.Key) table.Cell {
				var value table.ValueCellConfig
				value.SetDefaults()
				value.Channel = ch
				return table.Cell{Key: key, Config: table.CellConfig{Variant: value}}
			}
			tbl := table.Table{
				Name:    "heal",
				Project: &proj.Key,
				Columns: []table.Column{{Key: "a", Size: 72}, {Key: "b", Size: 72}},
				Rows: []table.Row{{
					Key:   "row",
					Size:  36,
					Cells: []table.Cell{cell("a", dead.Key()), cell("b", live.Key())},
				}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Table.NewWriter(tx).Create(ctx, &tbl)
			})).To(Succeed())
			takeOver(ctx, dead, live)
			var stored table.Table
			Expect(layer.Table.NewRetrieve().
				Where(table.MatchKeys(tbl.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			cells := stored.Rows[0].Cells
			held := make([]channel.Key, len(cells))
			for i, c := range cells {
				value, ok := c.Config.Variant.(table.ValueCellConfig)
				Expect(ok).To(BeTrue())
				held[i] = value.Channel
			}
			Expect(held).To(Equal([]channel.Key{live.Key(), live.Key()}))
		})
})

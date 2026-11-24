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

var _ = Describe("Heal break p2", Ordered, func() {
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
		proj = project.Project{Name: "heal-break-p2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	It("Should heal a setpoint's command and state and a table cell in one create",
		func(ctx SpecContext) {
			dead := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &dead)
			})).To(Succeed())
			var setpoint schematic.SetpointNodeConfig
			setpoint.SetDefaults()
			setpoint.CommandChannel, setpoint.StateChannel = dead.Key(), dead.Key()
			sch := schematic.Schematic{
				Name:    "heal",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "setpoint",
					Config: schematic.NodeConfig{Variant: setpoint},
				}},
			}
			var cell table.ValueCellConfig
			cell.SetDefaults()
			cell.Channel = dead.Key()
			tbl := table.Table{
				Name:    "heal",
				Project: &proj.Key,
				Columns: []table.Column{{Key: "a", Size: 72}},
				Rows: []table.Row{{
					Key:  "row",
					Size: 36,
					Cells: []table.Cell{{
						Key:    "a",
						Config: table.CellConfig{Variant: cell},
					}},
				}},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Schematic.NewWriter(tx).Create(ctx, &sch); err != nil {
					return err
				}
				if err := layer.Table.NewWriter(tx).Create(ctx, &tbl); err != nil {
					return err
				}
				return layer.Channel.NewWriter(tx).Delete(ctx, dead.Key(), false)
			})).To(Succeed())
			live := channel.Channel{
				Name:     dead.Name,
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &live)
			})).To(Succeed())
			var storedSch schematic.Schematic
			Expect(layer.Schematic.NewRetrieve().
				Where(schematic.MatchKeys(sch.Key)).
				Entry(&storedSch).
				Exec(ctx, db)).To(Succeed())
			variant := storedSch.Nodes[0].Config.Variant
			healed, ok := variant.(schematic.SetpointNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(healed.CommandChannel).To(Equal(live.Key()))
			Expect(healed.StateChannel).To(Equal(live.Key()))
			var storedTbl table.Table
			Expect(layer.Table.NewRetrieve().
				Where(table.MatchKeys(tbl.Key)).
				Entry(&storedTbl).
				Exec(ctx, db)).To(Succeed())
			cellVariant := storedTbl.Rows[0].Cells[0].Config.Variant
			healedCell, ok := cellVariant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			Expect(healedCell.Channel).To(Equal(live.Key()))
		})
})

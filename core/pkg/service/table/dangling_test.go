// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// issuedChannel stands in for the channel table that a table cell's weak link needs.
type issuedChannel struct{ Key channel.Key }

func (c issuedChannel) GorpKey() channel.Key { return c.Key }

var _ = Describe("DanglingMigration", func() {
	It("Should clear each channel key that issued reports as never issued", func(
		ctx SpecContext,
	) {
		fresh := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(fresh.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[channel.Key, issuedChannel]{
			DB:     fresh,
			Name:   "channel",
			Issued: func(channel.Key) bool { return true },
		}))
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: fresh}))
		MustOpen(color.OpenService(ctx, color.ServiceConfig{DB: fresh, Group: groups}))
		MustOpen(project.OpenService(ctx, project.ServiceConfig{DB: fresh}))
		tables := MustOpen(table.OpenService(ctx, table.ServiceConfig{
			DB:   fresh,
			ImEx: imex.NewService(),
		}))
		const kept, cleared channel.Key = 1<<20 | 3, 1<<20 | 4
		row := table.Row{Key: "r0", Size: 30}
		tbl := table.Table{Key: uuid.New(), Name: "imported"}
		for i, k := range []channel.Key{kept, cleared} {
			tbl.Columns = append(tbl.Columns, table.Column{
				Key:  []string{"c0", "c1"}[i],
				Size: 80,
			})
			row.Cells = append(row.Cells, table.Cell{
				Key: []string{"cell_0", "cell_1"}[i],
				Config: defaults.Fill(table.CellConfig{
					Variant: table.ValueCellConfig{Channel: k},
				}),
			})
		}
		tbl.Rows = []table.Row{row}
		Expect(gorp.WrapWriter[table.Key, table.Table](fresh).Set(ctx, tbl)).
			To(Succeed())
		issued := func(t gorp.Target) (bool, error) {
			return t != gorp.NewTarget("channel", cleared), nil
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         fresh,
			Namespace:  "table_dangling",
			Migrations: []gorp.Migration{tables.DanglingMigration("clear", issued)},
		})).To(Succeed())
		var got table.Table
		Expect(tables.NewRetrieve().
			Where(table.MatchKeys(tbl.Key)).
			Entry(&got).
			Exec(ctx, fresh)).To(Succeed())
		channels := make([]channel.Key, len(got.Rows[0].Cells))
		for i, cell := range got.Rows[0].Cells {
			value, ok := cell.Config.Variant.(table.ValueCellConfig)
			Expect(ok).To(BeTrue())
			channels[i] = value.Channel
		}
		Expect(channels).To(Equal([]channel.Key{kept, 0}))
	})
})

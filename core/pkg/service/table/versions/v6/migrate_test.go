// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v6"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv1 "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should store every cell color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Table]{DB: db}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := v5.Table{
				Key:     uuid.New(),
				Name:    "t",
				Rows:    []v5.Row{{Key: "r", Cells: []string{"text", "value"}}},
				Columns: []v5.Column{{Key: "c0"}, {Key: "c1"}},
			}
			old.Cells = map[string]v5.CellConfig{
				"text": {Variant: v5.TextCellConfig{FillColor: c(1), TextColor: c(2)}},
				"value": {Variant: v5.ValueCellConfig{
					FillColor:      c(3),
					TextColor:      c(4),
					StalenessColor: c(5),
					Redline: colorv1.Scale{Bands: []colorv1.Band{
						{Key: "b", Color: *c(6)},
					}},
				}},
			}
			Expect(gorp.NewCreate[v5.Key, v5.Table]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v6.Key, v6.Table]{
				DB: db, Migrations: []gorp.Migration{v6.Migration},
			}))
			var got v6.Table
			Expect(gorp.NewRetrieve[v6.Key, v6.Table]().
				Where(gorp.MatchKeys[v6.Key, v6.Table](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			n := func(r uint8) *colorv2.Color { return &colorv2.Color{R: r, A: expected} }
			cells := got.Rows[0].Cells
			Expect(cells[0].Key).To(Equal("text"))
			text, ok := cells[0].Config.Variant.(v6.TextCellConfig)
			Expect(ok).To(BeTrue())
			Expect(text.FillColor).To(Equal(n(1)))
			Expect(text.TextColor).To(Equal(n(2)))
			Expect(cells[1].Key).To(Equal("value"))
			value, ok := cells[1].Config.Variant.(v6.ValueCellConfig)
			Expect(ok).To(BeTrue())
			Expect(value.FillColor).To(Equal(n(3)))
			Expect(value.TextColor).To(Equal(n(4)))
			Expect(value.StalenessColor).To(Equal(n(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]colorv2.Band{{Key: "b", Color: *n(6)}}))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)
})

var _ = Describe("MigrateTable", func() {
	text := func(value string) v5.CellConfig {
		return v5.CellConfig{Variant: v5.TextCellConfig{Value: value}}
	}
	cell := func(key, value string) v6.Cell {
		return v6.Cell{Key: key, Config: v6.CellConfig{
			Variant: v6.TextCellConfig{Value: value},
		}}
	}
	empty := func(key string) v6.Cell {
		c := v6.Cell{Key: key, Config: v6.CellConfig{Variant: v6.TextCellConfig{}}}
		c.Config = defaults.Fill(c.Config)
		return c
	}
	cols := func(n int) []v5.Column {
		c := make([]v5.Column, n)
		for i := range c {
			c[i] = v5.Column{Key: fmt.Sprintf("c%d", i), Size: 72}
		}
		return c
	}
	It(
		"Should move each cell config into the row cell that references it",
		func(ctx SpecContext) {
			out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
				Rows: []v5.Row{
					{Key: "r0", Size: 40, Cells: []string{"a", "b"}},
					{Key: "r1", Size: 50, Cells: []string{"c", "d"}},
				},
				Columns: cols(2),
				Cells: map[string]v5.CellConfig{
					"a": text("A"),
					"b": text("B"),
					"c": text("C"),
					"d": text("D"),
				},
			}))
			Expect(out.Rows).To(Equal([]v6.Row{
				{Key: "r0", Size: 40, Cells: []v6.Cell{cell("a", "A"), cell("b", "B")}},
				{Key: "r1", Size: 50, Cells: []v6.Cell{cell("c", "C"), cell("d", "D")}},
			}))
		},
	)
	It(
		"Should give a row cell with no config an empty text cell",
		func(ctx SpecContext) {
			out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
				Rows:    []v5.Row{{Key: "r", Cells: []string{"a", "missing", "b"}}},
				Columns: cols(3),
				Cells:   map[string]v5.CellConfig{"a": text("A"), "b": text("B")},
			}))
			Expect(out.Rows[0].Cells).To(Equal([]v6.Cell{
				cell("a", "A"),
				empty("missing"),
				cell("b", "B"),
			}))
		},
	)
	It(
		"Should give each later cell that repeats a key a new key",
		func(ctx SpecContext) {
			out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
				Rows: []v5.Row{
					{Key: "r0", Cells: []string{"a", "a-1"}},
					{Key: "r1", Cells: []string{"a", "b"}},
				},
				Columns: cols(2),
				Cells: map[string]v5.CellConfig{
					"a":   text("A"),
					"a-1": text("B"),
					"b":   text("C"),
				},
			}))
			Expect(
				out.Rows[0].Cells,
			).To(Equal([]v6.Cell{cell("a", "A"), cell("a-1", "B")}))
			Expect(
				out.Rows[1].Cells,
			).To(Equal([]v6.Cell{cell("a-2", "A"), cell("b", "C")}))
		},
	)
	It("Should drop the cells past the last column", func(ctx SpecContext) {
		out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
			Rows:    []v5.Row{{Key: "r", Cells: []string{"a", "b"}}},
			Columns: cols(1),
			Cells:   map[string]v5.CellConfig{"a": text("A"), "b": text("B")},
		}))
		Expect(out.Rows[0].Cells).To(Equal([]v6.Cell{cell("a", "A")}))
	})
	It("Should fill a short row with empty text cells", func(ctx SpecContext) {
		out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
			Rows:    []v5.Row{{Key: "r", Cells: []string{"a"}}},
			Columns: cols(3),
			Cells:   map[string]v5.CellConfig{"a": text("A")},
		}))
		Expect(out.Rows[0].Cells).To(Equal([]v6.Cell{
			cell("a", "A"),
			empty("r-1"),
			empty("r-2"),
		}))
	})
	It("Should drop a cell that no row references", func(ctx SpecContext) {
		out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
			Rows:    []v5.Row{{Key: "r", Cells: []string{"a"}}},
			Columns: cols(1),
			Cells:   map[string]v5.CellConfig{"a": text("A"), "orphan": text("O")},
		}))
		Expect(out.Rows[0].Cells).To(Equal([]v6.Cell{cell("a", "A")}))
	})
	It("Should store the cell color alpha as a byte", func(ctx SpecContext) {
		out := MustSucceed(v6.MigrateTable(ctx, v5.Table{
			Rows:    []v5.Row{{Key: "r", Cells: []string{"a"}}},
			Columns: cols(1),
			Cells: map[string]v5.CellConfig{
				"a": {
					Variant: v5.TextCellConfig{FillColor: &color.Color{R: 1, A: 0.5}},
				},
			},
		}))
		text, ok := out.Rows[0].Cells[0].Config.Variant.(v6.TextCellConfig)
		Expect(ok).To(BeTrue())
		Expect(text.FillColor).To(Equal(&colorv2.Color{R: 1, A: 128}))
	})
})

var _ = Describe("NameMigration", func() {
	migrated := func(ctx SpecContext, in v6.Table) v6.Table {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		Expect(gorp.NewCreate[v6.Key, v6.Table]().Entry(&in).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Table",
			Migrations: []gorp.Migration{v6.NameMigration},
		})).To(Succeed())
		var out v6.Table
		Expect(gorp.NewRetrieve[v6.Key, v6.Table]().
			Where(gorp.MatchKeys[v6.Key, v6.Table](in.Key)).
			Entry(&out).Exec(ctx, db)).To(Succeed())
		return out
	}
	key := uuid.MustParse("1a2b3c4d-0000-0000-0000-000000000000")
	DescribeTable(
		"Should name only nameless tables",
		func(ctx SpecContext, name, expected string) {
			Expect(migrated(ctx, v6.Table{Key: key, Name: name}).Name).
				To(Equal(expected))
		},
		Entry("empty name", "", "Table 1a2b3c4d"),
		Entry("named table", "Named", "Named"),
	)
})

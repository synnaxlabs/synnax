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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

// cell constructs a Cell with the given key and variant, defaulting props to nil.
func cell(key, variant string) table.Cell {
	return table.Cell{Key: key, Variant: variant}
}

// cellWithProps constructs a Cell with the given key, variant, and props.
func cellWithProps(key, variant string, props msgpack.EncodedJSON) table.Cell {
	return table.Cell{Key: key, Variant: variant, Props: props}
}

// create2x2 returns a 2x2 table with cells a,b across row 0 and c,d across row 1.
// Variant is "text" with no props on every cell.
func create2x2() table.Table {
	return table.Table{
		Name: "t",
		Rows: []table.Row{
			{Size: 30, Cells: []string{"a", "b"}},
			{Size: 30, Cells: []string{"c", "d"}},
		},
		Columns: []table.Column{{Size: 80}, {Size: 80}},
		Cells: map[string]table.Cell{
			"a": cell("a", "text"),
			"b": cell("b", "text"),
			"c": cell("c", "text"),
			"d": cell("d", "text"),
		},
	}
}

// create3x3 returns a 3x3 table with cells a..i in row-major order. Variant is
// "value" on every cell; props are nil.
func create3x3() table.Table {
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	cells := make(map[string]table.Cell, len(keys))
	for _, k := range keys {
		cells[k] = cell(k, "value")
	}
	return table.Table{
		Name: "t",
		Rows: []table.Row{
			{Size: 36, Cells: []string{"a", "b", "c"}},
			{Size: 36, Cells: []string{"d", "e", "f"}},
			{Size: 36, Cells: []string{"g", "h", "i"}},
		},
		Columns: []table.Column{{Size: 80}, {Size: 80}, {Size: 80}},
		Cells:   cells,
	}
}

var _ = Describe("Reducer", func() {
	Describe("Rename", func() {
		It("Should replace the table's name", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewRenameAction(table.RenamePayload{
				Name: "renamed",
			})))
			Expect(out.Name).To(Equal("renamed"))
		})
	})

	Describe("AddRow", func() {
		It("Should append a row with the given cells to the cells map", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewAddRowAction(table.AddRowPayload{
				Index: 2,
				Size:  40,
				Cells: []table.Cell{
					cellWithProps("e", "text", msgpack.EncodedJSON{"value": "E"}),
					cellWithProps("f", "text", msgpack.EncodedJSON{"value": "F"}),
				},
			})))
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Rows[2].Cells).To(Equal([]string{"e", "f"}))
			Expect(out.Cells["e"].Props["value"]).To(Equal("E"))
		})

		It("Should clamp out-of-range indices to the end of the rows slice", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewAddRowAction(table.AddRowPayload{
				Index: 999,
				Size:  50,
				Cells: []table.Cell{cell("z", "text")},
			})))
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Rows[2].Cells).To(Equal([]string{"z"}))
		})

		It("Should bootstrap one default-sized column per cell against an empty table", func() {
			out := MustSucceed(table.Reduce(table.Table{}, table.NewAddRowAction(table.AddRowPayload{
				Index: 0,
				Size:  36,
				Cells: []table.Cell{cell("a", "text"), cell("b", "text")},
			})))
			Expect(out.Rows).To(HaveLen(1))
			Expect(out.Columns).To(HaveLen(2))
			Expect(out.Columns[0].Size).To(Equal(72.0))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a", "b"}))
		})
	})

	Describe("RemoveRow", func() {
		It("Should remove the row and drop every cell it referenced", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewRemoveRowAction(table.RemoveRowPayload{
				Index: 0,
			})))
			Expect(out.Rows).To(HaveLen(1))
			Expect(out.Rows[0].Cells).To(Equal([]string{"c", "d"}))
			Expect(out.Cells).NotTo(HaveKey("a"))
			Expect(out.Cells).NotTo(HaveKey("b"))
		})

		It("Should be a no-op when the index is out of range", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewRemoveRowAction(table.RemoveRowPayload{
				Index: 99,
			})))
			Expect(out.Rows).To(HaveLen(2))
			Expect(out.Cells).To(HaveLen(4))
		})
	})

	Describe("AddCol", func() {
		It("Should insert a column at the given index across every row", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewAddColAction(table.AddColPayload{
				Index: 1,
				Size:  90,
				Cells: []table.Cell{
					cellWithProps("m1", "text", msgpack.EncodedJSON{"value": "M1"}),
					cellWithProps("m2", "text", msgpack.EncodedJSON{"value": "M2"}),
				},
			})))
			Expect(out.Columns).To(HaveLen(3))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a", "m1", "b"}))
			Expect(out.Rows[1].Cells).To(Equal([]string{"c", "m2", "d"}))
		})

		It("Should bootstrap one default-sized row per cell against an empty table", func() {
			out := MustSucceed(table.Reduce(table.Table{}, table.NewAddColAction(table.AddColPayload{
				Index: 0,
				Size:  72,
				Cells: []table.Cell{cell("a", "text"), cell("b", "text")},
			})))
			Expect(out.Columns).To(HaveLen(1))
			Expect(out.Rows).To(HaveLen(2))
			Expect(out.Rows[0].Size).To(Equal(36.0))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a"}))
			Expect(out.Rows[1].Cells).To(Equal([]string{"b"}))
		})
	})

	Describe("RemoveCol", func() {
		It("Should remove the column and drop every cell in it across every row", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewRemoveColAction(table.RemoveColPayload{
				Index: 0,
			})))
			Expect(out.Columns).To(HaveLen(1))
			Expect(out.Rows[0].Cells).To(Equal([]string{"b"}))
			Expect(out.Rows[1].Cells).To(Equal([]string{"d"}))
			Expect(out.Cells).NotTo(HaveKey("a"))
			Expect(out.Cells).NotTo(HaveKey("c"))
		})
	})

	Describe("CellTemplate", func() {
		// templateKey is a 36-character UUID v4 whose last four hex digits will
		// be replaced with the per-replica index when the reducer expands it.
		const templateKey = "11111111-2222-4333-8444-555555555555"
		template := &table.Cell{Key: templateKey, Variant: "text"}

		DescribeTable("replicates the template across the opposing axis",
			func(state table.Table, action table.Action, wantRow0 []string) {
				out := MustSucceed(table.Reduce(state, action))
				Expect(out.Rows[0].Cells).To(Equal(wantRow0))
				Expect(out.Cells["11111111-2222-4333-8444-555555550000"].Variant).
					To(Equal("text"))
			},
			Entry("AddRow against two existing columns",
				create2x2(),
				table.NewAddRowAction(table.AddRowPayload{
					Index: 2, Size: 36, CellTemplate: template,
				}),
				[]string{"a", "b"}, // existing row unchanged; new row's keys checked via Cells map
			),
			Entry("AddCol against two existing rows",
				create2x2(),
				table.NewAddColAction(table.AddColPayload{
					Index: 2, Size: 72, CellTemplate: template,
				}),
				[]string{"a", "b", "11111111-2222-4333-8444-555555550000"},
			),
		)

		It("Should seed a 1x1 grid when AddRow with a template fires against an empty table", func() {
			out := MustSucceed(table.Reduce(table.Table{}, table.NewAddRowAction(table.AddRowPayload{
				Index: 0, Size: 36, CellTemplate: template,
			})))
			Expect(out.Columns).To(HaveLen(1))
			Expect(out.Rows).To(HaveLen(1))
			Expect(out.Rows[0].Cells).To(Equal([]string{"11111111-2222-4333-8444-555555550000"}))
		})

		It("Should add a column with no cells when AddCol with a template fires against a row-less table", func() {
			out := MustSucceed(table.Reduce(table.Table{
				Columns: []table.Column{{Size: 80}, {Size: 80}},
			}, table.NewAddColAction(table.AddColPayload{
				Index: 2, Size: 72, CellTemplate: template,
			})))
			Expect(out.Rows).To(BeEmpty())
			Expect(out.Columns).To(HaveLen(3))
			Expect(out.Cells).To(BeEmpty())
		})

		It("Should add a row with no cells when AddRow with a template fires against a column-less table", func() {
			out := MustSucceed(table.Reduce(table.Table{
				Rows: []table.Row{{Size: 36, Cells: []string{}}},
			}, table.NewAddRowAction(table.AddRowPayload{
				Index: 1, Size: 36, CellTemplate: template,
			})))
			Expect(out.Rows).To(HaveLen(2))
			Expect(out.Rows[1].Cells).To(BeEmpty())
			Expect(out.Columns).To(BeEmpty())
			Expect(out.Cells).To(BeEmpty())
		})
	})

	Describe("Resize and clamping", func() {
		const minDim = 32.0

		DescribeTable("clamps sizes below the minimum cell dimension",
			func(action table.Action, projection func(table.Table) float64) {
				out := MustSucceed(table.Reduce(create2x2(), action))
				Expect(projection(out)).To(Equal(minDim))
			},
			Entry("AddRow",
				table.NewAddRowAction(table.AddRowPayload{
					Index: 2, Size: 5, Cells: []table.Cell{cell("z", "text"), cell("y", "text")},
				}),
				func(t table.Table) float64 { return t.Rows[2].Size },
			),
			Entry("AddCol",
				table.NewAddColAction(table.AddColPayload{
					Index: 2, Size: 0, Cells: []table.Cell{cell("z", "text"), cell("y", "text")},
				}),
				func(t table.Table) float64 { return t.Columns[2].Size },
			),
			Entry("ResizeRow",
				table.NewResizeRowAction(table.ResizeRowPayload{Index: 0, Size: 5}),
				func(t table.Table) float64 { return t.Rows[0].Size },
			),
			Entry("ResizeCol",
				table.NewResizeColAction(table.ResizeColPayload{Index: 0, Size: 0}),
				func(t table.Table) float64 { return t.Columns[0].Size },
			),
		)

		It("Should resize a row and a column independently", func() {
			out := MustSucceed(table.Reduce(create2x2(),
				table.NewResizeRowAction(table.ResizeRowPayload{Index: 0, Size: 55}),
				table.NewResizeColAction(table.ResizeColPayload{Index: 1, Size: 200}),
			))
			Expect(out.Rows[0].Size).To(Equal(55.0))
			Expect(out.Columns[1].Size).To(Equal(200.0))
		})
	})

	Describe("SetCell", func() {
		It("Should replace the cell stored under the given key", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewSetCellAction(table.SetCellPayload{
				Cell: cellWithProps("a", "value", msgpack.EncodedJSON{"units": "psi"}),
			})))
			Expect(out.Cells["a"].Variant).To(Equal("value"))
			Expect(out.Cells["a"].Props["units"]).To(Equal("psi"))
			Expect(out.Cells).To(HaveLen(4))
		})

		It("Should be a no-op when no cell with the given key exists", func() {
			out := MustSucceed(table.Reduce(create2x2(), table.NewSetCellAction(table.SetCellPayload{
				Cell: cell("ghost", "text"),
			})))
			Expect(out.Cells).NotTo(HaveKey("ghost"))
			Expect(out.Cells).To(HaveLen(4))
		})
	})

	Describe("EraseCells", func() {
		template := table.CellTemplate{Variant: "text"}

		DescribeTable("removes fully-selected axes and resets surviving cells",
			func(selection []string, wantRows int, wantCols int, wantRow0 []string) {
				out := MustSucceed(table.Reduce(create3x3(), table.NewEraseCellsAction(table.EraseCellsPayload{
					Cells: selection, Template: template,
				})))
				Expect(out.Rows).To(HaveLen(wantRows))
				Expect(out.Columns).To(HaveLen(wantCols))
				if len(out.Rows) > 0 {
					Expect(out.Rows[0].Cells).To(Equal(wantRow0))
				}
			},
			Entry("partial selection: surviving cells reset to template, axes untouched",
				[]string{"b", "e"}, 3, 3, []string{"a", "b", "c"},
			),
			Entry("fully-selected row: row removed, columns untouched",
				[]string{"d", "e", "f"}, 2, 3, []string{"a", "b", "c"},
			),
			Entry("fully-selected column: column removed, rows untouched",
				[]string{"b", "e", "h"}, 3, 2, []string{"a", "c"},
			),
			Entry("full row + full column in one call",
				[]string{"a", "b", "c", "f", "i"}, 2, 2, []string{"d", "e"},
			),
		)

		It("Should reset partial selections to the template variant", func() {
			out := MustSucceed(table.Reduce(create3x3(), table.NewEraseCellsAction(table.EraseCellsPayload{
				Cells: []string{"b", "e"}, Template: template,
			})))
			Expect(out.Cells["b"].Variant).To(Equal("text"))
			Expect(out.Cells["e"].Variant).To(Equal("text"))
			Expect(out.Cells["a"].Variant).To(Equal("value"))
		})

		It("Should drop cells from the cells map when their row is fully removed", func() {
			out := MustSucceed(table.Reduce(create3x3(), table.NewEraseCellsAction(table.EraseCellsPayload{
				Cells: []string{"d", "e", "f"}, Template: template,
			})))
			for _, k := range []string{"d", "e", "f"} {
				Expect(out.Cells).NotTo(HaveKey(k))
			}
		})

		It("Should be a no-op for an empty selection", func() {
			out := MustSucceed(table.Reduce(create3x3(), table.NewEraseCellsAction(table.EraseCellsPayload{
				Cells: []string{}, Template: template,
			})))
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Cells).To(HaveLen(9))
		})
	})
})

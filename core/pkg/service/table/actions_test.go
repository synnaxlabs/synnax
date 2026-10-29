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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// textCfg constructs a complete text cell config with the given content, matching what
// a client sends once its own defaults have filled.
func textCfg(value string) table.CellConfig {
	cfg := table.CellConfig{Variant: table.TextCellConfig{Value: value}}
	cfg.ApplyDefaults()
	return cfg
}

// valueCfg constructs a complete value cell config with the given unit suffix, matching
// what a client sends once its own defaults have filled.
func valueCfg(units string) table.CellConfig {
	cfg := table.CellConfig{Variant: table.ValueCellConfig{Units: units}}
	cfg.ApplyDefaults()
	return cfg
}

// cell constructs a Cell with the given key and config.
func cell(key string, cfg table.CellConfig) table.Cell {
	return table.Cell{Key: key, Config: cfg}
}

// asText asserts that cfg holds a text variant and returns it.
func asText(cfg table.CellConfig) table.TextCellConfig {
	GinkgoHelper()
	text, ok := cfg.Variant.(table.TextCellConfig)
	return MustBeOk(text, ok)
}

// create2x2 returns a 2x2 table with cells a,b across row r0 and c,d across row r1, in
// columns c0 and c1. Every cell is an empty text cell.
func create2x2() table.Table {
	return table.Table{
		Name: "t",
		Rows: []table.Row{
			{Key: "r0", Size: 30, Cells: []string{"a", "b"}},
			{Key: "r1", Size: 30, Cells: []string{"c", "d"}},
		},
		Columns: []table.Column{{Key: "c0", Size: 80}, {Key: "c1", Size: 80}},
		Cells: map[string]table.CellConfig{
			"a": textCfg(""),
			"b": textCfg(""),
			"c": textCfg(""),
			"d": textCfg(""),
		},
	}
}

// create3x3 returns a 3x3 table with cells a..i in row-major order, in rows r0..r2 and
// columns c0..c2. Every cell is an empty value cell.
func create3x3() table.Table {
	keys := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	cells := make(map[string]table.CellConfig, len(keys))
	for _, k := range keys {
		cells[k] = valueCfg("")
	}
	return table.Table{
		Name: "t",
		Rows: []table.Row{
			{Key: "r0", Size: 36, Cells: []string{"a", "b", "c"}},
			{Key: "r1", Size: 36, Cells: []string{"d", "e", "f"}},
			{Key: "r2", Size: 36, Cells: []string{"g", "h", "i"}},
		},
		Columns: []table.Column{
			{Key: "c0", Size: 80},
			{Key: "c1", Size: 80},
			{Key: "c2", Size: 80},
		},
		Cells: cells,
	}
}

var _ = Describe("Reducer", func() {
	Describe("Rename", func() {
		It("Should replace the table's name", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewRenameAction(table.RenamePayload{
					Name: "renamed",
				})),
			)
			Expect(out.Name).To(Equal("renamed"))
		})
	})

	Describe("AddRow", func() {
		It("Should append a row with the given cells to the cells map", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewAddRowAction(table.AddRowPayload{
					Index: 2,
					Key:   "r2",
					Size:  40,
					Cells: []table.Cell{
						cell("e", textCfg("E")),
						cell("f", textCfg("F")),
					},
				})),
			)
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Rows[2].Key).To(Equal("r2"))
			Expect(out.Rows[2].Cells).To(Equal([]string{"e", "f"}))
			Expect(out.Cells["e"]).To(Equal(textCfg("E")))
		})

		It("Should clamp out-of-range indices to the end of the rows slice", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewAddRowAction(table.AddRowPayload{
					Index: 999,
					Key:   "r2",
					Size:  50,
					Cells: []table.Cell{cell("z", textCfg(""))},
				})),
			)
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Rows[2].Cells).To(Equal([]string{"z"}))
		})

		It(
			"Should bootstrap one default-sized column per cell against an empty table",
			func() {
				out := MustSucceed(
					table.Reduce(
						table.Table{},
						table.NewAddRowAction(table.AddRowPayload{
							Index: 0,
							Key:   "n",
							Size:  36,
							Cells: []table.Cell{
								cell("a", textCfg("")),
								cell("b", textCfg("")),
							},
						}),
					),
				)
				Expect(out.Rows).To(HaveLen(1))
				Expect(out.Columns).To(Equal([]table.Column{
					{Key: "n-0000", Size: 72},
					{Key: "n-0001", Size: 72},
				}))
				Expect(out.Rows[0].Cells).To(Equal([]string{"a", "b"}))
			},
		)
	})

	Describe("RemoveRow", func() {
		It("Should remove the row and drop every cell it referenced", func() {
			out := MustSucceed(
				table.Reduce(
					create2x2(),
					table.NewRemoveRowAction(table.RemoveRowPayload{Key: "r0"}),
				),
			)
			Expect(out.Rows).To(HaveLen(1))
			Expect(out.Rows[0].Cells).To(Equal([]string{"c", "d"}))
			Expect(out.Cells).NotTo(HaveKey("a"))
			Expect(out.Cells).NotTo(HaveKey("b"))
		})
	})

	Describe("AddCol", func() {
		It("Should insert a column at the given index across every row", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewAddColAction(table.AddColPayload{
					Index: 1,
					Key:   "m",
					Size:  90,
					Cells: []table.Cell{
						cell("m1", textCfg("M1")),
						cell("m2", textCfg("M2")),
					},
				})),
			)
			Expect(out.Columns).To(HaveLen(3))
			Expect(out.Columns[1].Key).To(Equal("m"))
			Expect(out.Rows[0].Cells).To(Equal([]string{"a", "m1", "b"}))
			Expect(out.Rows[1].Cells).To(Equal([]string{"c", "m2", "d"}))
		})

		It(
			"Should bootstrap one default-sized row per cell against an empty table",
			func() {
				out := MustSucceed(
					table.Reduce(
						table.Table{},
						table.NewAddColAction(table.AddColPayload{
							Index: 0,
							Key:   "n",
							Size:  72,
							Cells: []table.Cell{
								cell("a", textCfg("")),
								cell("b", textCfg("")),
							},
						}),
					),
				)
				Expect(out.Columns).To(HaveLen(1))
				Expect(out.Rows).To(Equal([]table.Row{
					{Key: "n-0000", Size: 36, Cells: []string{"a"}},
					{Key: "n-0001", Size: 36, Cells: []string{"b"}},
				}))
			},
		)
	})

	Describe("RemoveCol", func() {
		It(
			"Should remove the column and drop every cell in it across every row",
			func() {
				out := MustSucceed(
					table.Reduce(
						create2x2(),
						table.NewRemoveColAction(table.RemoveColPayload{Key: "c0"}),
					),
				)
				Expect(out.Columns).To(HaveLen(1))
				Expect(out.Rows[0].Cells).To(Equal([]string{"b"}))
				Expect(out.Rows[1].Cells).To(Equal([]string{"d"}))
				Expect(out.Cells).NotTo(HaveKey("a"))
				Expect(out.Cells).NotTo(HaveKey("c"))
			},
		)
	})

	Describe("CellTemplate", func() {
		// templateKey is a 36-character UUID v4 whose last four hex digits will
		// be replaced with the per-replica index when the reducer expands it.
		const templateKey = "11111111-2222-4333-8444-555555555555"
		template := &table.Cell{Key: templateKey, Config: textCfg("")}

		DescribeTable("replicates the template across the opposing axis",
			func(state table.Table, action table.Action, wantRow0 []string) {
				out := MustSucceed(table.Reduce(state, action))
				Expect(out.Rows[0].Cells).To(Equal(wantRow0))
				Expect(out.Cells["11111111-2222-4333-8444-555555550000"]).
					To(Equal(textCfg("")))
			},
			Entry(
				"AddRow against two existing columns",
				create2x2(),
				table.NewAddRowAction(table.AddRowPayload{
					Index: 2, Key: "r2", Size: 36, CellTemplate: template,
				}),
				[]string{
					"a",
					"b",
				}, // existing row unchanged; new row's keys checked via Cells map
			),
			Entry("AddCol against two existing rows",
				create2x2(),
				table.NewAddColAction(table.AddColPayload{
					Index: 2, Key: "c2", Size: 72, CellTemplate: template,
				}),
				[]string{"a", "b", "11111111-2222-4333-8444-555555550000"},
			),
		)

		It(
			"Should seed a 1x1 grid when AddRow with a template fires against an empty table",
			func() {
				out := MustSucceed(
					table.Reduce(
						table.Table{},
						table.NewAddRowAction(table.AddRowPayload{
							Index: 0, Key: "n", Size: 36, CellTemplate: template,
						}),
					),
				)
				Expect(out.Columns).To(Equal([]table.Column{{Key: "n-0000", Size: 72}}))
				Expect(out.Rows).To(HaveLen(1))
				Expect(
					out.Rows[0].Cells,
				).To(Equal([]string{"11111111-2222-4333-8444-555555550000"}))
			},
		)

		It(
			"Should add a column with no cells when AddCol with a template fires against a row-less table",
			func() {
				out := MustSucceed(table.Reduce(table.Table{
					Columns: []table.Column{
						{Key: "c0", Size: 80},
						{Key: "c1", Size: 80},
					},
				}, table.NewAddColAction(table.AddColPayload{
					Index: 2, Key: "c2", Size: 72, CellTemplate: template,
				})))
				Expect(out.Rows).To(BeEmpty())
				Expect(out.Columns).To(HaveLen(3))
				Expect(out.Cells).To(BeEmpty())
			},
		)

		It(
			"Should add a row with no cells when AddRow with a template fires against a column-less table",
			func() {
				out := MustSucceed(table.Reduce(table.Table{
					Rows: []table.Row{{Key: "r0", Size: 36, Cells: []string{}}},
				}, table.NewAddRowAction(table.AddRowPayload{
					Index: 1, Key: "r1", Size: 36, CellTemplate: template,
				})))
				Expect(out.Rows).To(HaveLen(2))
				Expect(out.Rows[1].Cells).To(BeEmpty())
				Expect(out.Columns).To(BeEmpty())
				Expect(out.Cells).To(BeEmpty())
			},
		)
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
					Index: 2,
					Key:   "r2",
					Size:  5,
					Cells: []table.Cell{cell("z", textCfg("")), cell("y", textCfg(""))},
				}),
				func(t table.Table) float64 { return t.Rows[2].Size },
			),
			Entry("AddCol",
				table.NewAddColAction(table.AddColPayload{
					Index: 2,
					Key:   "c2",
					Size:  0,
					Cells: []table.Cell{cell("z", textCfg("")), cell("y", textCfg(""))},
				}),
				func(t table.Table) float64 { return t.Columns[2].Size },
			),
			Entry("ResizeRow",
				table.NewResizeRowAction(table.ResizeRowPayload{Key: "r0", Size: 5}),
				func(t table.Table) float64 { return t.Rows[0].Size },
			),
			Entry("ResizeCol",
				table.NewResizeColAction(table.ResizeColPayload{Key: "c0", Size: 0}),
				func(t table.Table) float64 { return t.Columns[0].Size },
			),
		)

		It("Should resize a row and a column independently", func() {
			out := MustSucceed(table.Reduce(create2x2(),
				table.NewResizeRowAction(table.ResizeRowPayload{Key: "r0", Size: 55}),
				table.NewResizeColAction(table.ResizeColPayload{Key: "c1", Size: 200}),
			))
			Expect(out.Rows[0].Size).To(Equal(55.0))
			Expect(out.Columns[1].Size).To(Equal(200.0))
		})
	})

	Describe("MoveRow", func() {
		DescribeTable("lands the row at the target index",
			func(key string, to uint32, expected [][]string) {
				out := MustSucceed(table.Reduce(
					create3x3(),
					table.NewMoveRowAction(table.MoveRowPayload{Key: key, To: to}),
				))
				Expect(out.Rows).To(HaveLen(len(expected)))
				for i, cells := range expected {
					Expect(out.Rows[i].Cells).To(Equal(cells))
				}
				Expect(out.Cells).To(HaveLen(9))
			},
			Entry("forward", "r0", uint32(2), [][]string{
				{"d", "e", "f"}, {"g", "h", "i"}, {"a", "b", "c"},
			}),
			Entry("backward", "r2", uint32(0), [][]string{
				{"g", "h", "i"}, {"a", "b", "c"}, {"d", "e", "f"},
			}),
			Entry("onto itself", "r1", uint32(1), [][]string{
				{"a", "b", "c"}, {"d", "e", "f"}, {"g", "h", "i"},
			}),
			Entry("past the end, clamped to the last row", "r0", uint32(99),
				[][]string{{"d", "e", "f"}, {"g", "h", "i"}, {"a", "b", "c"}},
			),
		)

		It("Should carry the row's key and size with it", func() {
			t := create3x3()
			t.Rows[0].Size = 50
			out := MustSucceed(table.Reduce(
				t,
				table.NewMoveRowAction(table.MoveRowPayload{Key: "r0", To: 2}),
			))
			Expect(out.Rows[2]).To(Equal(table.Row{
				Key: "r0", Size: 50, Cells: []string{"a", "b", "c"},
			}))
			Expect(out.Rows[0].Size).To(Equal(36.0))
		})
	})

	Describe("MoveCol", func() {
		DescribeTable("lands the column at the target index in every row",
			func(key string, to uint32, expected [][]string) {
				out := MustSucceed(table.Reduce(
					create3x3(),
					table.NewMoveColAction(table.MoveColPayload{Key: key, To: to}),
				))
				Expect(out.Columns).To(HaveLen(3))
				for i, cells := range expected {
					Expect(out.Rows[i].Cells).To(Equal(cells))
				}
				Expect(out.Cells).To(HaveLen(9))
			},
			Entry("forward", "c0", uint32(2), [][]string{
				{"b", "c", "a"}, {"e", "f", "d"}, {"h", "i", "g"},
			}),
			Entry("backward", "c2", uint32(0), [][]string{
				{"c", "a", "b"}, {"f", "d", "e"}, {"i", "g", "h"},
			}),
			Entry("onto itself", "c1", uint32(1), [][]string{
				{"a", "b", "c"}, {"d", "e", "f"}, {"g", "h", "i"},
			}),
			Entry("past the end, clamped to the last column", "c0", uint32(99),
				[][]string{{"b", "c", "a"}, {"e", "f", "d"}, {"h", "i", "g"}},
			),
		)

		It("Should carry the column's key and size with it", func() {
			t := create3x3()
			t.Columns[0].Size = 120
			out := MustSucceed(table.Reduce(
				t,
				table.NewMoveColAction(table.MoveColPayload{Key: "c0", To: 2}),
			))
			Expect(out.Columns[2]).To(Equal(table.Column{Key: "c0", Size: 120}))
			Expect(out.Columns[0].Size).To(Equal(80.0))
		})

		It("Should leave a row missing a cell at either index unchanged", func() {
			t := create3x3()
			t.Rows[1].Cells = []string{"d", "e"}
			out := MustSucceed(table.Reduce(
				t,
				table.NewMoveColAction(table.MoveColPayload{Key: "c0", To: 2}),
			))
			Expect(out.Rows[0].Cells).To(Equal([]string{"b", "c", "a"}))
			Expect(out.Rows[1].Cells).To(Equal([]string{"d", "e"}))
			Expect(out.Rows[2].Cells).To(Equal([]string{"h", "i", "g"}))
		})
	})

	DescribeTable("is a no-op when no row or column has the key",
		func(action table.Action) {
			Expect(table.Reduce(create3x3(), action)).To(Equal(create3x3()))
		},
		Entry("RemoveRow", table.NewRemoveRowAction(table.RemoveRowPayload{Key: "c0"})),
		Entry("RemoveCol", table.NewRemoveColAction(table.RemoveColPayload{Key: "r0"})),
		Entry("ResizeRow", table.NewResizeRowAction(table.ResizeRowPayload{
			Key: "ghost", Size: 50,
		})),
		Entry("ResizeCol", table.NewResizeColAction(table.ResizeColPayload{
			Key: "ghost", Size: 50,
		})),
		Entry("MoveRow", table.NewMoveRowAction(table.MoveRowPayload{
			Key: "ghost", To: 0,
		})),
		Entry("MoveCol", table.NewMoveColAction(table.MoveColPayload{
			Key: "ghost", To: 0,
		})),
	)

	DescribeTable("rejects an added row or column with no key",
		func(action table.Action, msg string) {
			Expect(table.Reduce(create3x3(), action)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(msg),
			))
		},
		Entry(
			"AddRow",
			table.NewAddRowAction(table.AddRowPayload{Size: 40}),
			"row key is required: validation error",
		),
		Entry(
			"AddCol",
			table.NewAddColAction(table.AddColPayload{Size: 80}),
			"column key is required: validation error",
		),
	)

	Describe("SetCell", func() {
		It("Should replace the cell stored under the given key", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewSetCellAction(table.SetCellPayload{
					Cell: cell("a", valueCfg("psi")),
				})),
			)
			Expect(out.Cells["a"]).To(Equal(valueCfg("psi")))
			Expect(out.Cells).To(HaveLen(4))
		})

		It("Should be a no-op when no cell with the given key exists", func() {
			out := MustSucceed(
				table.Reduce(create2x2(), table.NewSetCellAction(table.SetCellPayload{
					Cell: cell("ghost", textCfg("")),
				})),
			)
			Expect(out.Cells).NotTo(HaveKey("ghost"))
			Expect(out.Cells).To(HaveLen(4))
		})
	})

	Describe("EraseCells", func() {
		template := textCfg("")

		DescribeTable(
			"removes fully-selected axes and resets surviving cells",
			func(selection []string, wantRows, wantCols int, wantRow0 []string) {
				out := MustSucceed(
					table.Reduce(
						create3x3(),
						table.NewEraseCellsAction(table.EraseCellsPayload{
							Cells: selection, Template: template,
						}),
					),
				)
				Expect(out.Rows).To(HaveLen(wantRows))
				Expect(out.Columns).To(HaveLen(wantCols))
				if len(out.Rows) > 0 {
					Expect(out.Rows[0].Cells).To(Equal(wantRow0))
				}
			},
			Entry(
				"partial selection: surviving cells reset to template, axes untouched",
				[]string{"b", "e"},
				3,
				3,
				[]string{"a", "b", "c"},
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
			out := MustSucceed(
				table.Reduce(
					create3x3(),
					table.NewEraseCellsAction(table.EraseCellsPayload{
						Cells: []string{"b", "e"}, Template: template,
					}),
				),
			)
			Expect(out.Cells["b"]).To(Equal(textCfg("")))
			Expect(out.Cells["e"]).To(Equal(textCfg("")))
			Expect(out.Cells["a"]).To(Equal(valueCfg("")))
		})

		It(
			"Should drop cells from the cells map when their row is fully removed",
			func() {
				out := MustSucceed(
					table.Reduce(
						create3x3(),
						table.NewEraseCellsAction(table.EraseCellsPayload{
							Cells: []string{"d", "e", "f"}, Template: template,
						}),
					),
				)
				for _, k := range []string{"d", "e", "f"} {
					Expect(out.Cells).NotTo(HaveKey(k))
				}
			},
		)

		It("Should be a no-op for an empty selection", func() {
			out := MustSucceed(
				table.Reduce(
					create3x3(),
					table.NewEraseCellsAction(table.EraseCellsPayload{
						Cells: []string{}, Template: template,
					}),
				),
			)
			Expect(out.Rows).To(HaveLen(3))
			Expect(out.Cells).To(HaveLen(9))
		})
	})
})

var _ = Describe("Path actions", func() {
	set := func(path []string, value any, guards ...patch.Guard) table.Action {
		return table.NewSetAction(
			table.SetPayload{Path: path, Value: value, Guards: guards},
		)
	}

	Describe("SetPayload", func() {
		It("Should set a field of a cell config", func() {
			next := MustSucceed(table.Reduce(
				create2x2(),
				set([]string{"cells", "a", "value"}, "Pressure"),
			))
			Expect(asText(next.Cells["a"]).Value).To(Equal("Pressure"))
			Expect(next.Rows).To(Equal(create2x2().Rows))
		})

		It("Should do nothing when the variant guard fails", func() {
			next := MustSucceed(table.Reduce(create2x2(), set(
				[]string{"cells", "a", "value"},
				"Pressure",
				patch.Guard{Path: []string{"cells", "a", "variant"}, Value: "value"},
			)))
			Expect(asText(next.Cells["a"]).Value).To(BeEmpty())
		})

		It("Should reject a path outside the cells", func() {
			Expect(table.Reduce(create2x2(), set([]string{"name"}, "Other"))).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("the path is outside")),
			))
		})

		It("Should reject a value that fails the cell config's schema", func() {
			Expect(table.Reduce(create2x2(), set([]string{"cells", "a", "weight"}, 0))).
				Error().To(MatchError(ContainSubstring(
				"cells.a.weight: must be greater than or equal to 1",
			)))
		})

		It("Should edit a cell beside a stored cell that fails its schema", func() {
			state := create2x2()
			state.Cells["b"] = table.CellConfig{Variant: table.TextCellConfig{}}
			next := MustSucceed(table.Reduce(
				state,
				set([]string{"cells", "a", "value"}, "Pressure"),
			))
			Expect(asText(next.Cells["a"]).Value).To(Equal("Pressure"))
		})

		It("Should reject a cell config that names no variant", func() {
			Expect(table.Reduce(create2x2(), set([]string{"cells", "a"}, nil))).
				Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`config for cell "a" names no variant`)),
			))
		})
	})

	Describe("RemovePayload", func() {
		It("Should remove a field from a cell config", func() {
			state := create2x2()
			text := asText(textCfg(""))
			text.FillColor = new(MustSucceed(color.FromHex("#ff0000")))
			state.Cells["a"] = table.CellConfig{Variant: text}
			next := MustSucceed(table.Reduce(state, table.NewRemoveAction(
				table.RemovePayload{Path: []string{"cells", "a"}, Key: "fill_color"},
			)))
			Expect(asText(next.Cells["a"]).FillColor).To(BeNil())
		})
	})

	Describe("InsertPayload", func() {
		It("Should reject a list the cell config does not declare", func() {
			Expect(table.Reduce(create2x2(), table.NewInsertAction(table.InsertPayload{
				Path:  []string{"cells", "a", "tags"},
				Value: map[string]any{"key": "x"},
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding entries")),
			))
		})
	})

	Describe("MovePayload", func() {
		It("Should leave the table unchanged when the list is missing", func() {
			next := MustSucceed(table.Reduce(create2x2(), table.NewMoveAction(
				table.MovePayload{Path: []string{"cells", "a", "tags"}, Key: "x"},
			)))
			Expect(next).To(Equal(create2x2()))
		})
	})
})

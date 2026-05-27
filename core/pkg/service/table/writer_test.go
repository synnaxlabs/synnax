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
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

type scopedAction = actions.Scoped[table.Key, table.Action]

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Table", func(ctx SpecContext) {
			t := table.Table{
				Name:    "test",
				Rows:    []table.Row{{Size: 30, Cells: []string{"a"}}},
				Columns: []table.Column{{Size: 80}},
				Cells: map[string]table.Cell{
					"a": {Key: "a", Variant: "text", Props: msgpack.EncodedJSON{"value": "hello"}},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &t)).To(Succeed())
			Expect(t.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should create a Table without a workspace", func(ctx SpecContext) {
			t := table.Table{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, uuid.Nil, &t)).To(Succeed())
			Expect(t.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a Table", func(ctx SpecContext) {
			s := table.Table{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, s.Key, "test2")).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should replace the body of a Table while preserving key and name", func(ctx SpecContext) {
			s := table.Table{
				Name:    "test",
				Rows:    []table.Row{{Size: 30, Cells: []string{"a"}}},
				Columns: []table.Column{{Size: 80}},
				Cells: map[string]table.Cell{
					"a": {Key: "a", Variant: "text", Props: msgpack.EncodedJSON{"value": "v1"}},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			updated := table.Table{
				Rows:    []table.Row{{Size: 40, Cells: []string{"a", "b"}}},
				Columns: []table.Column{{Size: 100}, {Size: 120}},
				Cells: map[string]table.Cell{
					"a": {Key: "a", Variant: "text", Props: msgpack.EncodedJSON{"value": "v2"}},
					"b": {Key: "b", Variant: "value", Props: msgpack.EncodedJSON{"units": "psi"}},
				},
			}
			Expect(svc.NewWriter(tx).SetData(ctx, s.Key, updated)).To(Succeed())
			var got table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&got).Exec(ctx, tx)).To(Succeed())
			Expect(got.Key).To(Equal(s.Key))
			Expect(got.Name).To(Equal("test"))
			Expect(got.Rows).To(HaveLen(1))
			Expect(got.Rows[0].Size).To(Equal(40.0))
			Expect(got.Rows[0].Cells).To(Equal([]string{"a", "b"}))
			Expect(got.Columns).To(HaveLen(2))
			Expect(got.Cells).To(HaveLen(2))
			Expect(got.Cells["a"].Props["value"]).To(Equal("v2"))
			Expect(got.Cells["b"].Variant).To(Equal("value"))
		})
	})
	Describe("Dispatch", func() {
		seed := func(ctx SpecContext) table.Table {
			s := table.Table{
				Name:    "test",
				Rows:    []table.Row{{Size: 30, Cells: []string{"a", "b"}}},
				Columns: []table.Column{{Size: 80}, {Size: 100}},
				Cells: map[string]table.Cell{
					"a": {Key: "a", Variant: "text", Props: msgpack.EncodedJSON{"value": "A"}},
					"b": {Key: "b", Variant: "text", Props: msgpack.EncodedJSON{"value": "B"}},
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			return s
		}

		It("Should apply Rename", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "renamed"}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("renamed"))
		})

		It("Should apply AddRow with cells appended to the cells map", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddRowAction(table.AddRowPayload{
					Index: 1,
					Size:  40,
					Cells: []table.Cell{
						{Key: "c", Variant: "text", Props: msgpack.EncodedJSON{"value": "C"}},
						{Key: "d", Variant: "text", Props: msgpack.EncodedJSON{"value": "D"}},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(HaveLen(2))
			Expect(res.Rows[1].Cells).To(Equal([]string{"c", "d"}))
			Expect(res.Cells).To(HaveLen(4))
			Expect(res.Cells["c"].Props["value"]).To(Equal("C"))
		})

		It("Should clamp out-of-range AddRow indices to the end", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddRowAction(table.AddRowPayload{
					Index: 999,
					Size:  50,
					Cells: []table.Cell{{Key: "z", Variant: "text"}},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(HaveLen(2))
			Expect(res.Rows[1].Cells).To(Equal([]string{"z"}))
		})

		It("Should remove the row and drop its cells on RemoveRow", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRemoveRowAction(table.RemoveRowPayload{Index: 0}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(BeEmpty())
			Expect(res.Cells).To(BeEmpty())
		})

		It("Should be a no-op for RemoveRow with an out-of-range index", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRemoveRowAction(table.RemoveRowPayload{Index: 99}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(HaveLen(1))
			Expect(res.Cells).To(HaveLen(2))
		})

		It("Should insert a column at the given index across every row", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddColAction(table.AddColPayload{
					Index: 1,
					Size:  90,
					Cells: []table.Cell{
						{Key: "mid-1", Variant: "text", Props: msgpack.EncodedJSON{"value": "M1"}},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns).To(HaveLen(3))
			Expect(res.Rows[0].Cells).To(Equal([]string{"a", "mid-1", "b"}))
			Expect(res.Cells["mid-1"].Variant).To(Equal("text"))
		})

		It("Should remove the column and drop every cell in that column", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRemoveColAction(table.RemoveColPayload{Index: 0}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns).To(HaveLen(1))
			Expect(res.Rows[0].Cells).To(Equal([]string{"b"}))
			Expect(res.Cells).To(HaveKey("b"))
			Expect(res.Cells).NotTo(HaveKey("a"))
		})

		It("Should resize a row and a column independently", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewResizeRowAction(table.ResizeRowPayload{Index: 0, Size: 55}),
				table.NewResizeColAction(table.ResizeColPayload{Index: 1, Size: 200}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows[0].Size).To(Equal(55.0))
			Expect(res.Columns[1].Size).To(Equal(200.0))
		})

		It("Should replicate AddRow's CellTemplate across existing columns", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddRowAction(table.AddRowPayload{
					Index: 1,
					Size:  36,
					CellTemplate: table.Cell{
						Key:     "11111111-2222-4333-8444-555555555555",
						Variant: "text",
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(HaveLen(2))
			// Two existing columns -> two derived replicas with hex-suffixed keys.
			Expect(res.Rows[1].Cells).To(Equal([]string{
				"11111111-2222-4333-8444-555555550000",
				"11111111-2222-4333-8444-555555550001",
			}))
			Expect(res.Cells["11111111-2222-4333-8444-555555550000"].Variant).To(Equal("text"))
		})

		It("Should replicate AddCol's CellTemplate across existing rows", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddColAction(table.AddColPayload{
					Index: 2,
					Size:  72,
					CellTemplate: table.Cell{
						Key:     "11111111-2222-4333-8444-555555555555",
						Variant: "text",
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns).To(HaveLen(3))
			Expect(res.Rows[0].Cells).To(Equal([]string{
				"a", "b", "11111111-2222-4333-8444-555555550000",
			}))
		})

		It("Should bootstrap columns when AddRow fires against an empty table", func(ctx SpecContext) {
			t := table.Table{Name: "empty"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &t)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, t.Key, "dk-1", []table.Action{
				table.NewAddRowAction(table.AddRowPayload{
					Index: 0,
					Size:  36,
					Cells: []table.Cell{
						{Key: "a", Variant: "text"},
						{Key: "b", Variant: "text"},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(t.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows).To(HaveLen(1))
			Expect(res.Columns).To(HaveLen(2))
			Expect(res.Columns[0].Size).To(Equal(72.0))
			Expect(res.Rows[0].Cells).To(Equal([]string{"a", "b"}))
		})

		It("Should bootstrap rows when AddCol fires against an empty table", func(ctx SpecContext) {
			t := table.Table{Name: "empty"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &t)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, t.Key, "dk-1", []table.Action{
				table.NewAddColAction(table.AddColPayload{
					Index: 0,
					Size:  72,
					Cells: []table.Cell{
						{Key: "a", Variant: "text"},
						{Key: "b", Variant: "text"},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(t.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns).To(HaveLen(1))
			Expect(res.Rows).To(HaveLen(2))
			Expect(res.Rows[0].Size).To(Equal(36.0))
			Expect(res.Rows[0].Cells).To(Equal([]string{"a"}))
			Expect(res.Rows[1].Cells).To(Equal([]string{"b"}))
		})

		It("Should clamp AddRow sizes below the minimum cell dimension", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddRowAction(table.AddRowPayload{
					Index: 1,
					Size:  5,
					Cells: []table.Cell{{Key: "z", Variant: "text"}},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows[1].Size).To(Equal(32.0))
		})

		It("Should clamp AddCol sizes below the minimum cell dimension", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewAddColAction(table.AddColPayload{
					Index: 2,
					Size:  0,
					Cells: []table.Cell{{Key: "z", Variant: "text"}},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns[2].Size).To(Equal(32.0))
		})

		It("Should clamp ResizeRow sizes below the minimum cell dimension", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewResizeRowAction(table.ResizeRowPayload{Index: 0, Size: 5}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Rows[0].Size).To(Equal(32.0))
		})

		It("Should clamp ResizeCol sizes below the minimum cell dimension", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewResizeColAction(table.ResizeColPayload{Index: 0, Size: 0}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Columns[0].Size).To(Equal(32.0))
		})

		It("Should replace a cell on SetCell while preserving the table's key", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewSetCellAction(table.SetCellPayload{
					Cell: table.Cell{
						Key:     "a",
						Variant: "value",
						Props:   msgpack.EncodedJSON{"units": "psi"},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Cells["a"].Variant).To(Equal("value"))
			Expect(res.Cells["a"].Props["units"]).To(Equal("psi"))
			Expect(res.Cells).To(HaveLen(2))
		})

		It("Should be a no-op for SetCell targeting an unknown key", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewSetCellAction(table.SetCellPayload{
					Cell: table.Cell{Key: "ghost", Variant: "text"},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Cells).NotTo(HaveKey("ghost"))
			Expect(res.Cells).To(HaveLen(2))
		})

		It("Should apply a multi-action sequence atomically", func(ctx SpecContext) {
			s := seed(ctx)
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "multi"}),
				table.NewAddRowAction(table.AddRowPayload{
					Index: 1,
					Size:  40,
					Cells: []table.Cell{
						{Key: "c", Variant: "text"},
						{Key: "d", Variant: "text"},
					},
				}),
				table.NewSetCellAction(table.SetCellPayload{
					Cell: table.Cell{
						Key:     "c",
						Variant: "value",
						Props:   msgpack.EncodedJSON{"telem": "ch1"},
					},
				}),
			})).To(Succeed())
			var res table.Table
			Expect(svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("multi"))
			Expect(res.Rows).To(HaveLen(2))
			Expect(res.Cells["c"].Variant).To(Equal("value"))
			Expect(res.Cells["c"].Props["telem"]).To(Equal("ch1"))
		})

		It("Should notify the action observer once per Dispatch with monotonic seq", func(ctx SpecContext) {
			s := seed(ctx)
			var received []scopedAction
			disconnect := svc.OnAction(func(_ context.Context, sa scopedAction) {
				received = append(received, sa)
			})
			defer disconnect()
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "first"}),
			})).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-2", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "second"}),
			})).To(Succeed())
			Expect(received).To(HaveLen(2))
			Expect(received[0].DispatchKey).To(Equal("dk-1"))
			Expect(received[1].DispatchKey).To(Equal("dk-2"))
			Expect(received[1].Seq > received[0].Seq).To(BeTrue())
		})
	})
})

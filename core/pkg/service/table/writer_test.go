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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Writer", func() {
	// seed creates a 2x2 table named "test" with cells a,b across row 0 and
	// no second row populated; only the writer's persistence path is
	// exercised. Reducer behavior is covered in actions_test.go.
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

	retrieve := func(ctx SpecContext, key table.Key) table.Table {
		var res table.Table
		Expect(svc.NewRetrieve().Where(table.MatchKeys(key)).Entry(&res).Exec(ctx, tx)).To(Succeed())
		return res
	}

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

	Describe("Dispatch", func() {
		It("Should rename a Table via a Rename action", func(ctx SpecContext) {
			s := table.Table{Name: "test"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "test2"}),
			})).To(Succeed())
			Expect(retrieve(ctx, s.Key).Name).To(Equal("test2"))
		})

		It("Should apply a multi-action sequence atomically and persist the result", func(ctx SpecContext) {
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
			res := retrieve(ctx, s.Key)
			Expect(res.Name).To(Equal("multi"))
			Expect(res.Rows).To(HaveLen(2))
			Expect(res.Cells["c"].Variant).To(Equal("value"))
			Expect(res.Cells["c"].Props["telem"]).To(Equal("ch1"))
		})

		It("Should notify the action observer once per Dispatch with monotonic seq", func(ctx SpecContext) {
			s := seed(ctx)
			rec := &Recorder[table.Key, table.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-1", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "first"}),
			})).To(Succeed())
			Expect(svc.NewWriter(tx).Dispatch(ctx, s.Key, "dk-2", []table.Action{
				table.NewRenameAction(table.RenamePayload{Name: "second"}),
			})).To(Succeed())
			received := rec.Snapshot()
			Expect(received).To(HaveLen(2))
			Expect(received[0].DispatchKey).To(Equal("dk-1"))
			Expect(received[1].DispatchKey).To(Equal("dk-2"))
			Expect(received[1].Seq > received[0].Seq).To(BeTrue())
		})
	})
})

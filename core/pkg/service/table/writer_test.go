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
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

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
})

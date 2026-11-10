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
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
)

type cellsUpdate = patch.TypedMap[string, table.CellConfig, table.CellConfigUpdate]

var _ = Describe("Update", func() {
	var t table.Table
	BeforeEach(func() { t = create2x2() })
	update := func(cells cellsUpdate) (table.Table, error) {
		return table.Reduce(t, table.NewUpdateAction(
			table.UpdatePayload{Value: table.Update{Cells: &cells}},
		))
	}

	It("Should merge an update into the cell under its key", func() {
		next := MustSucceed(update(cellsUpdate{
			Update: map[string]table.CellConfigUpdate{
				"a": {Variant: table.TextCellConfigUpdate{Value: new("Pressure")}},
			},
		}))
		Expect(next.Cells["a"]).To(Equal(textCfg("Pressure")))
		Expect(next.Cells["b"]).To(Equal(textCfg("")))
		Expect(t.Cells["a"]).To(Equal(textCfg("")))
	})

	It("Should replace a cell with the defaults of another variant", func() {
		next := MustSucceed(update(cellsUpdate{
			Update: map[string]table.CellConfigUpdate{
				"a": {Variant: table.ValueCellConfigUpdate{Units: new("psi")}},
			},
		}))
		Expect(next.Cells["a"]).To(Equal(valueCfg("psi")))
	})

	It("Should insert and remove cells by key", func() {
		next := MustSucceed(update(cellsUpdate{
			Insert: map[string]table.CellConfig{"e": textCfg("New")},
			Remove: []string{"d"},
		}))
		Expect(next.Cells).To(HaveLen(4))
		Expect(next.Cells).To(HaveKeyWithValue("e", textCfg("New")))
		Expect(next.Cells).ToNot(HaveKey("d"))
	})

	It("Should leave the table alone when a cell is missing", func() {
		next, err := update(cellsUpdate{
			Update: map[string]table.CellConfigUpdate{
				"z": {Variant: table.TextCellConfigUpdate{Value: new("Pressure")}},
			},
		})
		Expect(err).To(And(
			MatchError(patch.ErrStale),
			MatchError(ContainSubstring("z not found")),
		))
		Expect(next).To(Equal(t))
	})
})

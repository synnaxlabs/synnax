// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateTable", func() {
	It("Should key every row and column by its position", func(ctx SpecContext) {
		key := uuid.New()
		cells := map[string]v2.CellConfig{
			"a": {Variant: v2.TextCellConfig{Value: "A"}},
		}
		out := MustSucceed(v3.MigrateTable(ctx, v2.Table{
			Key:  key,
			Name: "t",
			Rows: []v2.Row{
				{Size: 36, Cells: []string{"a", "b"}},
				{Size: 40, Cells: []string{"c", "d"}},
			},
			Columns: []v2.Column{{Size: 72}, {Size: 100}},
			Cells:   cells,
		}))
		Expect(out.Key).To(Equal(key))
		Expect(out.Name).To(Equal("t"))
		Expect(out.Rows).To(Equal([]v3.Row{
			{Key: "row-0", Size: 36, Cells: []string{"a", "b"}},
			{Key: "row-1", Size: 40, Cells: []string{"c", "d"}},
		}))
		Expect(out.Columns).To(Equal([]v3.Column{
			{Key: "col-0", Size: 72},
			{Key: "col-1", Size: 100},
		}))
		Expect(out.Cells).To(Equal(cells))
	})
})

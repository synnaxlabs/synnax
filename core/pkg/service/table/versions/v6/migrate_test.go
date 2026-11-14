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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v6"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv1 "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should store every cell color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v5.Key, v5.Table]{DB: db}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := v5.Table{Key: uuid.New(), Name: "t", Cells: map[string]v2.CellConfig{
				"text": {Variant: v2.TextCellConfig{FillColor: c(1), TextColor: c(2)}},
				"value": {Variant: v2.ValueCellConfig{
					FillColor:      c(3),
					TextColor:      c(4),
					StalenessColor: c(5),
					Redline: colorv1.Scale{Bands: []colorv1.Band{
						{Key: "b", Color: *c(6)},
					}},
				}},
			}}
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
			text := got.Cells["text"].Variant.(v6.TextCellConfig)
			Expect(text.FillColor).To(Equal(n(1)))
			Expect(text.TextColor).To(Equal(n(2)))
			value := got.Cells["value"].Variant.(v6.ValueCellConfig)
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
	It("Should store the cell color alpha as a byte", func(ctx SpecContext) {
		out := MustSucceed(
			v6.MigrateTable(ctx, v5.Table{Cells: map[string]v2.CellConfig{
				"a": {
					Variant: v2.TextCellConfig{FillColor: &color.Color{R: 1, A: 0.5}},
				},
			}}),
		)
		Expect(out.Cells["a"].Variant.(v6.TextCellConfig).FillColor).
			To(Equal(&colorv2.Color{R: 1, A: 128}))
	})
})

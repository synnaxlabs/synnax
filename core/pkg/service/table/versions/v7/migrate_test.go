// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v6 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v7"
	color "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should store each cell color of a table that v0.62 stored as a fixed color",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v6.Key, v6.Table]{DB: db}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: 128} }
			old := v6.Table{
				Key:  uuid.New(),
				Name: "t",
				Rows: []v6.Row{{Key: "r", Cells: []v6.Cell{
					{Key: "text", Config: v6.CellConfig{Variant: v6.TextCellConfig{
						Value: "Pressure", FillColor: c(1),
					}}},
					{Key: "value", Config: v6.CellConfig{Variant: v6.ValueCellConfig{
						FillColor:      c(3),
						TextColor:      c(4),
						StalenessColor: c(5),
						Redline: color.Scale{Bands: []color.Band{
							{Key: "b", Color: *c(6)},
						}},
					}}},
				}}},
				Columns: []v6.Column{{Key: "c0"}, {Key: "c1"}},
			}
			Expect(gorp.NewCreate[v6.Key, v6.Table]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v7.Key, v7.Table]{
				DB: db, Migrations: []gorp.Migration{v7.Migration},
			}))
			var got v7.Table
			Expect(gorp.NewRetrieve[v7.Key, v7.Table]().
				Where(gorp.MatchKeys[v7.Key, v7.Table](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			fixed := func(r uint8) *servicecolor.Ref {
				return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: *c(r)}}
			}
			cells := got.Rows[0].Cells
			Expect(cells).To(HaveLen(2))
			Expect(cells[0].Key).To(Equal("text"))
			text, ok := cells[0].Config.Variant.(v7.TextCellConfig)
			Expect(ok).To(BeTrue())
			Expect(text.Value).To(Equal("Pressure"))
			Expect(text.FillColor).To(Equal(fixed(1)))
			Expect(text.TextColor).To(BeNil())
			Expect(cells[1].Key).To(Equal("value"))
			value, ok := cells[1].Config.Variant.(v7.ValueCellConfig)
			Expect(ok).To(BeTrue())
			Expect(value.FillColor).To(Equal(fixed(3)))
			Expect(value.TextColor).To(Equal(fixed(4)))
			Expect(value.StalenessColor).To(Equal(fixed(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]color.Band{{Key: "b", Color: *c(6)}}))
		},
	)
})

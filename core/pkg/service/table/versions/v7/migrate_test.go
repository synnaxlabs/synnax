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
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v6 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v7"
	color "github.com/synnaxlabs/x/color/versions/v2"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MigrateCellConfig", func() {
	c := color.Color{R: 1, G: 2, A: 128}
	fixed := &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: c}}
	DescribeTable("Should store each cell color as a fixed color",
		func(ctx SpecContext, old v6.CellConfig, expected v7.CellConfig) {
			Expect(v7.MigrateCellConfig(ctx, old)).To(Equal(expected))
		},
		Entry("text with colors",
			v6.CellConfig{Variant: v6.TextCellConfig{
				Value: "Pressure", FillColor: &c, TextColor: &c,
			}},
			v7.CellConfig{Variant: v7.TextCellConfig{
				Value: "Pressure", FillColor: fixed, TextColor: fixed,
			}},
		),
		Entry("value with colors",
			v6.CellConfig{Variant: v6.ValueCellConfig{
				Channel:          1,
				FillColor:        &c,
				TextColor:        &c,
				StalenessColor:   &c,
				StalenessTimeout: 5,
			}},
			v7.CellConfig{Variant: v7.ValueCellConfig{
				Channel:          1,
				FillColor:        fixed,
				TextColor:        fixed,
				StalenessColor:   fixed,
				StalenessTimeout: 5 * telem.Second,
			}},
		),
		Entry("text with absent colors",
			v6.CellConfig{Variant: v6.TextCellConfig{Value: "Pressure"}},
			v7.CellConfig{Variant: v7.TextCellConfig{Value: "Pressure"}},
		),
		Entry("value with absent colors",
			v6.CellConfig{Variant: v6.ValueCellConfig{Channel: 1, StalenessTimeout: 5}},
			v7.CellConfig{Variant: v7.ValueCellConfig{
				Channel: 1, StalenessTimeout: 5 * telem.Second,
			}},
		),
	)
	DescribeTable("Should store the staleness timeout in seconds as a span",
		func(ctx SpecContext, seconds float64, expected telem.TimeSpan) {
			c := MustSucceed(v7.MigrateCellConfig(ctx, v6.CellConfig{
				Variant: v6.ValueCellConfig{StalenessTimeout: seconds},
			}))
			v, ok := c.Variant.(v7.ValueCellConfig)
			Expect(ok).To(BeTrue())
			Expect(v.StalenessTimeout).To(Equal(expected))
		},
		Entry("whole", 5.0, 5*telem.Second),
		Entry("fraction", 1.5, 1500*telem.Millisecond),
		Entry("sub-nanosecond fraction", 2.0000000004, 2*telem.Second),
		Entry("one second", 1.0, telem.Second),
		Entry("below a second", 0.5, telem.Second),
		Entry("zero", 0.0, 5*telem.Second),
		Entry("negative", -2.0, 5*telem.Second),
		Entry("NaN", math.NaN(), 5*telem.Second),
		Entry("too large", 1e12, telem.TimeSpan(1<<53-1)),
		Entry("infinite", math.Inf(1), telem.TimeSpan(1<<53-1)),
	)
})

var _ = Describe("Migration", func() {
	It("Should store the colors and staleness timeout of a table that v0.62 stored",
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
						FillColor:        c(3),
						TextColor:        c(4),
						StalenessColor:   c(5),
						StalenessTimeout: 2.5,
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
			Expect(value.StalenessTimeout).To(Equal(2500 * telem.Millisecond))
			Expect(
				value.Redline.Bands,
			).To(Equal([]color.Band{{Key: "b", Color: *c(6)}}))
		},
	)
})

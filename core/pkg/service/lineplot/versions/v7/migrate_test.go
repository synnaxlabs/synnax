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
	"context"
	"embed"
	"encoding/hex"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var fixtures embed.FS

// truncatedErr matches the reader errors a truncated Orc buffer produces.
var truncatedErr = SatisfyAny(MatchError(io.EOF), MatchError(io.ErrUnexpectedEOF))

func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(
		strings.ReplaceAll(string(raw), "\n", ""),
	))
}

func decodeV6(ctx context.Context) v6.LinePlot {
	GinkgoHelper()
	var lp v6.LinePlot
	Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &lp)).To(Succeed())
	return lp
}

var _ = Describe("MigrateLinePlot", func() {
	It("Should carry every field but the line reduction settings",
		func(ctx SpecContext) {
			in := decodeV6(ctx)
			out := MustSucceed(v7.MigrateLinePlot(ctx, in))
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.Name).To(Equal(in.Name))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges).To(Equal(in.Ranges))
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Rules).To(Equal(in.Rules))
			Expect(out.Lines).To(HaveLen(len(in.Lines)))
			for i, l := range out.Lines {
				Expect(l.Key).To(Equal(in.Lines[i].Key))
				Expect(l.Label).To(Equal(in.Lines[i].Label))
				Expect(l.Color).To(Equal(in.Lines[i].Color))
				Expect(l.StrokeWidth).To(Equal(in.Lines[i].StrokeWidth))
			}
		})

	It("Should round-trip the migrated plot through the v7 codec",
		func(ctx SpecContext) {
			out := MustSucceed(v7.MigrateLinePlot(ctx, decodeV6(ctx)))
			encoded := MustSucceed(orc.Codec.Encode(ctx, out))
			var decoded v7.LinePlot
			Expect(orc.Codec.Decode(ctx, encoded, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(out))
		})
})

var _ = Describe("MigrateLine", func() {
	DescribeTable("Should map the downsample mode to an aggregation",
		func(ctx SpecContext, old v6.Line, agg v7.Aggregation) {
			out := MustSucceed(v7.MigrateLine(ctx, old))
			Expect(out.Aggregation).To(Equal(agg))
			Expect(out.Detail).To(Equal(v7.DetailMedium))
		},
		Entry("the zero line", v6.Line{}, v7.AggregationMinMax),
		Entry("a decimated line", v6.Line{
			Downsample:     1,
			DownsampleMode: v6.DownsampleModeDecimate,
		}, v7.AggregationMinMax),
		Entry("an averaged line", v6.Line{
			Downsample:     8,
			DownsampleMode: v6.DownsampleModeAverage,
		}, v7.AggregationAverage),
	)

	It("Should keep the line's style", func(ctx SpecContext) {
		label := "pressure"
		out := MustSucceed(v7.MigrateLine(ctx, v6.Line{
			Key:         "k",
			Label:       &label,
			StrokeWidth: 3,
			Downsample:  4,
		}))
		Expect(out.Key).To(Equal("k"))
		Expect(out.Label).To(HaveValue(Equal("pressure")))
		Expect(out.StrokeWidth).To(Equal(3.0))
	})
})

var _ = Describe("Storage migration", func() {
	It("Should lift a stored v6 entry to v7 through the gorp migration",
		func(ctx SpecContext) {
			stored := decodeV6(ctx)
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[v6.Key, v6.LinePlot]{DB: db},
			))
			Expect(gorp.NewCreate[v6.Key, v6.LinePlot]().
				Entry(&stored).Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: []migrate.Migration{v7.Migration},
			})).To(Succeed())
			var got v7.LinePlot
			Expect(gorp.NewRetrieve[v7.Key, v7.LinePlot]().
				Where(gorp.MatchKeys[v7.Key, v7.LinePlot](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(MustSucceed(v7.MigrateLinePlot(ctx, stored))))
		})
})

var _ = Describe("v7 wire format", func() {
	It("Should decode the pinned v7 payload", func(ctx SpecContext) {
		var lp v7.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v7_initial.hex"), &lp)).To(Succeed())
		Expect(lp.Lines).To(HaveLen(2))
		Expect(lp.Lines[0].Aggregation).To(Equal(v7.AggregationAverage))
		Expect(lp.Lines[0].Detail).To(Equal(v7.DetailHigh))
		Expect(lp.Lines[1].Aggregation).To(Equal(v7.AggregationMinMax))
		Expect(lp.Lines[1].Detail).To(Equal(v7.DetailLow))
	})

	It("Should re-encode the payload byte for byte", func(ctx SpecContext) {
		raw := loadWire("v7_initial.hex")
		var lp v7.LinePlot
		Expect(orc.Codec.Decode(ctx, raw, &lp)).To(Succeed())
		Expect(orc.Codec.Encode(ctx, lp)).To(Equal(raw))
	})

	It("Should reject every truncation of a stored payload", func(ctx SpecContext) {
		raw := loadWire("v7_initial.hex")
		for n := 3; n < len(raw); n++ {
			var lp v7.LinePlot
			Expect(orc.Codec.Decode(ctx, raw[:n], &lp)).
				To(truncatedErr, "prefix length %d", n)
		}
	})
})

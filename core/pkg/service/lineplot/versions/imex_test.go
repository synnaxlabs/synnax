// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/text"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.LinePlot {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		lp := decode(ctx, "testdata/import_v5.json")
		Expect(lp.Channels.Y1).To(Equal([]channel.Key{1, 2}))
		Expect(lp.Ranges.X1).To(Equal([]string{"recent"}))
		Expect(lp.Title.Level).To(Equal(text.Level("h4")))
		Expect(lp.Lines).To(HaveLen(1))
		Expect(lp.Lines[0].StrokeWidth).To(Equal(2.5))
		Expect(
			lp.Lines[0].DownsampleMode,
		).To(Equal(versions.DownsampleMode("decimate")))
	})

	It("Should lift a Console state through the legacy chain", func(ctx SpecContext) {
		lp := decode(ctx, "testdata/import_v0_state.json")
		Expect(lp.Channels.Y1).To(Equal([]channel.Key{9}))
		Expect(lp.Axes.Y1.Label).To(Equal("pressure"))
		// autoBounds inverts into ManualBounds: an auto bound is not a manual one.
		Expect(lp.Axes.Y1.ManualBounds).To(
			Equal(versions.ManualBounds{Lower: true, Upper: false}),
		)
		Expect(lp.Rules).To(HaveLen(1))
		Expect(lp.Rules[0].Label).To(Equal("max"))
		Expect(lp.Rules[0].Position).To(Equal(42.0))
	})

	// The fixtures below are line plots a shipped Console exported.
	It("Should lift a version 3 Console export", func(ctx SpecContext) {
		lp := decode(ctx, "testdata/import_console_v3.json")
		Expect(lp.Channels.Y1).To(Equal([]channel.Key{1048586, 1048587}))
		Expect(lp.Ranges.X1).To(Equal([]string{"recent"}))
		Expect(lp.Lines).To(HaveLen(2))
		Expect(lp.Lines[0].Label).To(HaveValue(Equal("stream_write_data_1")))
		Expect(lp.Lines[0].StrokeWidth).To(Equal(2.0))
		// A v3 export nests the six axes under axes.axes; the flat shape came later.
		Expect(lp.Axes.Y1.TickSpacing).To(Equal(75.0))
		// autoBounds inverts: a bound the Console computed is not a manual one.
		Expect(lp.Axes.Y1.ManualBounds).To(Equal(versions.ManualBounds{}))
		Expect(lp.Legend.Hidden).To(BeFalse())
	})

	It("Should ignore the keys a Console export carries beyond the schema", func(
		ctx SpecContext,
	) {
		// The file holds plots, mode, toolbar, selection, and control, none of which
		// survive into the stored shape.
		lp := decode(ctx, "testdata/import_console_v2_extra_keys.json")
		Expect(lp.Title.Level).To(Equal(text.Level("h4")))
		Expect(lp.Channels.Y1).To(BeEmpty())
		Expect(lp.Lines).To(BeEmpty())
	})

	It("Should lift the Console state that added the sticky legend", func(
		ctx SpecContext,
	) {
		lp := decode(ctx, "testdata/import_v1_state.json")
		Expect(lp.Channels.X1).To(Equal(channel.Key(65540)))
		Expect(lp.Channels.Y1).To(Equal([]channel.Key{65541, 65542}))
		Expect(lp.Ranges.X1).To(Equal([]string{"recent"}))
		Expect(lp.Axes.Y1.Label).To(Equal("Pressure (psi)"))
		// autoBounds inverts: a bound the Console computed is not a manual one.
		Expect(lp.Axes.Y1.ManualBounds).To(Equal(versions.ManualBounds{}))
		Expect(lp.Lines).To(HaveLen(1))
		Expect(lp.Lines[0].StrokeWidth).To(Equal(3.0))
		Expect(lp.Lines[0].Downsample).To(BeEquivalentTo(2))
		Expect(lp.Rules).To(HaveLen(1))
		Expect(lp.Rules[0].Position).To(Equal(950.0))
		Expect(lp.Title.Visible).To(BeTrue())
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v5.json").Key).To(Equal(uuid.Nil))
	})

	It("Should take the name from the envelope header", func(ctx SpecContext) {
		env := LoadEnvelope("testdata/import_v0_state.json")
		env.Name = "Renamed"
		Expect(MustSucceed(versions.DecodeImExEnvelope(ctx, env)).Name).
			To(Equal("Renamed"))
	})

	It("Should reject a version newer than Latest", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_bad_version.json"),
		)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("lineplot version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a v5 envelope with a mistyped body", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_v5_bad_body.json"),
		)).Error().To(MatchError(ContainSubstring("decode envelope body")))
	})

	It("Should reject a body carrying no line plot structure", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_unrecognized.json"),
		)).Error().To(MatchError(ContainSubstring(
			`file is not a line plot: no "axes" field`,
		)))
	})
})

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
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Symbol {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		sym := decode(ctx, "testdata/import_v2.json")
		Expect(sym.Data.SVG).To(Equal("<svg><rect/></svg>"))
		Expect(sym.Data.Variant).To(Equal("valve"))
		Expect(sym.Data.Scale).To(Equal(1.0))
		Expect(sym.Data.StrokeScaled).To(BeTrue())
		Expect(sym.Data.States).To(HaveLen(1))
		Expect(sym.Data.States[0].Regions[0].StrokeColor).
			To(HaveValue(Equal(color.MustFromHex("#ff0000"))))
		Expect(sym.Data.States[0].Regions[0].FillColor).
			To(HaveValue(Equal(color.MustFromHex("#00ff00"))))
		Expect(sym.Data.Handles).To(Equal([]versions.Handle{{
			Key:         "h1",
			Position:    spatial.XY{X: 0, Y: 0.5},
			Orientation: spatial.OuterLocationLeft,
		}}))
		Expect(sym.Data.PreviewViewport).To(HaveValue(Equal(spatial.Viewport{
			Position: spatial.XY{X: 1, Y: 2}, Zoom: 3,
		})))
	})

	It("Should decode the camelCase Console export", func(ctx SpecContext) {
		sym := decode(ctx, "testdata/import_console.json")
		Expect(sym.Data.SVG).To(Equal("<svg><circle/></svg>"))
		Expect(sym.Data.Variant).To(Equal("sensor"))
		Expect(sym.Data.Scale).To(Equal(2.0))
		Expect(sym.Data.StrokeScaled).To(BeTrue())
		Expect(sym.Data.States).To(Equal([]versions.State{{
			Key:  "base",
			Name: "Base",
			Regions: []versions.Region{{
				Key:         "r1",
				Name:        "Body",
				Selectors:   []string{"#body"},
				StrokeColor: new(color.MustFromHex("#123456")),
				FillColor:   new(color.MustFromHex("#654321")),
			}},
		}}))
		Expect(sym.Data.Handles).To(Equal([]versions.Handle{{
			Key:         "h1",
			Position:    spatial.XY{X: 1, Y: 0.5},
			Orientation: spatial.OuterLocationRight,
		}}))
		Expect(sym.Data.PreviewViewport).To(HaveValue(Equal(spatial.Viewport{
			Position: spatial.XY{X: 4, Y: 5}, Zoom: 6,
		})))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v2.json").Key).To(Equal(uuid.Nil))
	})

	It("Should take the name from the envelope header", func(ctx SpecContext) {
		env := LoadEnvelope("testdata/import_console.json")
		env.Name = "Renamed"
		Expect(MustSucceed(versions.DecodeImExEnvelope(ctx, env)).Name).
			To(Equal("Renamed"))
	})

	It("Should reject a version newer than Latest", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_bad_version.json"),
		)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("schematic_symbol version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	DescribeTable("Should reject a body carrying no symbol structure",
		func(ctx SpecContext, path string) {
			Expect(versions.DecodeImExEnvelope(
				ctx, LoadEnvelope(path),
			)).Error().To(MatchError(ContainSubstring(
				`file is not a symbol: no "data.svg" field`,
			)))
		},
		Entry("no data object", "testdata/import_unrecognized.json"),
		Entry("data object without an svg", "testdata/import_dataless.json"),
	)
})

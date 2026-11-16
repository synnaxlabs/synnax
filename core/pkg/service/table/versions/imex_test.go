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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Table {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	text := func(c versions.CellConfig) versions.TextCellConfig {
		GinkgoHelper()
		cfg, ok := c.Variant.(versions.TextCellConfig)
		return MustBeOk(cfg, ok)
	}

	type rowLayout struct {
		key   string
		size  float64
		cells []string
	}
	layout := func(t versions.Table) []rowLayout {
		rows := make([]rowLayout, len(t.Rows))
		for i, r := range t.Rows {
			rows[i] = rowLayout{key: r.Key, size: r.Size, cells: lo.Map(
				r.Cells,
				func(c versions.Cell, _ int) string { return c.Key },
			)}
		}
		return rows
	}

	cell := func(t versions.Table, key string) versions.CellConfig {
		GinkgoHelper()
		for _, r := range t.Rows {
			for _, c := range r.Cells {
				if c.Key == key {
					return c.Config
				}
			}
		}
		Fail("no row holds cell " + key)
		return versions.CellConfig{}
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		t := decode(ctx, "testdata/import_v1.json")
		Expect(layout(t)).To(Equal([]rowLayout{
			{key: "row-0", size: 30, cells: []string{"c1"}},
		}))
		Expect(t.Columns).To(Equal([]versions.Column{{Key: "col-0", Size: 100}}))
		Expect(text(cell(t, "c1")).Value).To(Equal("hi"))
	})

	It("Should key the rows and columns of a v2 envelope", func(ctx SpecContext) {
		t := decode(ctx, "testdata/import_v2.json")
		Expect(layout(t)).To(Equal([]rowLayout{
			{key: "row-0", size: 30, cells: []string{"c1"}},
			{key: "row-1", size: 40, cells: []string{"c2"}},
		}))
		Expect(t.Columns).To(Equal([]versions.Column{{Key: "col-0", Size: 100}}))
		Expect(text(cell(t, "c2")).Value).To(Equal("there"))
	})

	It(
		"Should decode the Console typed export stamped at the same version",
		func(ctx SpecContext) {
			t := decode(ctx, "testdata/import_typed_console.json")
			Expect(layout(t)).To(Equal([]rowLayout{
				{key: "row-0", size: 25, cells: []string{"c1"}},
			}))
			Expect(t.Columns).To(Equal([]versions.Column{{Key: "col-0", Size: 90}}))
			Expect(text(cell(t, "c1")).Weight).To(Equal(700.0))
		},
	)

	It("Should lift a Console state through the legacy chain", func(ctx SpecContext) {
		t := decode(ctx, "testdata/import_v0_state.json")
		Expect(layout(t)).To(Equal([]rowLayout{
			{key: "row-0", size: 30, cells: []string{"c1", "c2"}},
		}))
		Expect(t.Columns).To(Equal([]versions.Column{
			{Key: "col-0", Size: 100},
			{Key: "col-1", Size: 120},
		}))
		Expect(
			cell(t, "c2").Variant,
		).To(BeAssignableToTypeOf(versions.ValueCellConfig{}))
		Expect(text(cell(t, "c1")).Weight).To(Equal(700.0))
	})

	// The fixtures below are tables a shipped Console exported.
	It("Should lift a Console export whose rows sit at the top level", func(
		ctx SpecContext,
	) {
		t := decode(ctx, "testdata/import_console_v0_dataless.json")
		Expect(t.Rows).ToNot(BeEmpty())
		Expect(t.Columns).ToNot(BeEmpty())
	})

	It("Should lift a Console export whose state sits under data", func(
		ctx SpecContext,
	) {
		// The Console briefly wrote the whole table state twice: once at the top
		// level and once nested under `data`. The top level wins.
		t := decode(ctx, "testdata/import_console_v0_data.json")
		Expect(t.Rows).To(HaveLen(1))
		Expect(t.Rows[0].Size).To(Equal(36.0))
		Expect(layout(t)[0].cells).To(Equal([]string{
			"C0sVTFh81T3", "QghGMReXEch", "qp8JeDfJs10",
		}))
		Expect(t.Columns).To(HaveLen(3))
		Expect(
			text(cell(t, "C0sVTFh81T3")).Align,
		).To(Equal(versions.FlexAlignmentCenter))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v1.json").Key).To(Equal(uuid.Nil()))
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
			MatchError(ContainSubstring("table version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a body carrying no table structure", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_unrecognized.json"),
		)).Error().To(MatchError(ContainSubstring(
			`file is not a table: no "layout" field`,
		)))
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/service/table/versions"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Table {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	props := func(c versions.Cell) map[string]any {
		GinkgoHelper()
		var out map[string]any
		Expect(c.Props.Unmarshal(&out)).To(Succeed())
		return out
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		t := decode(ctx, "testdata/import_v1.json")
		Expect(t.Rows).To(Equal([]versions.Row{{Size: 30, Cells: []string{"c1"}}}))
		Expect(t.Columns).To(Equal([]versions.Column{{Size: 100}}))
		Expect(t.Cells["c1"].Variant).To(Equal("text"))
		Expect(props(t.Cells["c1"])).To(HaveKeyWithValue("value", "hi"))
	})

	It(
		"Should decode the Console typed export stamped at the same version",
		func(ctx SpecContext) {
			t := decode(ctx, "testdata/import_typed_console.json")
			Expect(t.Rows).To(Equal([]versions.Row{{Size: 25, Cells: []string{"c1"}}}))
			Expect(t.Columns).To(Equal([]versions.Column{{Size: 90}}))
			Expect(props(t.Cells["c1"])).To(HaveKeyWithValue("fooBar", 1.0))
		},
	)

	It("Should lift a Console state through the legacy chain", func(ctx SpecContext) {
		t := decode(ctx, "testdata/import_v0_state.json")
		Expect(t.Rows).To(
			Equal([]versions.Row{{Size: 30, Cells: []string{"c1", "c2"}}}),
		)
		Expect(t.Columns).To(Equal([]versions.Column{{Size: 100}, {Size: 120}}))
		Expect(t.Cells).To(HaveLen(2))
		Expect(t.Cells["c2"].Variant).To(Equal("value"))
		Expect(props(t.Cells["c1"])).To(HaveKeyWithValue("fooBar", 3.0))
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
		Expect(t.Rows[0].Cells).To(Equal([]string{
			"C0sVTFh81T3", "QghGMReXEch", "qp8JeDfJs10",
		}))
		Expect(t.Columns).To(HaveLen(3))
		Expect(t.Cells).To(HaveLen(3))
		Expect(t.Cells["C0sVTFh81T3"].Variant).To(Equal("text"))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v1.json").Key).To(Equal(uuid.Nil))
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

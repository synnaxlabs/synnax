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
	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Schematic {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	config := func(s versions.Schematic, node string) map[string]any {
		GinkgoHelper()
		var out map[string]any
		Expect(s.Configs[node].Unmarshal(&out)).To(Succeed())
		return out
	}

	It("Should decode a current-version envelope", func(ctx SpecContext) {
		sch := decode(ctx, "testdata/import_v8.json")
		Expect(sch.Snapshot).To(BeTrue())
		Expect(sch.Nodes).To(Equal([]versions.Node{
			{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 4},
		}))
		Expect(sch.Edges).To(Equal([]versions.Edge{{
			Key:    "e1",
			Source: versions.Handle{Node: "n1", Param: "out"},
			Target: versions.Handle{Node: "n2", Param: "in"},
		}}))
		Expect(config(sch, "n1")).To(HaveKeyWithValue("variant", "valve"))
	})

	It("Should lift a v7 envelope, dropping measured", func(ctx SpecContext) {
		sch := decode(ctx, "testdata/import_v7.json")
		Expect(sch.Snapshot).To(BeTrue())
		Expect(sch.Nodes).To(Equal([]versions.Node{
			{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 4},
		}))
		Expect(sch.Edges).To(Equal([]versions.Edge{{
			Key:    "e1",
			Source: versions.Handle{Node: "n1", Param: "out"},
			Target: versions.Handle{Node: "n2", Param: "in"},
		}}))
		Expect(config(sch, "n1")).To(HaveKeyWithValue("variant", "valve"))
	})

	It("Should decode the camelCase Console export", func(ctx SpecContext) {
		sch := decode(ctx, "testdata/import_typed_console.json")
		Expect(sch.Snapshot).To(BeFalse())
		Expect(sch.Nodes[0].ZIndex).To(Equal(int16(4)))
		Expect(sch.Edges[0].Source).
			To(Equal(versions.Handle{Node: "n1", Param: "out"}))
		Expect(config(sch, "n1")).To(HaveKeyWithValue("strokeWidth", 2.0))
	})

	It("Should lift a Console state through the legacy chain", func(ctx SpecContext) {
		sch := decode(ctx, "testdata/import_v3_state.json")
		Expect(sch.Nodes[0].ZIndex).To(Equal(int16(3)))
		Expect(sch.Edges[0].Source).To(Equal(versions.Handle{Node: "n1", Param: "a"}))
		Expect(sch.Edges[0].Target).To(Equal(versions.Handle{Node: "n2", Param: "b"}))
		Expect(config(sch, "n1")).To(HaveKeyWithValue("color", "#ff0000"))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v7.json").Key).To(Equal(uuid.Nil))
	})

	It("Should take the name from the envelope header", func(ctx SpecContext) {
		env := LoadEnvelope("testdata/import_v3_state.json")
		env.Name = "Renamed"
		Expect(MustSucceed(versions.DecodeImExEnvelope(ctx, env)).Name).
			To(Equal("Renamed"))
	})

	It("Should reject a version newer than Latest", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_bad_version.json"),
		)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("schematic version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})

	It("Should reject a body carrying no schematic structure", func(ctx SpecContext) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_unrecognized.json"),
		)).Error().To(MatchError(ContainSubstring(
			`file is not a schematic: no "props" field`,
		)))
	})

	// The typed export names its per-node bag configs, so the state chain's props
	// marker would reject every valid one.
	It("Should reject a typed export carrying no schematic structure", func(
		ctx SpecContext,
	) {
		Expect(versions.DecodeImExEnvelope(
			ctx, LoadEnvelope("testdata/import_unrecognized_typed.json"),
		)).Error().To(MatchError(ContainSubstring(
			`file is not a schematic: no "configs" field`,
		)))
	})
})

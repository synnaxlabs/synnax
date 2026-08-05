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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DecodeImExEnvelope", func() {
	decode := func(ctx SpecContext, path string) versions.Arc {
		GinkgoHelper()
		return MustSucceed(versions.DecodeImExEnvelope(ctx, LoadEnvelope(path)))
	}

	inputs := func(a versions.Arc, node string) map[string]any {
		GinkgoHelper()
		var out map[string]any
		Expect(a.Graph.Inputs[node].Unmarshal(&out)).To(Succeed())
		return out
	}

	It("Should decode a server-exported envelope", func(ctx SpecContext) {
		a := decode(ctx, "testdata/import_v3.json")
		Expect(a.Mode).To(Equal(versions.ModeGraph))
		Expect(a.Graph.Nodes).To(HaveLen(1))
		Expect(a.Graph.Nodes[0].Key).To(Equal("n1"))
		Expect(a.Graph.Edges[0].Source).To(Equal(ir.Handle{Node: "n1", Param: "out"}))
		Expect(a.Graph.Edges[0].Kind).To(Equal(ir.EdgeKindContinuous))
		Expect(inputs(a, "n1")).To(HaveKeyWithValue("type", "constant"))
	})

	It("Should decode the versionless Console export", func(ctx SpecContext) {
		a := decode(ctx, "testdata/import_typed_console.json")
		Expect(a.Mode).To(Equal(versions.ModeGraph))
		Expect(a.Graph.Nodes).To(HaveLen(2))
		Expect(a.Graph.Nodes[0].Key).To(Equal("n9"))
		Expect(a.Graph.Functions).To(HaveLen(1))
		Expect(a.Graph.Edges[0].Source).To(Equal(ir.Handle{Node: "n9", Param: "out"}))
		// The Console wrote keyless edges; the graph lift mints one.
		Expect(a.Graph.Edges[0].Key).ToNot(BeEmpty())
		Expect(inputs(a, "n9")).To(HaveKeyWithValue("keyOrName", "kept"))
	})

	It(
		"Should lift a v0 Console state, nesting edges and renaming set_status",
		func(ctx SpecContext) {
			a := decode(ctx, "testdata/import_v0_state.json")
			Expect(a.Mode).To(Equal(versions.ModeGraph))
			Expect(a.Graph.Nodes).To(HaveLen(2))
			Expect(a.Graph.Edges[0].Source).
				To(Equal(ir.Handle{Node: "n1", Param: "out"}))
			Expect(a.Graph.Edges[0].Target).
				To(Equal(ir.Handle{Node: "n2", Param: "in"}))
			Expect(inputs(a, "n1")).To(SatisfyAll(
				HaveKeyWithValue("type", "status.set"),
				HaveKeyWithValue("key_or_name", "st1"),
				HaveKeyWithValue("variant", "warning"),
				HaveKeyWithValue("message", "hi"),
			))
			// The CRDT document is seeded by the writer, so decode carries the raw
			// source alone.
			Expect(a.Text.Raw).To(Equal("a = b"))
		},
	)

	It("Should lift a v2 Console state without rewriting props", func(
		ctx SpecContext,
	) {
		a := decode(ctx, "testdata/import_v2_state.json")
		Expect(a.Mode).To(Equal(versions.ModeText))
		Expect(a.Graph.Edges[0].Source).To(Equal(ir.Handle{Node: "n1", Param: "out"}))
		Expect(inputs(a, "n1")).To(SatisfyAll(
			HaveKeyWithValue("type", "status.set"),
			HaveKeyWithValue("key_or_name", "st2"),
		))
	})

	It("Should drop the key on the wire", func(ctx SpecContext) {
		Expect(decode(ctx, "testdata/import_v3.json").Key).To(Equal(uuid.Nil))
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
			MatchError(ContainSubstring("arc version 99")),
			MatchError(ContainSubstring("newer than this Core supports")),
		))
	})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// loadEnvelope reads a wire-format envelope fixture from versions/testdata and
// unmarshals it into an imex.Envelope, binding the codec that Decode needs.
func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImEx", func() {
	Describe("Export", func() {
		It("Should export an arc as a versioned envelope", func(ctx SpecContext) {
			a := arc.Arc{Name: "exported", Mode: arc.ModeText}
			Expect(svc.NewWriter(nil).Create(ctx, &a)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, arc.OntologyID(a.Key)))
			Expect(env.Version).To(Equal(arc.Version))
			Expect(env.Type).To(Equal("arc"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[arc.Arc](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Mode).To(Equal(arc.ModeText))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeArc, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeArc, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, path string, opts imex.ImportOptions,
		) arc.Arc {
			id := MustSucceed(imexSvc.Import(ctx, db, loadEnvelope(path), opts))
			Expect(id.Type).To(Equal(ontology.ResourceTypeArc))
			key := MustSucceed(uuid.Parse(id.Key))
			var res arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		inputsOf := func(a arc.Arc, node string) map[string]any {
			Expect(a.Graph.Inputs).To(HaveKey(node))
			return map[string]any(a.Graph.Inputs[node])
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v3.json", imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Mode).To(Equal(arc.ModeGraph))
			Expect(res.Graph.Nodes).To(HaveLen(1))
			Expect(res.Graph.Nodes[0].Key).To(Equal("n1"))
			Expect(res.Graph.Edges).To(HaveLen(1))
			Expect(res.Graph.Edges[0].Source).To(
				Equal(ir.Handle{Node: "n1", Param: "out"}),
			)
			Expect(inputsOf(res, "n1")).To(HaveKeyWithValue("type", "constant"))
		})

		It("Should import a Console typed export carrying camelCase keys", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_typed_console.json", imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Console Typed"))
			Expect(res.Graph.Nodes[0].Key).To(Equal("n9"))
			// Inputs are opaque per-function records: their keys keep whatever
			// casing the file carried.
			Expect(inputsOf(res, "n9")).To(HaveKeyWithValue("keyOrName", "kept"))
		})

		It("Should import a v3 Console state from its pendingUpload", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v3_state.json",
				imex.ImportOptions{FileName: "My Arc.json"},
			)
			Expect(res.Name).To(Equal("My Arc"))
			Expect(res.Mode).To(Equal(arc.ModeGraph))
			Expect(res.Graph.Nodes[0].Key).To(Equal("n1"))
			Expect(res.Graph.Edges[0].Kind).To(Equal(ir.EdgeKindContinuous))
			Expect(res.Text.Materialize().Raw).To(Equal("x = 1"))
			Expect(inputsOf(res, "n1")).To(HaveKeyWithValue("value", 7.0))
		})

		It("Should reject a v3 Console state with no graph data", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_v3_state_empty.json"),
				imex.ImportOptions{FileName: "empty.json"},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("no graph data")),
			))
		})

		It("Should import a v0 Console state, lifting flat edges and renaming set_status", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v0_state.json",
				imex.ImportOptions{FileName: "Legacy Arc.json"},
			)
			Expect(res.Name).To(Equal("Legacy Arc"))
			Expect(res.Mode).To(Equal(arc.ModeGraph))
			Expect(res.Graph.Nodes).To(HaveLen(2))
			Expect(res.Graph.Edges).To(HaveLen(1))
			Expect(res.Graph.Edges[0].Source).To(
				Equal(ir.Handle{Node: "n1", Param: "out"}),
			)
			Expect(res.Graph.Edges[0].Target).To(
				Equal(ir.Handle{Node: "n2", Param: "in"}),
			)
			Expect(res.Graph.Edges[0].Kind).To(Equal(ir.EdgeKindContinuous))
			n1 := inputsOf(res, "n1")
			Expect(n1).To(HaveKeyWithValue("type", "status.set"))
			Expect(n1).To(HaveKeyWithValue("key_or_name", "st1"))
			Expect(n1).To(HaveKeyWithValue("variant", "warning"))
			Expect(n1).To(HaveKeyWithValue("message", "hi"))
			n2 := inputsOf(res, "n2")
			Expect(n2).To(HaveKeyWithValue("type", "constant"))
			Expect(n2).To(HaveKeyWithValue("value", 9.0))
			Expect(res.Text.Materialize().Raw).To(Equal("a = b"))
		})

		It("Should import a v2 Console state without rewriting props", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v2_state.json",
				imex.ImportOptions{FileName: "V2 Arc.json"},
			)
			Expect(res.Mode).To(Equal(arc.ModeText))
			Expect(res.Graph.Edges[0].Source).To(
				Equal(ir.Handle{Node: "n1", Param: "out"}),
			)
			n1 := inputsOf(res, "n1")
			Expect(n1).To(HaveKeyWithValue("type", "status.set"))
			Expect(n1).To(HaveKeyWithValue("key_or_name", "st2"))
		})

		It("Should reject an envelope newer than the supported version", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_bad_version.json"),
				imex.ImportOptions{},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("arc version 99")),
				MatchError(ContainSubstring("newer than this Core supports")),
			))
		})

		It("Should generate a fresh key, discarding the key on the wire", func(ctx SpecContext) {
			id := MustSucceed(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_v3.json"), imex.ImportOptions{},
			))
			Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
		})
	})

	Describe("Round trip", func() {
		It("Should preserve arc content through export then import", func(ctx SpecContext) {
			original := arc.Arc{Name: "round-trip", Mode: arc.ModeText}
			original.Text.Raw = "b = 2"
			Expect(svc.NewWriter(nil).Create(ctx, &original)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, arc.OntologyID(original.Key)))
			id := MustSucceed(imexSvc.Import(
				ctx, db, WireRoundTrip(env), imex.ImportOptions{},
			))
			key := MustSucceed(uuid.Parse(id.Key))
			Expect(key).ToNot(Equal(original.Key))
			var res arc.Arc
			Expect(svc.NewRetrieve().
				Where(arc.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Name).To(Equal("round-trip"))
			Expect(res.Mode).To(Equal(arc.ModeText))
			Expect(res.Text.Materialize().Raw).To(Equal("b = 2"))
		})
	})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
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
		It("Should export a schematic as a versioned envelope", func(ctx SpecContext) {
			s := schematic.Schematic{Name: "exported", Snapshot: true}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &s)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, schematic.OntologyID(s.Key)))
			Expect(env.Version).To(Equal(schematic.Version))
			Expect(env.Type).To(Equal("schematic"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[schematic.Schematic](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Snapshot).To(BeTrue())
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: uuid.NewString()}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeSchematic, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})

	Describe("Import", func() {
		importAndRetrieve := func(
			ctx SpecContext, path string, opts imex.ImportOptions,
		) schematic.Schematic {
			id := MustSucceed(imexSvc.Import(ctx, db, loadEnvelope(path), opts))
			Expect(id.Type).To(Equal(ontology.ResourceTypeSchematic))
			key := MustSucceed(uuid.Parse(id.Key))
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			return res
		}

		It("Should import a current snake_case envelope", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v7.json", imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Server Typed"))
			Expect(res.Snapshot).To(BeTrue())
			Expect(res.Nodes).To(HaveLen(1))
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(4)))
			Expect(res.Edges[0].Source).To(
				Equal(schematic.Handle{Node: "n1", Param: "out"}),
			)
			Expect(res.Configs).To(HaveKey("n1"))
		})

		It("Should import a Console typed export carrying camelCase keys", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_typed_console.json", imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Console Typed"))
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(4)))
			var cfg map[string]any
			Expect(res.Configs["n1"].Unmarshal(&cfg)).To(Succeed())
			Expect(cfg).To(HaveKeyWithValue("variant", "valve"))
			Expect(cfg).To(HaveKey("strokeWidth"))
		})

		It("Should import a versionless Console typed export", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_typed_versionless.json",
				imex.ImportOptions{},
			)
			Expect(res.Name).To(Equal("Old Typed"))
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(1)))
		})

		It("Should import a v6 Console state from its pendingUpload", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v6_state.json",
				imex.ImportOptions{FileName: "My Schematic.json"},
			)
			Expect(res.Name).To(Equal("My Schematic"))
			Expect(res.Snapshot).To(BeTrue())
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(2)))
			Expect(res.Edges[0].Source).To(
				Equal(schematic.Handle{Node: "n1", Param: "out"}),
			)
			var cfg map[string]any
			Expect(res.Configs["n1"].Unmarshal(&cfg)).To(Succeed())
			Expect(cfg).To(HaveKeyWithValue("variant", "tank"))
			Expect(cfg).To(HaveKey("strokeWidth"))
		})

		It("Should reject a v6 Console state with no document", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_v6_state_empty.json"),
				imex.ImportOptions{FileName: "empty.json"},
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("no document data")),
			))
		})

		It("Should import a v3 Console state through the legacy chain", func(ctx SpecContext) {
			res := importAndRetrieve(
				ctx, "versions/testdata/import_v3_state.json",
				imex.ImportOptions{FileName: "Legacy Schematic.json"},
			)
			Expect(res.Name).To(Equal("Legacy Schematic"))
			Expect(res.Nodes[0].ZIndex).To(Equal(int16(3)))
			Expect(res.Edges[0].Source).To(
				Equal(schematic.Handle{Node: "n1", Param: "a"}),
			)
			Expect(res.Edges[0].Target).To(
				Equal(schematic.Handle{Node: "n2", Param: "b"}),
			)
			var cfg map[string]any
			Expect(res.Configs["n1"].Unmarshal(&cfg)).To(Succeed())
			Expect(cfg).To(HaveKeyWithValue("variant", "valve"))
			Expect(cfg).To(HaveKeyWithValue("color", "#ff0000"))
		})

		It("Should reject an envelope newer than the supported version", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_bad_version.json"),
				imex.ImportOptions{},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("schematic version 99")),
				MatchError(ContainSubstring("newer than this Core supports")),
			))
		})

		It("Should generate a fresh key, discarding the key on the wire", func(ctx SpecContext) {
			id := MustSucceed(imexSvc.Import(ctx, db,
				loadEnvelope("versions/testdata/import_v7.json"), imex.ImportOptions{},
			))
			Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
		})
	})

	Describe("Round trip", func() {
		It("Should preserve schematic content through export then import", func(ctx SpecContext) {
			original := schematic.Schematic{
				Name:  "round-trip",
				Nodes: []schematic.Node{{Key: "n1", ZIndex: 5}},
				Edges: []schematic.Edge{{
					Key:    "e1",
					Source: schematic.Handle{Node: "n1", Param: "out"},
					Target: schematic.Handle{Node: "n2", Param: "in"},
				}},
			}
			Expect(svc.NewWriter(nil).Create(ctx, proj.Key, &original)).To(Succeed())
			env := MustSucceed(svc.Export(ctx, schematic.OntologyID(original.Key)))
			id := MustSucceed(imexSvc.Import(
				ctx, db, WireRoundTrip(env), imex.ImportOptions{},
			))
			key := MustSucceed(uuid.Parse(id.Key))
			Expect(key).ToNot(Equal(original.Key))
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			Expect(res.Name).To(Equal("round-trip"))
			Expect(res.Nodes).To(Equal(original.Nodes))
			Expect(res.Edges).To(Equal(original.Edges))
		})
	})
})

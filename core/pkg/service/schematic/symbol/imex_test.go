// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"encoding/json"
	"os"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
		It("Should export a symbol as a versioned envelope", func(ctx SpecContext) {
			sym := symbol.Symbol{
				Name: "exported",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve"},
			}
			// Export reads committed data, so the symbol is created outside the
			// per-spec tx and must be deleted to keep the shared DB's counts intact.
			Expect(svc.NewWriter(nil).Create(ctx, &sym, proj.OntologyID())).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, sym.Key)).To(Succeed())
			})
			env := MustSucceed(svc.Export(ctx, symbol.OntologyID(sym.Key)))
			Expect(env.Version).To(Equal(symbol.Version))
			Expect(env.Type).To(Equal("schematic_symbol"))
			Expect(env.Name).To(Equal("exported"))

			decoded := MustSucceed(imex.Decode[symbol.Symbol](ctx, WireRoundTrip(env)))
			Expect(decoded.Name).To(Equal("exported"))
			Expect(decoded.Data.SVG).To(Equal("<svg/>"))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := ontology.ID{
				Type: ontology.ResourceTypeSchematicSymbol,
				Key:  uuid.NewString(),
			}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should error on an invalid UUID key", func(ctx SpecContext) {
			id := ontology.ID{Type: ontology.ResourceTypeSchematicSymbol, Key: "not-a-uuid"}
			Expect(svc.Export(ctx, id)).Error().To(MatchError(ContainSubstring("UUID")))
		})
	})

	Describe("Import", func() {
		// Imports run on the per-spec tx so created rows roll back and the shared
		// DB's symbol counts stay intact for the other specs.
		importAndRetrieve := func(ctx SpecContext, path string) symbol.Symbol {
			id := MustSucceed(imexSvc.Import(
				ctx, tx, loadEnvelope(path), imex.ImportOptions{},
			))
			Expect(id.Type).To(Equal(ontology.ResourceTypeSchematicSymbol))
			key := MustSucceed(uuid.Parse(id.Key))
			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			return res
		}

		It("Should import a server export carrying snake_case keys", func(ctx SpecContext) {
			res := importAndRetrieve(ctx, "versions/testdata/import_v2.json")
			Expect(res.Name).To(Equal("Server Symbol"))
			Expect(res.Data.SVG).To(Equal("<svg><rect/></svg>"))
			Expect(res.Data.Variant).To(Equal("valve"))
			Expect(res.Data.ScaleStroke).To(BeTrue())
			Expect(res.Data.States).To(HaveLen(1))
			region := res.Data.States[0].Regions[0]
			Expect(region.StrokeColor).To(HaveValue(Equal("#ff0000")))
			Expect(region.FillColor).To(HaveValue(Equal("#00ff00")))
			Expect(res.Data.Handles).To(HaveLen(1))
			Expect(res.Data.PreviewViewport).ToNot(BeNil())
			Expect(res.Data.PreviewViewport.Zoom).To(Equal(3.0))
		})

		It("Should import a Console export carrying camelCase keys", func(ctx SpecContext) {
			res := importAndRetrieve(ctx, "versions/testdata/import_console.json")
			Expect(res.Name).To(Equal("Console Symbol"))
			Expect(res.Data.Variant).To(Equal("sensor"))
			Expect(res.Data.Scale).To(Equal(2.0))
			Expect(res.Data.ScaleStroke).To(BeTrue())
			Expect(res.Data.States).To(HaveLen(1))
			region := res.Data.States[0].Regions[0]
			Expect(region.StrokeColor).To(HaveValue(Equal("#123456")))
			Expect(region.FillColor).To(HaveValue(Equal("#654321")))
			Expect(res.Data.Handles).To(HaveLen(1))
			Expect(res.Data.PreviewViewport).ToNot(BeNil())
			Expect(res.Data.PreviewViewport.Zoom).To(Equal(6.0))
		})

		It("Should parent the imported symbol under the permanent symbol group", func(ctx SpecContext) {
			id := MustSucceed(imexSvc.Import(
				ctx, tx,
				loadEnvelope("versions/testdata/import_v2.json"),
				imex.ImportOptions{},
			))
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: svc.Group().OntologyID(),
				Type: ontology.RelationshipTypeParentOf,
				To:   id,
			})).To(BeTrue())
		})

		It("Should reject an envelope newer than the supported version", func(ctx SpecContext) {
			Expect(imexSvc.Import(ctx, tx,
				loadEnvelope("versions/testdata/import_bad_version.json"),
				imex.ImportOptions{},
			)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("schematic_symbol version 99")),
				MatchError(ContainSubstring("newer than this Core supports")),
			))
		})

		It("Should generate a fresh key, discarding the key on the wire", func(ctx SpecContext) {
			id := MustSucceed(imexSvc.Import(ctx, tx,
				loadEnvelope("versions/testdata/import_v2.json"), imex.ImportOptions{},
			))
			Expect(id.Key).ToNot(Equal("11111111-2222-3333-4444-555555555555"))
		})
	})

	Describe("Round trip", func() {
		It("Should preserve symbol content through export then import", func(ctx SpecContext) {
			original := symbol.Symbol{
				Name: "round-trip",
				Data: symbol.Spec{SVG: "<svg/>", Variant: "valve", ScaleStroke: true},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &original, proj.OntologyID())).
				To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(svc.NewWriter(nil).Delete(ctx, original.Key)).To(Succeed())
			})
			env := MustSucceed(svc.Export(ctx, symbol.OntologyID(original.Key)))
			id := MustSucceed(imexSvc.Import(
				ctx, tx, WireRoundTrip(env), imex.ImportOptions{},
			))
			key := MustSucceed(uuid.Parse(id.Key))
			Expect(key).ToNot(Equal(original.Key))
			var res symbol.Symbol
			Expect(svc.NewRetrieve().
				Where(symbol.MatchKeys(key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("round-trip"))
			Expect(res.Data).To(Equal(original.Data))
		})
	})
})

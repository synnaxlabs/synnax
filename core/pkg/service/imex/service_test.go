// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

func samplePayload() map[string]any {
	return map[string]any{"field_one": "value", "field_two": float64(42)}
}

var _ = Describe("Service", func() {
	Describe("ServiceConfig", func() {
		Describe("Validate", func() {
			It("Should fail when DB is nil", func() {
				cfg := imex.ServiceConfig{}
				Expect(cfg.Validate()).To(MatchError(ContainSubstring("db")))
			})

			It("Should succeed when DB is set", func() {
				cfg := imex.ServiceConfig{DB: db}
				Expect(cfg.Validate()).To(Succeed())
			})
		})

		Describe("Override", func() {
			It("Should prefer the other config's DB when set", func() {
				initial := imex.ServiceConfig{}
				other := imex.ServiceConfig{DB: db}
				merged := initial.Override(other)
				Expect(merged.DB).To(BeIdenticalTo(other.DB))
			})

			It("Should keep the receiver's DB when other's is nil", func() {
				initial := imex.ServiceConfig{DB: db}
				merged := initial.Override(imex.ServiceConfig{})
				Expect(merged.DB).To(BeIdenticalTo(db))
			})
		})
	})

	Describe("NewService", func() {
		It("Should fail when no DB is provided", func() {
			Expect(imex.NewService(imex.ServiceConfig{})).Error().To(
				MatchError(ContainSubstring("db")),
			)
		})

		It("Should succeed when a DB is provided", func() {
			Expect(imex.NewService(imex.ServiceConfig{DB: db})).ToNot(BeNil())
		})
	})

	Describe("ImporterType", func() {
		It("Should return the registered importer's broader Type", func() {
			Expect(svc.ImporterType(string(testResourceType))).To(Equal(testResourceType))
		})

		It("Should return a validation error scoped to the type field if not registered", func() {
			Expect(svc.ImporterType("nonexistent")).Error().To(SatisfyAll(
				MatchError(ContainSubstring("type")),
				MatchError(ContainSubstring("no importer registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
	})

	Describe("RegisterImporter", func() {
		It("Should register an importer under a narrow type string", func() {
			s := MustSucceed(imex.NewService(imex.ServiceConfig{DB: db}))
			s.RegisterImporter("narrow", noopImporter{typ: "broad"})
			Expect(s.ImporterType("narrow")).To(Equal(ontology.ResourceType("broad")))
		})

		It(
			"Should map the narrow type to the importer's broader Type for access control",
			func(ctx SpecContext) {
				s := MustSucceed(imex.NewService(imex.ServiceConfig{DB: db}))
				s.RegisterImporter("http_read", noopImporter{typ: "task"})
				s.RegisterImporter("opc_scan", noopImporter{typ: "task"})
				Expect(s.ImporterType("http_read")).To(Equal(ontology.ResourceType("task")))
				Expect(s.ImporterType("opc_scan")).To(Equal(ontology.ResourceType("task")))
				keys := MustSucceed(s.Import(ctx, []imex.Envelope{
					{Version: 1, Type: "http_read", Name: "ingest"},
					{Version: 1, Type: "opc_scan", Name: "scan"},
				}))
				Expect(keys).To(HaveLen(2))
				Expect(keys[0]).To(Equal("noop-key"))
				Expect(keys[1]).To(Equal("noop-key"))
			},
		)
	})

	Describe("RegisterExporter", func() {
		It("Should register an exporter under its own Type", func(ctx SpecContext) {
			s := MustSucceed(imex.NewService(imex.ServiceConfig{DB: db}))
			s.RegisterExporter(noopExporter{typ: "noop_export"})
			result := MustSucceed(s.Export(ctx, []ontology.ID{{
				Type: "noop_export",
				Key:  "any",
			}}))
			Expect(result).To(HaveLen(1))
			Expect(result[0].Type).To(Equal("noop_export"))
			Expect(result[0].Name).To(Equal("noop"))
		})
	})

	Describe("Import", func() {
		It("Should route to the correct service by type and return the new key", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: testVersion,
				Type:    string(testResourceType),
				Name:    "Registry Test",
				Data:    samplePayload(),
			}}
			keys := MustSucceed(svc.Import(ctx, envs))
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).NotTo(BeEmpty())
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: testVersion,
				Type:    "nonexistent",
				Name:    "Bad Type",
				Data:    map[string]any{},
			}}
			Expect(svc.Import(ctx, envs)).Error().To(SatisfyAll(
				MatchError(ContainSubstring("no importer registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
		It("Should pass errors from the importer through verbatim", func(ctx SpecContext) {
			Expect(svc.Import(ctx, []imex.Envelope{{
				Version: testVersion,
				Type:    string(errorResourceType),
				Name:    "Erroring",
				Data:    samplePayload(),
			}})).Error().To(MatchError(ContainSubstring("importer error: forced failure")))
		})

		It("Should roll back the transaction if any envelope's import fails", func(ctx SpecContext) {
			envs := []imex.Envelope{
				{
					Version: testVersion,
					Type:    string(testResourceType),
					Name:    "Good Record",
					Data:    samplePayload(),
				},
				{
					Version: testVersion,
					Type:    "nonexistent",
					Name:    "Bad Type",
					Data:    map[string]any{},
				},
			}
			Expect(svc.Import(ctx, envs)).Error().To(
				MatchError(ContainSubstring("no importer registered")),
			)
			Expect(ts.RetrieveByName(ctx, "Good Record")).Error().To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Export", func() {
		It("Should round-trip a registered resource through Import then Export", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: testVersion,
				Type:    string(testResourceType),
				Name:    "Round Trip",
				Data:    samplePayload(),
			}}
			keys := MustSucceed(svc.Import(ctx, envs))
			Expect(keys).To(HaveLen(1))
			result := MustSucceed(svc.Export(ctx, []ontology.ID{{
				Type: testResourceType,
				Key:  keys[0],
			}}))
			Expect(result).To(HaveLen(1))
			Expect(result[0].Version).To(Equal(testVersion))
			Expect(result[0].Type).To(Equal(string(testResourceType)))
			Expect(result[0].Name).To(Equal("Round Trip"))
			Expect(result[0].Data).To(HaveKeyWithValue("field_one", "value"))
		})
		It("Should pass errors from the exporter through verbatim", func(ctx SpecContext) {
			Expect(svc.Export(ctx, []ontology.ID{{
				Type: errorResourceType,
				Key:  "any-key",
			}})).Error().To(MatchError(ContainSubstring("exporter error: forced failure")))
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			Expect(svc.Export(ctx, []ontology.ID{{
				Type: "nonexistent",
				Key:  "660e8400-e29b-41d4-a716-446655440000",
			}})).Error().To(SatisfyAll(
				MatchError(ContainSubstring("no exporter registered")),
				MatchError(ContainSubstring("validation error")),
			))
		})
	})
})

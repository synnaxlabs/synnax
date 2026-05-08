// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
)

var _ = Describe("ServiceConfig", func() {
	Describe("Validate", func() {
		It("Should succeed when all required fields are set", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imexSvc,
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("Should fail when DB is nil", func() {
			cfg := log.ServiceConfig{
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imexSvc,
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("db")))
		})

		It("Should fail when Ontology is nil", func() {
			cfg := log.ServiceConfig{
				DB:     db,
				Search: &search.Index{},
				ImEx:   imexSvc,
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("ontology")))
		})

		It("Should fail when Search is nil", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				ImEx:     imexSvc,
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("search")))
		})

		It("Should fail when ImEx is nil", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("imex")))
		})
	})

	Describe("Override", func() {
		It("Should prefer the other config's ImEx when set", func() {
			initial := log.ServiceConfig{}
			other := log.ServiceConfig{ImEx: imexSvc}
			merged := initial.Override(other)
			Expect(merged.ImEx).To(BeIdenticalTo(other.ImEx))
		})

		It("Should keep the receiver's ImEx when other's is nil", func() {
			initial := log.ServiceConfig{ImEx: imexSvc}
			merged := initial.Override(log.ServiceConfig{})
			Expect(merged.ImEx).To(BeIdenticalTo(imexSvc))
		})

		It("Should prefer the other config's DB, Ontology, and Search when set", func() {
			otherImex, err := imex.NewService(imex.ServiceConfig{DB: db})
			Expect(err).ToNot(HaveOccurred())
			initial := log.ServiceConfig{}
			other := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     otherImex,
			}
			merged := initial.Override(other)
			Expect(merged.DB).To(BeIdenticalTo(other.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(other.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(other.Search))
			Expect(merged.ImEx).To(BeIdenticalTo(otherImex))
		})

		It("Should preserve the receiver's fields when other's are nil", func() {
			initial := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
				ImEx:     imexSvc,
			}
			merged := initial.Override(log.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(initial.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(initial.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(initial.Search))
			Expect(merged.ImEx).To(BeIdenticalTo(initial.ImEx))
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should fail when ServiceConfig is missing required fields", func(ctx SpecContext) {
		Expect(log.OpenService(ctx, log.ServiceConfig{DB: db})).Error().To(
			MatchError(ContainSubstring("ontology")),
		)
	})

	It("Should register itself as an importer/exporter on the ImEx service", func(ctx SpecContext) {
		// The suite-wide svc is already registered against imexSvc; verify that
		// ImporterType resolves to the log resource type.
		Expect(imexSvc.ImporterType(string(ontology.ResourceTypeLog))).
			To(Equal(ontology.ResourceTypeLog))
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/log"
)

var _ = Describe("ServiceConfig", func() {
	Describe("Validate", func() {
		It("Should succeed when all required fields are set", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		It("Should fail when DB is nil", func() {
			cfg := log.ServiceConfig{
				Ontology: otg,
				Search:   &search.Index{},
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("db")))
		})

		It("Should fail when Ontology is nil", func() {
			cfg := log.ServiceConfig{
				DB:     db,
				Search: &search.Index{},
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("ontology")))
		})

		It("Should fail when Search is nil", func() {
			cfg := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}
			Expect(cfg.Validate()).To(MatchError(ContainSubstring("search")))
		})
	})

	Describe("Override", func() {
		It("Should prefer the other config's DB, Ontology, and Search when set", func() {
			initial := log.ServiceConfig{}
			other := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
			}
			merged := initial.Override(other)
			Expect(merged.DB).To(BeIdenticalTo(other.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(other.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(other.Search))
		})

		It("Should preserve the receiver's fields when other's are nil", func() {
			initial := log.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   &search.Index{},
			}
			merged := initial.Override(log.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(initial.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(initial.Ontology))
			Expect(merged.Search).To(BeIdenticalTo(initial.Search))
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should fail when ServiceConfig is missing required fields", func(ctx SpecContext) {
		Expect(log.OpenService(ctx, log.ServiceConfig{DB: db})).Error().To(
			MatchError(ContainSubstring("ontology")),
		)
	})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
)

var _ = Describe("ServiceConfig", func() {
	full := func() panel.ServiceConfig {
		return panel.ServiceConfig{
			DB:       node.DB,
			Ontology: otg,
			Search:   searchIdx,
		}
	}

	Describe("Validate", func() {
		It("Should succeed when all required fields are set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a required field is missing",
			func(clear func(*panel.ServiceConfig), field string) {
				cfg := full()
				clear(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
			},
			Entry("db", func(c *panel.ServiceConfig) { c.DB = nil }, "db"),
			Entry(
				"ontology",
				func(c *panel.ServiceConfig) { c.Ontology = nil },
				"ontology",
			),
			Entry("search", func(c *panel.ServiceConfig) { c.Search = nil }, "search"),
		)

		It("Should succeed without a Signals provider", func() {
			cfg := full()
			cfg.Signals = nil
			Expect(cfg.Validate()).To(Succeed())
		})
	})

	Describe("Override", func() {
		It("Should prefer the other config's fields when set", func() {
			merged := panel.ServiceConfig{}.Override(full())
			Expect(merged.DB).To(BeIdenticalTo(node.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(otg))
			Expect(merged.Search).To(BeIdenticalTo(searchIdx))
		})

		It("Should preserve the receiver's fields when other's are nil", func() {
			merged := full().Override(panel.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(node.DB))
			Expect(merged.Ontology).To(BeIdenticalTo(otg))
		})
	})
})

var _ = Describe("OpenService", func() {
	It("Should fail when a required field is missing", func(ctx SpecContext) {
		Expect(panel.OpenService(ctx, panel.ServiceConfig{DB: node.DB})).Error().
			To(MatchError(ContainSubstring("ontology")))
	})
})

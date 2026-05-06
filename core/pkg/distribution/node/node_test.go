// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
)

var _ = Describe("Service", func() {
	Describe("ServiceConfig.Validate", func() {
		It("Should return nil when all required fields are set", func() {
			cfg := node.ServiceConfig{
				Cluster:  testCluster.Nodes[1].Cluster,
				Ontology: &ontology.Ontology{},
				Search:   &search.Index{},
			}
			Expect(cfg.Validate()).To(Succeed())
		})

		DescribeTable(
			"Should reject configurations missing a required field",
			func(field string, mutate func(*node.ServiceConfig)) {
				cfg := node.ServiceConfig{
					Cluster:  testCluster.Nodes[1].Cluster,
					Ontology: &ontology.Ontology{},
					Search:   &search.Index{},
				}
				mutate(&cfg)
				Expect(cfg.Validate()).To(SatisfyAll(
					MatchError(ContainSubstring(field)),
					MatchError(ContainSubstring("must be non-nil")),
				))
			},
			Entry("cluster", "cluster", func(c *node.ServiceConfig) { c.Cluster = nil }),
			Entry("ontology", "ontology", func(c *node.ServiceConfig) { c.Ontology = nil }),
			Entry("search", "search", func(c *node.ServiceConfig) { c.Search = nil }),
		)
	})

	Describe("NewService", func() {
		It("Should define the free-node sentinel resource in the ontology", func(ctx SpecContext) {
			Expect(testOtg.NewWriter(nil).HasResource(ctx, node.OntologyID(node.KeyFree))).
				To(BeTrue())
		})

		It("Should reject a config that fails validation", func(ctx SpecContext) {
			Expect(node.NewService(ctx, node.ServiceConfig{})).
				Error().To(MatchError(ContainSubstring("must be non-nil")))
		})
	})
})

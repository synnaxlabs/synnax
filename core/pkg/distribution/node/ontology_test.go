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
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
)

var ctx = context.Background()

var _ = Describe("Ontology", Ordered, func() {
	var mockCluster *mock.Cluster
	BeforeAll(func() { mockCluster = mock.ProvisionCluster(ctx, 3) })
	AfterAll(func() { Expect(mockCluster.Close()).To(Succeed()) })

	Describe("OntologyID", func() {
		It("Should return a correctly formatted ontology ID", func() {
			id := node.OntologyID(1)
			Expect(id.Type).To(Equal(node.OntologyType))
			Expect(id.Key).To(Equal("1"))
		})

		It("Should handle the free node key", func() {
			id := node.OntologyID(node.KeyFree)
			Expect(id.Type).To(Equal(node.OntologyType))
			Expect(id.Key).To(Equal("0"))
		})
	})

	Describe("OntologyService", func() {
		var svc *node.OntologyService

		BeforeAll(func() {
			svc = &node.OntologyService{
				Cluster:  mockCluster.Nodes[1].Cluster,
				Ontology: mockCluster.Nodes[1].Ontology,
			}
		})

		Describe("Type", func() {
			It("Should return the correct type", func() {
				Expect(svc.Type()).To(Equal(ontology.Type("node")))
			})
		})

		Describe("Schema", func() {
			It("Should return a non-nil schema", func() {
				Expect(svc.Schema()).ToNot(BeNil())
			})
		})

		Describe("RetrieveResource", func() {
			It("Should retrieve a node resource by key", func() {
				r, err := svc.RetrieveResource(ctx, "1", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(r.ID.Type).To(Equal(node.OntologyType))
				Expect(r.ID.Key).To(Equal("1"))
			})

			It("Should retrieve the free node resource", func() {
				r, err := svc.RetrieveResource(ctx, "0", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(r.ID.Type).To(Equal(node.OntologyType))
				Expect(r.ID.Key).To(Equal("0"))
			})

			It("Should return an error for an invalid key", func() {
				_, err := svc.RetrieveResource(ctx, "invalid", nil)
				Expect(err).To(HaveOccurred())
			})

			It("Should return an error for a non-existent node", func() {
				_, err := svc.RetrieveResource(ctx, "999", nil)
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("OpenNexter", func() {
			It("Should iterate over all nodes", func() {
				seq, closer, err := svc.OpenNexter(ctx)
				Expect(err).ToNot(HaveOccurred())
				defer closer.Close()
				resources := slices.Collect(seq)
				Expect(len(resources)).To(BeNumerically(">=", 3))
				for _, r := range resources {
					Expect(r.ID.Type).To(Equal(node.OntologyType))
				}
			})
		})
	})
})

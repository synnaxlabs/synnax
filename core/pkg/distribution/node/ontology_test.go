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
	"iter"
	"slices"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/change"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", Ordered, func() {
	var (
		mockCluster *mock.Cluster
		ctx         context.Context
	)
	BeforeAll(func() {
		ctx = context.Background()
		mockCluster = mock.ProvisionCluster(ctx, 3)
	})
	AfterAll(func() { Expect(mockCluster.Close()).To(Succeed()) })
	BeforeEach(func() { ctx = context.Background() })

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
			It("Should successfully dump a node to a standardized format", func() {
				n := node.Node{Key: 1, Address: "localhost:9090"}
				dumped, err := svc.Schema().Dump(n)
				Expect(err).ToNot(HaveOccurred())
				m, ok := dumped.(map[string]any)
				Expect(ok).To(BeTrue())
				Expect(m).To(HaveKey("key"))
				Expect(m).To(HaveKey("address"))
				Expect(m).To(HaveKey("state"))
			})
		})

		Describe("ListenForChanges", func() {
			It("Should define the free node resource in the ontology", func() {
				svc.ListenForChanges(ctx)
				r := MustSucceed(svc.RetrieveResource(ctx, "0", nil))
				Expect(r.ID).To(Equal(node.OntologyID(node.KeyFree)))
			})
		})

		Describe("RetrieveResource", func() {
			It("Should retrieve a node resource by key", func() {
				r := MustSucceed(svc.RetrieveResource(ctx, "1", nil))
				Expect(r.ID.Type).To(Equal(node.OntologyType))
				Expect(r.ID.Key).To(Equal("1"))
			})

			It("Should retrieve the free node resource", func() {
				r := MustSucceed(svc.RetrieveResource(ctx, "0", nil))
				Expect(r.ID.Type).To(Equal(node.OntologyType))
				Expect(r.ID.Key).To(Equal("0"))
			})

			It("Should return an error for an invalid key", func() {
				_, err := svc.RetrieveResource(ctx, "invalid", nil)
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(&strconv.NumError{}))
			})

			It("Should return an error for a non-existent node", func() {
				_, err := svc.RetrieveResource(ctx, "999", nil)
				Expect(err).To(HaveOccurredAs(aspen.ErrNodeNotFound))
				Expect(err.Error()).To(ContainSubstring("999"))
			})
		})

		Describe("OpenNexter", func() {
			It("Should iterate over all nodes", func() {
				seq, closer := MustSucceed2(svc.OpenNexter(ctx))
				defer closer.Close()
				resources := slices.Collect(seq)
				Expect(len(resources)).To(BeNumerically(">=", 3))
				for _, r := range resources {
					Expect(r.ID.Type).To(Equal(node.OntologyType))
				}
			})
		})

		Describe("OnChange", func() {
			It("Should propagate cluster topology changes", func() {
				changes := make(chan []ontology.Change, 5)
				dc := svc.OnChange(func(
					ctx context.Context,
					seq iter.Seq[ontology.Change],
				) {
					changes <- slices.Collect(seq)
				})
				defer dc()
				mockCluster.Provision(ctx)
				Eventually(func(g Gomega) {
					c := <-changes
					g.Expect(len(c)).To(BeNumerically(">", 0))
					g.Expect(c[0].Variant).To(Equal(change.VariantSet))
					g.Expect(c[0].Key.Type).To(Equal(node.OntologyType))
				}).Should(Succeed())
			})
		})
	})
})

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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// openTestService spins up a fresh ontology against the supplied cluster and returns a
// node Service registered with it. Test fixtures use this instead of the mock cluster's
// pre-registered services so that the test owns the ontology lifecycle and avoids
// duplicate-registration panics.
func openTestService(
	ctx context.Context,
	c cluster.Cluster,
) (*node.Service, *ontology.Ontology) {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	svc := MustSucceed(node.NewService(ctx, node.ServiceConfig{
		Cluster:  c,
		Ontology: otg,
	}))
	return svc, otg
}

var (
	testCluster *testutil.Cluster
	testSvc     *node.Service
	testOtg     *ontology.Ontology
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	testCluster = testutil.NewCluster(ctx, 2)
	testSvc, testOtg = openTestService(ctx, testCluster.Nodes[1].Cluster)
})

var _ = Describe("Ontology", func() {
	Describe("ResourceID", func() {
		It(
			"Should construct an ID with the node resource type and a stringified key",
			func() {
				id := node.ResourceID(7)
				Expect(id.Type).To(Equal(resource.TypeNode))
				Expect(id.Key).To(Equal("7"))
			},
		)

		It("Should round-trip the free node key", func() {
			id := node.ResourceID(node.KeyFree)
			Expect(id.Key).To(Equal(strconv.Itoa(int(node.KeyFree))))
		})
	})

	Describe("Service", func() {
		Describe("Type", func() {
			It("Should report the node ontology resource type", func() {
				Expect(testSvc.Type()).To(Equal(resource.TypeNode))
			})
		})

		Describe("RetrieveResource", func() {
			It("Should return a resource for the host node", func(ctx SpecContext) {
				host := testCluster.Nodes[1].Cluster.HostKey()
				res := MustSucceed(
					testSvc.RetrieveResource(ctx, strconv.Itoa(int(host)), nil),
				)
				Expect(res.ID).To(Equal(node.ResourceID(host)))
				Expect(res.Name).To(Equal("Node 1"))
			})

			It(
				"Should return a synthetic resource for the free node key",
				func(ctx SpecContext) {
					res := MustSucceed(testSvc.RetrieveResource(
						ctx,
						strconv.Itoa(int(node.KeyFree)),
						nil,
					))
					Expect(res.ID).To(Equal(node.ResourceID(node.KeyFree)))
				},
			)

			It(
				"Should return ErrNodeNotFound for a key that is not in the cluster",
				func(ctx SpecContext) {
					Expect(testSvc.RetrieveResource(ctx, "999", nil)).
						Error().To(MatchError(aspen.ErrNodeNotFound))
				},
			)

			It("Should return an error for a non-numeric key", func(ctx SpecContext) {
				Expect(testSvc.RetrieveResource(ctx, "not-a-number", nil)).
					Error().To(MatchError(ContainSubstring("invalid syntax")))
			})
		})

		Describe("Search", func() {
			DescribeTable("Should search the nodes currently in the cluster",
				func(ctx SpecContext, term string, limit int, expected []resource.ID) {
					q := search.Query[resource.ID]{Term: term, Limit: limit}
					Expect(
						MustSucceed(testSvc.Search(ctx, nil, q)).Keys,
					).To(Equal(expected))
				},
				Entry("every node", "node", 10,
					[]resource.ID{node.ResourceID(1), node.ResourceID(2)}),
				Entry("the closest node first", "node 2", 10,
					[]resource.ID{node.ResourceID(2), node.ResourceID(1)}),
				Entry("at most the limit", "node", 1,
					[]resource.ID{node.ResourceID(1)}),
				Entry("no match", "valve", 10, []resource.ID{}),
			)
		})

		Describe("OnChange", func() {
			It(
				"Should translate cluster changes into ontology changes when a node joins",
				func(ctx SpecContext) {
					ephemeral := testutil.NewCluster(ctx, 1)
					ephemeralSvc, _ := openTestService(ctx, ephemeral.Nodes[1].Cluster)

					received := make(chan ontology.Change, 8)
					disconnect := ephemeralSvc.OnChange(
						func(_ context.Context, changes iter.Seq[ontology.Change]) {
							for ch := range changes {
								received <- ch
							}
						},
					)
					DeferCleanup(disconnect)

					newNode := ephemeral.Provision(ctx)
					newKey := newNode.Cluster.HostKey()

					var change ontology.Change
					Eventually(received).Should(Receive(&change))
					Expect(change.Key).To(Equal(node.ResourceID(newKey).String()))
				},
			)
		})
	})
})

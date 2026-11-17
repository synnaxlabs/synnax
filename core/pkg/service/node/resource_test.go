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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each node by its key", func(ctx SpecContext) {
		ids := []resource.ID{node.ResourceID(node.KeyFree), node.ResourceID(1)}
		Expect(
			testSvc.Resources().Retrieve(ctx, nil, ids),
		).To(Equal([]resource.Resource{
			{ID: ids[0], Name: "Node 4095"},
			{ID: ids[1], Name: "Node 1"},
		}))
	})

	It("Should leave out a node that is not the host", func(ctx SpecContext) {
		Expect(testSvc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{node.ResourceID(999), node.ResourceID(1)},
		)).To(Equal([]resource.Resource{{ID: node.ResourceID(1), Name: "Node 1"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(testSvc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeNode, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid node key "pump"`)),
		))
	})

	It("Should return a validation error for a key with a leading zero", func(
		ctx SpecContext,
	) {
		Expect(testSvc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeNode, Key: "01"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid node key "01"`)),
		))
	})

	DescribeTable("Should list the host node",
		func(
			ctx SpecContext,
			limit int,
			except func(resource.ID) bool,
			expected []resource.Resource,
		) {
			Expect(testSvc.Resources().List(ctx, nil, limit, except)).
				To(Equal(expected))
		},
		Entry("the host", 10, nil, []resource.Resource{
			{ID: node.ResourceID(1), Name: "Node 1"},
		}),
		Entry("nothing at a zero limit", 0, nil, []resource.Resource{}),
		Entry(
			"nothing when the host is excepted",
			10,
			func(id resource.ID) bool { return id == node.ResourceID(1) },
			[]resource.Resource{},
		),
	)
})

var _ = Describe("Resource", func() {
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
		Describe("Search", func() {
			DescribeTable("Should search the host node",
				func(ctx SpecContext, term string, limit int, expected []resource.ID) {
					q := search.Query[resource.ID]{Term: term, Limit: limit}
					Expect(
						MustSucceed(testSvc.Resources().Search(ctx, nil, q)).Keys,
					).To(Equal(expected))
				},
				Entry("the host", "node", 10, []resource.ID{node.ResourceID(1)}),
				Entry("the host by every word", "node 1", 10,
					[]resource.ID{node.ResourceID(1)}),
				Entry("no match", "valve", 10, []resource.ID{}),
			)
		})
	})
})

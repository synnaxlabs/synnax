// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraint_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Logical", func() {
	var (
		c                 constraint.Constraint
		allCoversAll      constraint.Constraint
		channelOnlyCover  constraint.Constraint
		rangeOnlyCover    constraint.Constraint
		coversNothing     constraint.Constraint
		invalidConstraint constraint.Constraint
	)
	BeforeEach(func() {
		// Set up request with multiple objects
		params.Request.Objects = []ontology.ID{
			{Type: "channel", Key: "1"},
			{Type: "channel", Key: "2"},
			{Type: "range", Key: "1"},
		}
		params.Request.Action = access.ActionRetrieve

		// Constraint that covers all objects (action is in list)
		allCoversAll = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: constraint.OpIsIn,
			Actions:  set.New(access.ActionRetrieve),
		}
		// Constraint that covers only channel objects
		channelOnlyCover = constraint.Constraint{
			Kind:     constraint.KindMatch,
			Operator: constraint.OpContainsAny,
			IDs:      []ontology.ID{{Type: "channel"}},
		}
		// Constraint that covers only range objects
		rangeOnlyCover = constraint.Constraint{
			Kind:     constraint.KindMatch,
			Operator: constraint.OpContainsAny,
			IDs:      []ontology.ID{{Type: "range"}},
		}
		// Constraint that covers nothing (action not in list)
		coversNothing = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: constraint.OpIsIn,
			Actions:  set.New(access.ActionDelete),
		}
		invalidConstraint = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: "invalid",
			Actions:  set.New(access.ActionRetrieve),
		}
		c = constraint.Constraint{Kind: constraint.KindLogical}
	})

	Describe("OpContainsAny (OR) - Union", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return union of child results", func() {
			c.Constraints = []constraint.Constraint{channelOnlyCover, rangeOnlyCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(
				ontology.ID{Type: "channel", Key: "1"},
				ontology.ID{Type: "channel", Key: "2"},
				ontology.ID{Type: "range", Key: "1"},
			))
		})
		It("Should return results from single matching child", func() {
			c.Constraints = []constraint.Constraint{channelOnlyCover, coversNothing}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(
				ontology.ID{Type: "channel", Key: "1"},
				ontology.ID{Type: "channel", Key: "2"},
			))
		})
		It("Should return empty if no child covers any objects", func() {
			c.Constraints = []constraint.Constraint{coversNothing, coversNothing}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if any child covers all", func() {
			c.Constraints = []constraint.Constraint{allCoversAll, coversNothing}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(params.Request.Objects))
		})
		It("Should return all objects if the constraints list is empty", func() {
			c.Constraints = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{coversNothing, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})

	Describe("OpContainsAll (AND) - Intersection", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return intersection of child results", func() {
			// Both allCoversAll and channelOnlyCover - intersection is channels only
			c.Constraints = []constraint.Constraint{allCoversAll, channelOnlyCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(
				ontology.ID{Type: "channel", Key: "1"},
				ontology.ID{Type: "channel", Key: "2"},
			))
		})
		It("Should return empty if any child returns empty", func() {
			c.Constraints = []constraint.Constraint{allCoversAll, coversNothing}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return empty if children have no overlap", func() {
			c.Constraints = []constraint.Constraint{channelOnlyCover, rangeOnlyCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if all children cover all", func() {
			anotherAllCover := constraint.Constraint{
				Kind:     constraint.KindAction,
				Operator: constraint.OpIsIn,
				Actions:  set.New(access.ActionRetrieve, access.ActionUpdate),
			}
			c.Constraints = []constraint.Constraint{allCoversAll, anotherAllCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(params.Request.Objects))
		})
		It("Should return all objects if the constraints list is empty", func() {
			c.Constraints = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should short-circuit on first empty result", func() {
			c.Constraints = []constraint.Constraint{coversNothing, invalidConstraint}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{allCoversAll, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})

	Describe("OpContainsNone - Objects in no child results", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return objects that appear in no child results", func() {
			c.Constraints = []constraint.Constraint{channelOnlyCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(ontology.ID{Type: "range", Key: "1"}))
		})
		It("Should return empty if all objects appear in some child result", func() {
			c.Constraints = []constraint.Constraint{channelOnlyCover, rangeOnlyCover}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if no child covers anything", func() {
			c.Constraints = []constraint.Constraint{coversNothing, coversNothing}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(params.Request.Objects))
		})
		It("Should return all objects if the constraints list is empty", func() {
			c.Constraints = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{coversNothing, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})

	Describe("Nested Logical Constraints", func() {
		It("Should correctly evaluate nested AND within OR", func() {
			// Inner AND: channels AND action (intersection is channels)
			innerAnd := constraint.Constraint{
				Kind:        constraint.KindLogical,
				Operator:    constraint.OpContainsAll,
				Constraints: []constraint.Constraint{channelOnlyCover, allCoversAll},
			}
			// Outer OR: ranges OR (channels AND action) = ranges + channels
			c.Operator = constraint.OpContainsAny
			c.Constraints = []constraint.Constraint{rangeOnlyCover, innerAnd}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(params.Request.Objects))
		})
		It("Should correctly evaluate nested OR within AND", func() {
			// Inner OR: channels OR ranges (union is all objects)
			innerOr := constraint.Constraint{
				Kind:        constraint.KindLogical,
				Operator:    constraint.OpContainsAny,
				Constraints: []constraint.Constraint{channelOnlyCover, rangeOnlyCover},
			}
			// Outer AND: all AND (channels OR ranges) = intersection of all with all = all
			c.Operator = constraint.OpContainsAll
			c.Constraints = []constraint.Constraint{allCoversAll, innerOr}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(params.Request.Objects))
		})
	})

	Describe("Invalid Operator", func() {
		It("Should return ErrInvalidOperator if the operator is invalid", func() {
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Logical", func() {
	var c, trueConstraint, falseConstraint, invalidConstraint constraint.Constraint
	BeforeEach(func() {
		trueConstraint = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: constraint.OpIsIn,
			Actions:  set.New(access.ActionRetrieve),
		}
		falseConstraint = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: constraint.OpIsIn,
			Actions:  set.New(access.ActionDelete),
		}
		invalidConstraint = constraint.Constraint{
			Kind:     constraint.KindAction,
			Operator: "invalid",
			Actions:  set.New(access.ActionRetrieve),
		}
		params.Request.Action = access.ActionRetrieve
		c = constraint.Constraint{Kind: constraint.KindLogical}
	})
	Describe("OpContainsAny (OR)", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return true if any child constraint is satisfied", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, falseConstraint}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if no child constraint is satisfied", func() {
			c.Constraints = []constraint.Constraint{falseConstraint, falseConstraint}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if all child constraints are satisfied", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, trueConstraint}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should short-circuit on first true constraint", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return true if the constraints list is empty", func() {
			c.Constraints = nil
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})

		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{falseConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
	Describe("OpContainsAll (AND)", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return true if all child constraints are satisfied", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, trueConstraint}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if any child constraint is not satisfied", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, falseConstraint}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should short-circuit on first false constraint", func() {
			c.Constraints = []constraint.Constraint{falseConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraints list is empty", func() {
			c.Constraints = nil
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
	Describe("OpContainsNone (NOT)", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return true if no child constraint is satisfied", func() {
			c.Constraints = []constraint.Constraint{falseConstraint, falseConstraint}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if any child constraint is satisfied", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, falseConstraint}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should short-circuit on first true constraint", func() {
			c.Constraints = []constraint.Constraint{trueConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraints list is empty", func() {
			c.Constraints = nil
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should propagate errors from child constraints", func() {
			c.Constraints = []constraint.Constraint{falseConstraint, invalidConstraint}
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
	Describe("Nested Logical Constraints", func() {
		It("Should correctly evaluate nested AND within OR", func() {
			innerAnd := constraint.Constraint{
				Kind:        constraint.KindLogical,
				Operator:    constraint.OpContainsAll,
				Constraints: []constraint.Constraint{trueConstraint, trueConstraint},
			}
			c.Operator = constraint.OpContainsAny
			c.Constraints = []constraint.Constraint{falseConstraint, innerAnd}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should correctly evaluate nested OR within AND", func() {
			innerOr := constraint.Constraint{
				Kind:        constraint.KindLogical,
				Operator:    constraint.OpContainsAny,
				Constraints: []constraint.Constraint{trueConstraint, falseConstraint},
			}
			c.Operator = constraint.OpContainsAll
			c.Constraints = []constraint.Constraint{trueConstraint, innerOr}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
	})
	Describe("Invalid Operator", func() {
		It("Should return ErrInvalidOperator if the operator is invalid", func() {
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
})

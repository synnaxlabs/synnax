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

var _ = Describe("Action", func() {
	var c constraint.Constraint
	BeforeEach(func() {
		c = constraint.Constraint{
			Kind:    constraint.KindAction,
			Actions: set.New(access.ActionRetrieve, access.ActionUpdate),
		}
	})
	Describe("OpIsIn", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpIsIn
		})
		It("Should return true if the action is in the list", func() {
			params.Request.Action = access.ActionRetrieve
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the action is not in the list", func() {
			params.Request.Action = access.ActionCreate
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should not panic if the constraint's actions list is nil", func() {
			c.Actions = nil
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should not panic if the request's action is empty", func() {
			params.Request.Action = ""
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpIsNotIn", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpIsNotIn
		})
		It("Should return true if the action is not in the list", func() {
			params.Request.Action = access.ActionCreate
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the action is in the list", func() {
			params.Request.Action = access.ActionRetrieve
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should not panic if the constraint's actions list is nil", func() {
			c.Actions = nil
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's action is empty", func() {
			params.Request.Action = ""
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

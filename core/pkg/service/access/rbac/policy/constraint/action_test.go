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

var _ = Describe("Action", func() {
	var c constraint.Constraint
	BeforeEach(func() {
		c = constraint.Constraint{
			Kind:    constraint.KindAction,
			Actions: set.New(access.ActionRetrieve, access.ActionUpdate),
		}
		// Set up some objects for the request
		params.Request.Objects = []ontology.ID{
			{Type: "channel", Key: "1"},
			{Type: "channel", Key: "2"},
		}
	})
	Describe("OpIsIn", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpIsIn
		})
		It("Should return all objects if the action is in the list", func() {
			params.Request.Action = access.ActionRetrieve
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the action is not in the list", func() {
			params.Request.Action = access.ActionCreate
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return empty if the constraint's actions list is nil", func() {
			c.Actions = nil
			params.Request.Action = access.ActionRetrieve
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return empty if the request's action is empty", func() {
			params.Request.Action = ""
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
	})
	Describe("OpIsNotIn", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpIsNotIn
		})
		It("Should return all objects if the action is not in the list", func() {
			params.Request.Action = access.ActionCreate
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the action is in the list", func() {
			params.Request.Action = access.ActionRetrieve
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if the constraint's actions list is nil", func() {
			c.Actions = nil
			params.Request.Action = access.ActionCreate
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return all objects if the request's action is empty", func() {
			params.Request.Action = ""
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
	})
	Describe("Invalid Operator", func() {
		It("Should return ErrInvalidOperator if the operator is invalid", func() {
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
})

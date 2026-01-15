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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/set"
)

var _ = Describe("Properties", func() {
	var c constraint.Constraint
	BeforeEach(func() {
		c = constraint.Constraint{
			Kind:       constraint.KindProperties,
			Properties: []string{"name", "description"},
		}
	})
	Describe("OpContainsAny", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return true if the request contains any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request contains none of the properties", func() {
			params.Request.Properties = set.New("other", "another")
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if all request properties match", func() {
			params.Request.Properties = set.New("name", "description")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return true if the constraint's properties list is empty", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's properties set is nil", func() {
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsAll", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return true if the request contains all of the properties", func() {
			params.Request.Properties = set.New("name", "description", "other")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request is missing any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if request properties exactly match", func() {
			params.Request.Properties = set.New("name", "description")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return true if the constraint's properties list is empty", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's properties set is empty", func() {
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsNone", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return true if the request contains none of the properties", func() {
			params.Request.Properties = set.New("other", "another")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request contains any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return false if all request properties match", func() {
			params.Request.Properties = set.New("name", "description")
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should not panic if the constraint's properties list is nil", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's properties set is empty", func() {
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

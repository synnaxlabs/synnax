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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/constraint"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Properties", func() {
	var c constraint.Constraint
	BeforeEach(func() {
		c = constraint.Constraint{
			Kind:       constraint.KindProperties,
			Properties: []string{"name", "description"},
		}
		// Set up some objects for the request
		params.Request.Objects = []ontology.ID{
			{Type: "channel", Key: "1"},
			{Type: "channel", Key: "2"},
		}
	})
	Describe("OpContainsAny", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return all objects if the request contains any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request contains none of the properties", func() {
			params.Request.Properties = set.New("other", "another")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if all request properties match", func() {
			params.Request.Properties = set.New("name", "description")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return all objects if the constraint's properties list is empty", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request's properties set is nil", func() {
			params.Request.Properties = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
	})
	Describe("OpContainsAll", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return all objects if the request contains all of the properties", func() {
			params.Request.Properties = set.New("name", "description", "other")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request is missing any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if request properties exactly match", func() {
			params.Request.Properties = set.New("name", "description")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return all objects if the constraint's properties list is empty", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request's properties set is empty", func() {
			params.Request.Properties = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
	})
	Describe("OpContainsNone", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return all objects if the request contains none of the properties", func() {
			params.Request.Properties = set.New("other", "another")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request contains any of the properties", func() {
			params.Request.Properties = set.New("name", "other")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return empty if all request properties match", func() {
			params.Request.Properties = set.New("name", "description")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if the constraint's properties list is nil", func() {
			c.Properties = nil
			params.Request.Properties = set.New("name")
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return all objects if the request's properties set is empty", func() {
			params.Request.Properties = nil
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

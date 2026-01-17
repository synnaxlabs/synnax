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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Match", func() {
	var c constraint.Constraint
	BeforeEach(func() {
		c = constraint.Constraint{
			Kind: constraint.KindMatch,
			IDs: []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
			},
		}
	})
	Describe("OpContainsAny", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return matching objects when request contains some constraint IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(ontology.ID{Type: "channel", Key: "1"}))
		})
		It("Should return empty when request contains none of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "99"},
				{Type: "channel", Key: "100"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(
				ontology.ID{Type: "channel", Key: "99"},
				ontology.ID{Type: "channel", Key: "100"},
			))
		})
		It("Should not match type-only when request has different type", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request's objects list is empty", func() {
			params.Request.Objects = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return multiple matching objects", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(
				ontology.ID{Type: "channel", Key: "1"},
				ontology.ID{Type: "channel", Key: "2"},
			))
		})
	})
	Describe("OpContainsAll", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return all objects if the request contains all constraint IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request is missing any constraint ID", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}, {Type: "range"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "2"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty when missing a required type", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{{Type: "range", Key: "2"}}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return all objects if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if the request's objects list is empty", func() {
			params.Request.Objects = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
	})
	Describe("OpContainsNone", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return objects that don't match any constraint IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(ontology.ID{Type: "range", Key: "1"}))
		})
		It("Should exclude objects that match constraint IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(ontology.ID{Type: "range", Key: "1"}))
		})
		It("Should exclude by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "99"},
				{Type: "range", Key: "1"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(ConsistOf(ontology.ID{Type: "range", Key: "1"}))
		})
		It("Should return all objects if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(Equal(params.Request.Objects))
		})
		It("Should return empty if request's objects list is empty", func() {
			params.Request.Objects = nil
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
		It("Should return empty if all objects match constraint IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
			}
			covered := MustSucceed(c.Enforce(ctx, params))
			Expect(covered).To(BeEmpty())
		})
	})
	Describe("Invalid Operator", func() {
		It("Should return ErrInvalidOperator if the operator is invalid", func() {
			Expect(c.Enforce(ctx, params)).Error().
				To(MatchError(constraint.ErrInvalidOperator))
		})
	})
})

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
		It("Should return true if the request contains any of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request contains none of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "99"},
			}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not match type-only when request has different type", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			params.Request.Objects = nil
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsAll", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
		})
		It("Should return true if the request contains all of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "channel", Key: "2"},
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request is missing any of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}, {Type: "range"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "2"},
			}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should match by type for failing constraints when constraint ID is type-only", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{{Type: "range", Key: "2"}}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			params.Request.Objects = nil
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsNone", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return true if the request contains none of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the request contains any of the IDs", func() {
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "1"},
				{Type: "range", Key: "1"},
			}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: "channel"}}
			params.Request.Objects = []ontology.ID{
				{Type: "channel", Key: "99"},
			}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraint's IDs list is empty", func() {
			c.IDs = nil
			params.Request.Objects = []ontology.ID{{Type: "channel", Key: "1"}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			params.Request.Objects = nil
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

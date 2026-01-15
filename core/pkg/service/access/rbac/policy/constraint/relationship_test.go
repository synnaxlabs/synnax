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

const testType1 ontology.Type = "test1"
const testType2 ontology.Type = "test2"

// testRelType is a custom relationship type for testing where the requested object
// is the "From" side of the relationship, matching the traverser filter in
// relationship.go.
const testRelType ontology.RelationshipType = "related_to"

func newTestID1(key string) ontology.ID {
	return ontology.ID{Key: key, Type: testType1}
}

func newTestID2(key string) ontology.ID {
	return ontology.ID{Key: key, Type: testType2}
}

var _ = Describe("Relationship", func() {
	var (
		c          constraint.Constraint
		w          ontology.Writer
		requested  ontology.ID
		related1   ontology.ID
		related2   ontology.ID
		notRelated ontology.ID
	)
	BeforeEach(func() {
		w = otg.NewWriter(tx)
		// The traverser filter in relationship.go uses rel.From == res.ID with backward
		// direction, so the requested object must be the "From" side of the relationship.
		requested = newTestID1("requested")
		related1 = newTestID2("related1")
		related2 = newTestID2("related2")
		notRelated = newTestID1("not_related")
		Expect(w.DefineManyResources(ctx,
			[]ontology.ID{requested, related1, related2, notRelated})).To(Succeed())
		// requested -> testRelType -> related1
		// requested -> testRelType -> related2
		Expect(w.DefineFromOneToManyRelationships(
			ctx,
			requested,
			testRelType,
			[]ontology.ID{related1, related2},
		)).To(Succeed())
		c = constraint.Constraint{
			Kind:             constraint.KindRelationship,
			RelationshipType: testRelType,
		}
		params.Request.Objects = []ontology.ID{requested}
	})
	Describe("OpContainsAny", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAny
		})
		It("Should return true if the related resource matches any of the IDs", func() {
			c.IDs = []ontology.ID{related1, notRelated}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the related resource does not match any of the IDs", func() {
			c.IDs = []ontology.ID{notRelated}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: testType2}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not match type-only when related resource has different type", func() {
			c.IDs = []ontology.ID{{Type: testType1}}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return false if the request has no related resources", func() {
			unrelated := newTestID1("unrelated")
			Expect(w.DefineResource(ctx, unrelated)).To(Succeed())
			params.Request.Objects = []ontology.ID{unrelated}
			c.IDs = []ontology.ID{related1}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraint's IDs list is empty", func() {
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			params.Request.Objects = nil
			c.IDs = []ontology.ID{related1}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsAll", func() {
		var requested2, related2 ontology.ID
		BeforeEach(func() {
			c.Operator = constraint.OpContainsAll
			requested2 = newTestID1("requested2")
			related2 = newTestID2("related2")
			Expect(w.DefineResource(ctx, requested2)).To(Succeed())
			Expect(w.DefineResource(ctx, related2)).To(Succeed())
			Expect(w.DefineRelationship(ctx, requested2, testRelType, related2)).To(Succeed())
		})
		AfterEach(func() {
			Expect(w.DeleteResource(ctx, requested2)).To(Succeed())
			Expect(w.DeleteResource(ctx, related2)).To(Succeed())
		})
		It("Should return true if the related resources contain all of the IDs", func() {
			c.IDs = []ontology.ID{related1}
			params.Request.Objects = []ontology.ID{requested}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return true when requesting multiple objects with all related in constraint", func() {
			c.IDs = []ontology.ID{related1, related2}
			params.Request.Objects = []ontology.ID{requested, requested2}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the related resources are missing any of the IDs", func() {
			c.IDs = []ontology.ID{related1, newTestID1("missing")}
			params.Request.Objects = []ontology.ID{requested}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: testType2}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the related resources have a different type", func() {
			c.IDs = []ontology.ID{{Type: "newTestType"}}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true if the constraint's IDs list is nil", func() {
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			params.Request.Objects = nil
			c.IDs = []ontology.ID{related1}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
	})
	Describe("OpContainsNone", func() {
		BeforeEach(func() {
			c.Operator = constraint.OpContainsNone
		})
		It("Should return true if the related resources contain none of the IDs", func() {
			c.IDs = []ontology.ID{notRelated}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should return false if the related resources contain any of the IDs", func() {
			c.IDs = []ontology.ID{related1}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should match by type when the constraint ID is a type-only ID", func() {
			c.IDs = []ontology.ID{{Type: testType2}}
			Expect(c.Enforce(ctx, params)).To(BeFalse())
		})
		It("Should return true when type does not match", func() {
			c.IDs = []ontology.ID{{Type: testType1}}
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the constraint's IDs list is nil", func() {
			Expect(c.Enforce(ctx, params)).To(BeTrue())
		})
		It("Should not panic if the request's objects list is empty", func() {
			c.IDs = []ontology.ID{related1}
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

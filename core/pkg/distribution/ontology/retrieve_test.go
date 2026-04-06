// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/encoding/orc"
)

var _ = Describe("Retrieve", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriter(tx) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its Name", func(ctx SpecContext) {
			id := newSampleType("A")
			Expect(w.DefineResource(ctx, id)).To(Succeed())
			var r ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(id).
				Entry(&r).
				Exec(ctx, tx),
			).To(Succeed())
			var res Sample
			Expect(r.Parse(&res)).To(Succeed())
			Expect(res.Key).To(Equal("A"))
		})
		It("Should retrieve multiple resources by their Name", func(ctx SpecContext) {
			ids := []ontology.ID{newSampleType("A"), newSampleType("B")}
			Expect(w.DefineResource(ctx, ids[0])).To(Succeed())
			Expect(w.DefineResource(ctx, ids[1])).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(ids...).
				Entries(&r).
				Exec(ctx, nil),
			).To(Succeed())
			var res Sample
			Expect(r[0].Parse(&res)).To(Succeed())
			Expect(res.Key).To(Equal("A"))
			Expect(r[1].Parse(&res)).To(Succeed())
			Expect(res.Key).To(Equal("B"))
		})
	})
	Describe("Multi Clause", func() {
		Describe("Parental Traversal", func() {

			It("Should retrieve the parent of a resource", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&r).
					Exec(ctx, nil),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("B"))
			})

			It("Should retrieve the parents of multiple resources", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&r).
					Exec(ctx, tx),
				).To(Succeed())
				var res Sample
				Expect(r[0].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("B"))
				Expect(r[1].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("C"))
			})

			It("Should retrieve the grandparents of a resource", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.ChildrenTraverser).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&r).
					Exec(ctx, tx),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("C"))
			})

			It("Should retrieve resources at intermediate and final clauses", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var intermediate []ontology.Resource
				var final []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&intermediate).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&final).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(intermediate).To(HaveLen(1))
				Expect(final).To(HaveLen(1))
				var res Sample
				Expect(intermediate[0].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("B"))
				Expect(final[0].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("C"))
			})

			It("Should retrieve the parent of a resource using ParentsTraverser", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(b).
					TraverseTo(ontology.ParentsTraverser).
					Entry(&r).
					Exec(ctx, nil),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("A"))
			})

			It("Should retrieve multiple parents of a resource using ParentsTraverser", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(c).
					TraverseTo(ontology.ParentsTraverser).
					Entries(&r).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(r).To(HaveLen(2))
				keys := lo.Map(r, func(res ontology.Resource, _ int) string {
					var s Sample
					lo.Must0(res.Parse(&s))
					return s.Key
				})
				Expect(keys).To(ConsistOf("A", "B"))
			})

			It("Should traverse grandparents using ParentsTraverser", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(c).
					TraverseTo(ontology.ParentsTraverser).
					TraverseTo(ontology.ParentsTraverser).
					Entry(&r).
					Exec(ctx, tx),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("A"))
			})

			It("Should retrieve the resources of a parent by their type", func(ctx SpecContext) {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.ChildrenTraverser).
					WhereTypes(sampleOntologyType).
					Entries(&r).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(len(r)).To(Equal(2))
			})
		})
	})

	Describe("Multi-hop traversal with intermediate filters", func() {
		// Regression test for issue where intermediate clauses with filters
		// but no bound entries would silently drop results because gorp.NewRetrieve
		// created unbound entries that couldn't store query results.
		It("Should traverse through intermediate filtered clauses without bound entries", func(ctx SpecContext) {
			// Create a hierarchy: grandparent -> parent -> child
			// We'll query: grandparent -> TraverseTo(Children) -> WhereTypes(sample) -> TraverseTo(Children) -> Entries
			grandparent := newSampleType("grandparent")
			parent := newSampleType("parent")
			child := newSampleType("child")

			Expect(w.DefineResource(ctx, grandparent)).To(Succeed())
			Expect(w.DefineResource(ctx, parent)).To(Succeed())
			Expect(w.DefineResource(ctx, child)).To(Succeed())
			Expect(w.DefineRelationship(ctx, grandparent, ontology.RelationshipTypeParentOf, parent)).To(Succeed())
			Expect(w.DefineRelationship(ctx, parent, ontology.RelationshipTypeParentOf, child)).To(Succeed())

			// This query pattern has an intermediate clause (WhereTypes) with no bound entries
			var results []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(grandparent).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(sampleOntologyType). // Intermediate filter, no Entries() bound
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(sampleOntologyType).
				Entries(&results). // Only final clause has entries
				Exec(ctx, tx),
			).To(Succeed())

			Expect(results).To(HaveLen(1))
			var res Sample
			Expect(results[0].Parse(&res)).To(Succeed())
			Expect(res.Key).To(Equal("child"))
		})

		It("Should traverse through multiple intermediate filtered clauses", func(ctx SpecContext) {
			// Create a 4-level hierarchy: A -> B -> C -> D
			a := newSampleType("level-A")
			b := newSampleType("level-B")
			c := newSampleType("level-C")
			d := newSampleType("level-D")

			Expect(w.DefineResource(ctx, a)).To(Succeed())
			Expect(w.DefineResource(ctx, b)).To(Succeed())
			Expect(w.DefineResource(ctx, c)).To(Succeed())
			Expect(w.DefineResource(ctx, d)).To(Succeed())
			Expect(w.DefineRelationship(ctx, a, ontology.RelationshipTypeParentOf, b)).To(Succeed())
			Expect(w.DefineRelationship(ctx, b, ontology.RelationshipTypeParentOf, c)).To(Succeed())
			Expect(w.DefineRelationship(ctx, c, ontology.RelationshipTypeParentOf, d)).To(Succeed())

			// Multiple intermediate clauses with filters, no bound entries
			var results []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(a).
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(sampleOntologyType). // Intermediate filter #1
				TraverseTo(ontology.ChildrenTraverser).
				WhereTypes(sampleOntologyType). // Intermediate filter #2
				TraverseTo(ontology.ChildrenTraverser).
				Entries(&results). // Only final clause has entries
				Exec(ctx, tx),
			).To(Succeed())

			Expect(results).To(HaveLen(1))
			var res Sample
			Expect(results[0].Parse(&res)).To(Succeed())
			Expect(res.Key).To(Equal("level-D"))
		})
	})

	Describe("Limit + Offset", func() {
		It("Should page through resources in order", func(ctx SpecContext) {
			ids := make([]ontology.ID, 10)
			for i := range ids {
				Expect(w.DefineResource(ctx, newSampleType(strconv.Itoa(i)))).To(Succeed())
			}
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				Offset(0).
				Limit(5).
				Entries(&r).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(r)).To(Equal(5))
			var r2 []ontology.Resource
			Expect(w.NewRetrieve().
				Offset(5).
				Limit(5).
				Entries(&r2).
				Exec(ctx, tx)).To(Succeed())
			Expect(len(r2)).To(Equal(5))
			mapKeys := func(o ontology.Resource, _ int) string {
				return o.ID.String()
			}
			r1Keys := lo.Map(r, mapKeys)
			r2Keys := lo.Map(r2, mapKeys)
			Expect(lo.Intersect(r1Keys, r2Keys)).To(BeEmpty())
		})
	})

	Describe("WhereTypes", func() {
		It("Should retrieve resources of a single type using prefix matching", func(ctx SpecContext) {
			a := newSampleType("type-filter-A")
			b := newSampleType("type-filter-B")
			Expect(w.DefineResource(ctx, a)).To(Succeed())
			Expect(w.DefineResource(ctx, b)).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(sampleOntologyType).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(len(r)).To(BeNumerically(">=", 2))
			types := lo.Map(r, func(res ontology.Resource, _ int) ontology.ResourceType {
				return res.ID.Type
			})
			for _, t := range types {
				Expect(t).To(Equal(sampleOntologyType))
			}
		})

		It("Should return empty results when filtering by non-existent type", func(ctx SpecContext) {
			Expect(w.DefineResource(ctx, newSampleType("type-filter-C"))).To(Succeed())
			nonExistentType := ontology.ResourceType("nonexistent")
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(nonExistentType).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(r).To(BeEmpty())
		})

		It("Should retrieve resources matching any of multiple types using filter function", func(ctx SpecContext) {
			a := newSampleType("multi-type-A")
			b := newSampleType("multi-type-B")
			Expect(w.DefineResource(ctx, a)).To(Succeed())
			Expect(w.DefineResource(ctx, b)).To(Succeed())
			otherType := ontology.ResourceType("other")
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(sampleOntologyType, otherType).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(len(r)).To(BeNumerically(">=", 2))
			for _, res := range r {
				Expect(res.ID.Type).To(BeElementOf(sampleOntologyType, otherType))
			}
		})

		It("Should return empty when none of the multiple types match", func(ctx SpecContext) {
			Expect(w.DefineResource(ctx, newSampleType("multi-type-none"))).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(ontology.ResourceType("foo"), ontology.ResourceType("bar")).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(r).To(BeEmpty())
		})

		It("Should combine WhereTypes with WhereIDs", func(ctx SpecContext) {
			a := newSampleType("combined-A")
			b := newSampleType("combined-B")
			Expect(w.DefineResource(ctx, a)).To(Succeed())
			Expect(w.DefineResource(ctx, b)).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(a, b).
				WhereTypes(sampleOntologyType).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(len(r)).To(Equal(2))
			ids := lo.Map(r, func(res ontology.Resource, _ int) ontology.ID {
				return res.ID
			})
			Expect(ids).To(ContainElements(a, b))
		})

		It("Should filter out resources when WhereIDs and WhereTypes don't overlap using filter function", func(ctx SpecContext) {
			a := newSampleType("no-overlap-A")
			Expect(w.DefineResource(ctx, a)).To(Succeed())
			var r []ontology.Resource
			// Use multiple types to trigger filter function path (not prefix matching)
			Expect(w.NewRetrieve().
				WhereIDs(a).
				WhereTypes(ontology.ResourceType("different"), ontology.ResourceType("another")).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(r).To(BeEmpty())
		})

		It("Should work with Limit when using single type prefix matching", func(ctx SpecContext) {
			for i := range 5 {
				Expect(w.DefineResource(ctx, newSampleType("limit-type-"+strconv.Itoa(i)))).To(Succeed())
			}
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(sampleOntologyType).
				Limit(3).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(len(r)).To(Equal(3))
		})

		It("Should work with Limit when using multiple types filter", func(ctx SpecContext) {
			for i := range 5 {
				Expect(w.DefineResource(ctx, newSampleType("limit-multi-"+strconv.Itoa(i)))).To(Succeed())
			}
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereTypes(sampleOntologyType, ontology.ResourceType("other")).
				Limit(3).
				Entries(&r).
				Exec(ctx, tx),
			).To(Succeed())
			Expect(len(r)).To(Equal(3))
		})
	})

	Describe("RelationshipPrefix", func() {
		It("Should return a prefix matching the relationship key format", func() {
			prefix := ontology.RelationshipPrefix("parent")
			id := ontology.ID{Type: "channel", Key: "123"}
			Expect(string(prefix(id))).To(Equal("channel:123->parent->"))
		})
		It("Should produce a prefix that matches relationship GorpKeys", func() {
			prefix := ontology.RelationshipPrefix(ontology.RelationshipTypeParentOf)
			from := ontology.ID{Type: "rack", Key: "1"}
			rel := ontology.Relationship{
				From: from,
				Type: ontology.RelationshipTypeParentOf,
				To:   ontology.ID{Type: "device", Key: "2"},
			}
			key := rel.GorpKey()
			p := prefix(from)
			Expect(string(key)).To(HavePrefix(string(p)))
		})
	})

	Describe("ReadRawID", func() {
		It("Should read an ID from orc-encoded bytes", func() {
			w := orc.NewWriter(64)
			w.String("channel")
			w.String("42")
			id := ontology.ReadRawID(orc.Raw(w.Bytes()))
			Expect(id.Type).To(Equal(ontology.ResourceType("channel")))
			Expect(id.Key).To(Equal("42"))
		})

		It("Should read the ID at the current position, ignoring trailing data", func() {
			w := orc.NewWriter(64)
			w.String("device")
			w.String("abc")
			w.String("trailing")
			id := ontology.ReadRawID(orc.Raw(w.Bytes()))
			Expect(id.Type).To(Equal(ontology.ResourceType("device")))
			Expect(id.Key).To(Equal("abc"))
		})

		It("Should not panic on a malformed ID", func() {
			w := orc.NewWriter(1)
			w.String("c")
			id := ontology.ReadRawID(orc.Raw(w.Bytes()))
			Expect(id.Type).To(Equal(ontology.ResourceType("c")))
			Expect(id.Key).To(Equal(""))
		})
	})
})

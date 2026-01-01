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
)

var _ = Describe("retrieveResource", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriter(tx) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its Name", func() {
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
		It("Should retrieve multiple resources by their Name", func() {
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

			It("Should retrieve the parent of a resource", func() {
				a := newSampleType("A")
				b := newSampleType("B")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, b)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					Entry(&r).
					Exec(ctx, nil),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("B"))
			})

			It("Should retrieve the parents of multiple resources", func() {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					Entries(&r).
					Exec(ctx, tx),
				).To(Succeed())
				var res Sample
				Expect(r[0].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("B"))
				Expect(r[1].Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("C"))
			})

			It("Should retrieve the grandparents of a resource", func() {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.ParentOf, c)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					TraverseTo(ontology.Children).
					Entry(&r).
					Exec(ctx, tx),
				).To(Succeed())
				var res Sample
				Expect(r.Parse(&res)).To(Succeed())
				Expect(res.Key).To(Equal("C"))
			})

			It("Should retrieve resources at intermediate and final clauses", func() {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.ParentOf, c)).To(Succeed())
				var intermediate []ontology.Resource
				var final []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					Entries(&intermediate).
					TraverseTo(ontology.Children).
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

			It("Should retrieve the resources of a parent by their type", func() {
				a := newSampleType("A")
				b := newSampleType("B")
				c := newSampleType("C")
				Expect(w.DefineResource(ctx, a)).To(Succeed())
				Expect(w.DefineResource(ctx, b)).To(Succeed())
				Expect(w.DefineResource(ctx, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(ctx, a, ontology.ParentOf, c)).To(Succeed())
				Expect(w.DefineRelationship(ctx, b, ontology.ParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					WhereTypes(sampleType).
					Entries(&r).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(len(r)).To(Equal(2))
			})
		})
	})

	Describe("Limit + Offset", func() {
		It("Should page through resources in order", func() {
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
})

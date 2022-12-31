// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

var _ = Describe("retrieveEntity", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriterUsingTxn(txn) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its ID", func() {
			id := newEmptyID("A")
			Expect(w.DefineResource(id)).To(Succeed())
			var r ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(id).
				Entry(&r).
				Exec(),
			).To(Succeed())
			v, ok := schema.Get[string](r.Entity, "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("A"))
		})
		It("Should retrieve multiple resources by their ID", func() {
			ids := []ontology.ID{newEmptyID("A"), newEmptyID("B")}
			Expect(w.DefineResource(ids[0])).To(Succeed())
			Expect(w.DefineResource(ids[1])).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(ids...).
				Entries(&r).
				Exec(),
			).To(Succeed())
			v, ok := schema.Get[string](r[0].Entity, "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("A"))
			v, ok = schema.Get[string](r[1].Entity, "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("B"))
		})
	})
	Describe("Multi Clause", func() {
		Describe("Parental Traversal", func() {
			It("Should retrieve the parent of a resource", func() {
				a := newEmptyID("A")
				b := newEmptyID("B")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineRelationship(a, ontology.ParentOf, b)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					Entry(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r.Entity, "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("B"))
			})
			It("Should retrieve the parents of multiple resources", func() {
				a := newEmptyID("A")
				b := newEmptyID("B")
				c := newEmptyID("C")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineResource(c)).To(Succeed())
				Expect(w.DefineRelationship(a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(a, ontology.ParentOf, c)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					Entries(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r[0].Entity, "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("B"))
				v, ok = schema.Get[string](r[1].Entity, "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("C"))
			})
			It("Should retrieve the grandparents of a resource", func() {
				a := newEmptyID("A")
				b := newEmptyID("B")
				c := newEmptyID("C")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineResource(c)).To(Succeed())
				Expect(w.DefineRelationship(a, ontology.ParentOf, b)).To(Succeed())
				Expect(w.DefineRelationship(b, ontology.ParentOf, c)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Children).
					TraverseTo(ontology.Children).
					Entry(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r.Entity, "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("C"))
			})
		})
	})
})

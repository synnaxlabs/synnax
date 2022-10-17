package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/schema"
)

var _ = Describe("RetrieveEntity", func() {
	var w ontology.Writer
	BeforeEach(func() { w = otg.NewWriterUsingTxn(txn) })
	Describe("Single Clause", func() {
		It("Should retrieve a resource by its ID", func() {
			id := newEmptyID("Key")
			Expect(w.DefineResource(id)).To(Succeed())
			var r ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(id).
				Entry(&r).
				Exec(),
			).To(Succeed())
			v, ok := schema.Get[string](r.Entity(), "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("Key"))
		})
		It("Should retrieve multiple resources by their ID", func() {
			ids := []ontology.ID{newEmptyID("Key"), newEmptyID("V")}
			Expect(w.DefineResource(ids[0])).To(Succeed())
			Expect(w.DefineResource(ids[1])).To(Succeed())
			var r []ontology.Resource
			Expect(w.NewRetrieve().
				WhereIDs(ids...).
				Entries(&r).
				Exec(),
			).To(Succeed())
			v, ok := schema.Get[string](r[0].Entity(), "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("Key"))
			v, ok = schema.Get[string](r[1].Entity(), "key")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal("V"))
		})
	})
	Describe("Multi Clause", func() {
		Describe("Parental Traversal", func() {
			It("Should retrieve the parent of a resource", func() {
				a := newEmptyID("Key")
				b := newEmptyID("V")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineRelationship(a, b, ontology.Parent)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Parents).
					Entry(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r.Entity(), "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("V"))
			})
			It("Should retrieve the parents of multiple resources", func() {
				a := newEmptyID("Key")
				b := newEmptyID("V")
				c := newEmptyID("C")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineResource(c)).To(Succeed())
				Expect(w.DefineRelationship(a, b, ontology.Parent)).To(Succeed())
				Expect(w.DefineRelationship(a, c, ontology.Parent)).To(Succeed())
				var r []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Parents).
					Entries(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r[0].Entity(), "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("V"))
				v, ok = schema.Get[string](r[1].Entity(), "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("C"))
			})
			It("Should retrieve the grandparents of a resource", func() {
				a := newEmptyID("Key")
				b := newEmptyID("V")
				c := newEmptyID("C")
				Expect(w.DefineResource(a)).To(Succeed())
				Expect(w.DefineResource(b)).To(Succeed())
				Expect(w.DefineResource(c)).To(Succeed())
				Expect(w.DefineRelationship(a, b, ontology.Parent)).To(Succeed())
				Expect(w.DefineRelationship(b, c, ontology.Parent)).To(Succeed())
				var r ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(a).
					TraverseTo(ontology.Parents).
					TraverseTo(ontology.Parents).
					Entry(&r).
					Exec(),
				).To(Succeed())
				v, ok := schema.Get[string](r.Entity(), "key")
				Expect(ok).To(BeTrue())
				Expect(v).To(Equal("C"))
			})
		})
	})
})

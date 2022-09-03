package ontology_test

import (
	"github.com/arya-analytics/delta/pkg/distribution/ontology"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Write", func() {
	var (
		w  ontology.Writer
		id ontology.ID
	)
	BeforeEach(func() {
		w = otg.NewWriterUsingTxn(txn)
		id = newEmptyID("foo")
	})
	Describe("Resources", func() {
		Describe("Defining a Resource", func() {
			It("Should define a resource by its ID", func() {
				Expect(w.DefineResource(id)).To(Succeed())
				Expect(w.NewRetrieve().WhereIDs(id).Exec()).To(Succeed())
			})
		})
		Describe("Deleting a Resource", func() {
			It("Should delete a resource by its ID", func() {
				Expect(w.DefineResource(id)).To(Succeed())
				Expect(w.DeleteResource(id)).To(Succeed())
				err := w.NewRetrieve().WhereIDs(id).Exec()
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.NotFound)).To(BeTrue())
			})
		})
		Describe("Idempotency", func() {
			Specify("Defining a resource should be idempotent", func() {
				Expect(w.DefineResource(id)).To(Succeed())
				Expect(w.DefineResource(id)).To(Succeed())
				Expect(w.NewRetrieve().WhereIDs(id).Exec()).To(Succeed())
			})
		})
	})
	Describe("Relationships", func() {
		var idOne, idTwo ontology.ID
		BeforeEach(func() {
			idOne = newEmptyID("foo")
			idTwo = newEmptyID("bar")
			Expect(w.DefineResource(idOne)).To(Succeed())
			Expect(w.DefineResource(idTwo)).To(Succeed())
		})
		AfterEach(func() {
			Expect(w.DeleteResource(idOne)).To(Succeed())
			Expect(w.DeleteResource(idTwo)).To(Succeed())
		})
		Describe("Defining a Relationship", func() {
			It("Should define a relationship by its ID", func() {
				Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
				var res []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(idOne).
					TraverseTo(ontology.Parents).
					Entries(&res).
					Exec()).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].ID).To(Equal(idTwo))
			})
			Context("Resources are not defined", func() {
				It("Should return a query.NotFound error", func() {
					err := w.DefineRelationship(
						idOne,
						newEmptyID("42"),
						ontology.Parent,
					)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, query.NotFound)).To(BeTrue())
				})
			})
			Context("Cyclic violations", func() {
				It(
					"Should return an error if a relationship is defined in two directions",
					func() {
						Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
						err := w.DefineRelationship(idTwo, idOne, ontology.Parent)
						Expect(err).To(HaveOccurred())
						Expect(errors.Is(err, ontology.CyclicDependency)).To(BeTrue())
					},
				)
				It("Should return an error is a relationships creates a cycle",
					func() {
						Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
						idThree := ontology.ID{Key: "qux", Type: "quux"}
						Expect(w.DefineResource(idThree)).To(Succeed())
						Expect(w.DefineRelationship(idTwo, idThree, ontology.Parent)).To(Succeed())
						err := w.DefineRelationship(idThree, idOne, ontology.Parent)
						Expect(err).To(HaveOccurred())
						Expect(errors.Is(err, ontology.CyclicDependency)).To(BeTrue())
					})
			})
		})
		Describe("Deleting a Relationship", func() {
			It("Should delete a relationship by its ID", func() {
				Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
				Expect(w.DeleteRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
				var res []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(idOne).
					TraverseTo(ontology.Parents).
					Entries(&res).
					Exec()).To(Succeed())
				Expect(res).To(HaveLen(0))
			})
		})
		Describe("Idempotency", func() {
			Specify("Defining a relationship should be idempotent", func() {
				Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
				Expect(w.DefineRelationship(idOne, idTwo, ontology.Parent)).To(Succeed())
			})
		})
	})
})

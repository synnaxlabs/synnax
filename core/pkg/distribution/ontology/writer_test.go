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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var (
		w  ontology.Writer
		id ontology.ID
	)
	BeforeEach(func() {
		w = otg.NewWriter(tx)
		id = newSampleType("foo")
	})
	Describe("Resources", func() {
		Describe("Defining a Resource", func() {
			It("Should define a resource by its ID", func() {
				Expect(w.DefineResource(ctx, id)).To(Succeed())
				Expect(w.NewRetrieve().WhereIDs(id).Exec(ctx, tx)).To(Succeed())
			})
		})
		It("Should define many resources by their names", func() {
			ids := []ontology.ID{id, newSampleType("bar")}
			Expect(w.DefineManyResources(ctx, ids)).To(Succeed())
			Expect(w.NewRetrieve().WhereIDs(ids...).Exec(ctx, tx)).To(Succeed())
		})
		Describe("Deleting a Resource", func() {
			It("Should delete a resource by its ID", func() {
				Expect(w.DefineResource(ctx, id)).To(Succeed())
				Expect(w.DeleteResource(ctx, id)).To(Succeed())
				err := w.NewRetrieve().WhereIDs(id).Exec(ctx, tx)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, query.ErrNotFound)).To(BeTrue())
			})
		})
		Describe("Idempotency", func() {
			Specify("Defining a resource should be idempotent", func() {
				Expect(w.DefineResource(ctx, id)).To(Succeed())
				Expect(w.DefineResource(ctx, id)).To(Succeed())
				Expect(w.NewRetrieve().WhereIDs(id).Exec(ctx, tx)).To(Succeed())
			})
		})
	})
	Describe("Relationships", func() {
		var idOne, idTwo ontology.ID
		BeforeEach(func() {
			idOne = newSampleType("foo")
			idTwo = newSampleType("bar")
			Expect(w.DefineResource(ctx, idOne)).To(Succeed())
			Expect(w.DefineResource(ctx, idTwo)).To(Succeed())
		})
		AfterEach(func() {
			Expect(w.DeleteResource(ctx, idOne)).To(Succeed())
			Expect(w.DeleteResource(ctx, idTwo)).To(Succeed())
		})
		Describe("Defining a Relationship", func() {
			It("Should define a relationship by its ID", func() {
				Expect(w.DefineRelationship(
					ctx,
					idOne,
					ontology.RelationshipTypeParentOf,
					idTwo,
				)).To(Succeed())
				var res []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(idOne).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].ID).To(Equal(idTwo))
			})
			Context("Resources are not defined", func() {
				It("Should return a query.IDsNotFound error", func() {
					err := w.DefineRelationship(
						ctx,
						idOne,
						ontology.RelationshipTypeParentOf,
						newSampleType("42"),
					)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, query.ErrNotFound)).To(BeTrue())
				})
			})
			Context("Cyclic violations", func() {
				It(
					"Should return an error if a relationship is defined in two directions",
					func() {
						Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
						err := w.DefineRelationship(ctx, idTwo, ontology.RelationshipTypeParentOf, idOne)
						Expect(err).To(HaveOccurred())
						Expect(errors.Is(err, ontology.ErrCycle)).To(BeTrue())
					},
				)
				It("Should return an error is a relationships creates a cycle",
					func() {
						Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
						idThree := ontology.ID{Key: "qux", Type: "quux"}
						Expect(w.DefineResource(ctx, idThree)).To(Succeed())
						Expect(w.DefineRelationship(ctx, idTwo, ontology.RelationshipTypeParentOf, idThree)).To(Succeed())
						err := w.DefineRelationship(ctx, idThree, ontology.RelationshipTypeParentOf, idOne)
						Expect(err).To(HaveOccurred())
						Expect(errors.Is(err, ontology.ErrCycle)).To(BeTrue())
					})
			})
		})
		Describe("Defining a Relationship to Many Resources", func() {
			It("Should define a relationship to many resources by their IDs", func() {
				Expect(w.DefineFromOneToManyRelationships(
					ctx,
					idOne,
					ontology.RelationshipTypeParentOf,
					[]ontology.ID{idTwo},
				)).To(Succeed())
				var res []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(idOne).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].ID).To(Equal(idTwo))
			})
			It("Should return an error if any of the resources are not defined", func() {
				Expect(w.DefineFromOneToManyRelationships(
					ctx,
					idOne,
					ontology.RelationshipTypeParentOf,
					[]ontology.ID{newSampleType("42")},
				)).To(HaveOccurredAs(query.ErrNotFound))
			})
			It("Should return an error if a cyclic relationship is created", func() {
				Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
				err := w.DefineFromOneToManyRelationships(
					ctx,
					idTwo,
					ontology.RelationshipTypeParentOf,
					[]ontology.ID{idOne},
				)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ontology.ErrCycle)).To(BeTrue())
			})
		})
		Describe("Deleting a Relationship", func() {
			It("Should delete a relationship by its ID", func() {
				Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
				Expect(w.DeleteRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
				var res []ontology.Resource
				Expect(w.NewRetrieve().
					WhereIDs(idOne).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(0))
			})
			Describe("DeleteOutgoingRelationshipsOfType", func() {
				It("Should delete all outgoing relationships of a type", func() {
					var t ontology.RelationshipType = "baz"
					Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
					Expect(w.DefineRelationship(ctx, idOne, t, idTwo)).To(Succeed())
					Expect(w.DeleteOutgoingRelationshipsOfType(ctx, idOne, ontology.RelationshipTypeParentOf)).To(Succeed())
					var res []ontology.Resource
					Expect(w.NewRetrieve().
						WhereIDs(idOne).
						TraverseTo(ontology.ChildrenTraverser).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res).To(HaveLen(0))
				})
			})
			Describe("DeleteIncomingRelationshipsOfType", func() {
				It("Should delete all incoming relationships of a type", func() {
					Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
					Expect(w.DefineRelationship(ctx, idOne, label.OntologyRelationshipTypeLabeledBy, idTwo)).To(Succeed())
					Expect(w.DeleteIncomingRelationshipsOfType(ctx, idTwo, ontology.RelationshipTypeParentOf)).To(Succeed())
					var res []ontology.Resource
					Expect(w.NewRetrieve().
						WhereIDs(idTwo).
						TraverseTo(ontology.ParentsTraverser).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res).To(HaveLen(0))
					var res2 []ontology.Resource
					Expect(w.NewRetrieve().
						WhereIDs(idOne).
						TraverseTo(label.LabelsOntologyTraverser).
						Entries(&res2).
						Exec(ctx, tx)).To(Succeed())
					Expect(res2).To(HaveLen(1))

				})
			})
			Describe("DeleteOutgoingRelationshipsOfType", func() {
				It("Should delete all outgoing relationships of a type", func() {
					Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
					Expect(w.DefineRelationship(ctx, idOne, label.OntologyRelationshipTypeLabeledBy, idTwo)).To(Succeed())
					Expect(w.DeleteOutgoingRelationshipsOfType(ctx, idOne, ontology.RelationshipTypeParentOf)).To(Succeed())
					var res []ontology.Resource
					Expect(w.NewRetrieve().
						WhereIDs(idOne).
						TraverseTo(ontology.ChildrenTraverser).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(res).To(HaveLen(0))
					var res2 []ontology.Resource
					Expect(w.NewRetrieve().
						WhereIDs(idOne).
						TraverseTo(label.LabelsOntologyTraverser).
						Entries(&res2).
						Exec(ctx, tx)).To(Succeed())
					Expect(res2).To(HaveLen(1))
				})
			})
		})
		Describe("Idempotency", func() {
			Specify("Defining a relationship should be idempotent", func() {
				Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
				Expect(w.DefineRelationship(ctx, idOne, ontology.RelationshipTypeParentOf, idTwo)).To(Succeed())
			})
		})
	})
})

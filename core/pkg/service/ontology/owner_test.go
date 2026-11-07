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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Ownership", func() {
	var (
		owned   *ontology.Ontology
		owner   *ontology.Owner
		ownedTx gorp.Tx
		dir     = resource.ID{Type: resource.TypeGroup, Key: "dir"}
		child   = resource.ID{Type: resource.TypeArc, Key: "child"}
		task    = resource.ID{Type: resource.TypeTask, Key: "task"}
		other   = resource.ID{Type: resource.TypeChannel, Key: "other"}
		edge    = ontology.Relationship{
			From: dir,
			Type: ontology.RelationshipTypeParentOf,
			To:   child,
		}
	)
	BeforeEach(func(ctx SpecContext) {
		ownedDB := DeferClose(gorp.Wrap(memkv.New()))
		owned = MustOpen(ontology.Open(ctx, ontology.Config{DB: ownedDB}))
		owner = owned.Own(ontology.Ownership{
			Children: []resource.Type{
				resource.TypeArc,
				resource.TypeGroup,
			},
			Parents: []resource.Type{resource.TypeGroup},
		})
		ownedTx = DeferClose(ownedDB.OpenTx())
		Expect(owned.NewWriter(ownedTx).DefineResources(
			ctx, dir, child, task, other,
		)).To(Succeed())
	})

	exists := func(ctx SpecContext, rel ontology.Relationship) bool {
		return MustSucceed(owned.RelationshipExists(ctx, ownedTx, rel))
	}

	It("Should refuse another writer an edge to an owned child type", func(
		ctx SpecContext,
	) {
		Expect(owned.NewWriter(ownedTx).DefineRelationships(
			ctx, other, ontology.RelationshipTypeParentOf, child,
		)).To(MatchError(validate.ErrValidation))
	})

	It("Should refuse another writer an edge from an owned parent type", func(
		ctx SpecContext,
	) {
		Expect(owned.NewWriter(ownedTx).DefineRelationships(
			ctx, dir, ontology.RelationshipTypeParentOf, other,
		)).To(MatchError(validate.ErrValidation))
	})

	It("Should let the owner define and delete an owned edge", func(
		ctx SpecContext,
	) {
		w := owner.NewWriter(ownedTx)
		Expect(w.DefineRelationships(
			ctx, dir, ontology.RelationshipTypeParentOf, child,
		)).To(Succeed())
		Expect(exists(ctx, edge)).To(BeTrue())
		Expect(w.DeleteRelationships(ctx, edge)).To(Succeed())
		Expect(exists(ctx, edge)).To(BeFalse())
	})

	DescribeTable(
		"Should refuse deleting an owned edge from another writer",
		func(ctx SpecContext, del func(SpecContext, ontology.Writer) error) {
			Expect(owner.NewWriter(ownedTx).DefineRelationships(
				ctx, dir, ontology.RelationshipTypeParentOf, child,
			)).To(Succeed())
			Expect(del(ctx, owned.NewWriter(ownedTx))).
				To(MatchError(validate.ErrValidation))
			Expect(exists(ctx, edge)).To(BeTrue())
		},
		Entry("DeleteRelationships", func(ctx SpecContext, w ontology.Writer) error {
			return w.DeleteRelationships(ctx, edge)
		}),
		Entry(
			"DeleteIncomingRelationshipsOfType",
			func(ctx SpecContext, w ontology.Writer) error {
				return w.DeleteIncomingRelationshipsOfType(
					ctx, child, ontology.RelationshipTypeParentOf,
				)
			},
		),
		Entry(
			"DeleteOutgoingRelationshipsOfType",
			func(ctx SpecContext, w ontology.Writer) error {
				return w.DeleteOutgoingRelationshipsOfType(
					ctx, dir, ontology.RelationshipTypeParentOf,
				)
			},
		),
		Entry(
			"ReplaceOutgoingRelationshipsOfType",
			func(ctx SpecContext, w ontology.Writer) error {
				return w.ReplaceOutgoingRelationshipsOfType(
					ctx, dir, ontology.RelationshipTypeParentOf,
				)
			},
		),
		Entry("DeleteResources of the child", func(
			ctx SpecContext,
			w ontology.Writer,
		) error {
			return w.DeleteResources(ctx, child)
		}),
		Entry("DeleteResources of the parent", func(
			ctx SpecContext,
			w ontology.Writer,
		) error {
			return w.DeleteResources(ctx, dir)
		}),
	)

	It("Should leave the children of a type owned only as a child", func(
		ctx SpecContext,
	) {
		w := owned.NewWriter(ownedTx)
		Expect(w.DefineRelationships(
			ctx, child, ontology.RelationshipTypeParentOf, task,
		)).To(Succeed())
		Expect(w.DeleteResources(ctx, task)).To(Succeed())
	})

	It("Should leave relationships of other types", func(ctx SpecContext) {
		var t ontology.RelationshipType = "baz"
		w := owned.NewWriter(ownedTx)
		Expect(w.DefineRelationships(ctx, dir, t, child)).To(Succeed())
		Expect(w.DeleteOutgoingRelationshipsOfType(ctx, dir, t)).To(Succeed())
	})
})

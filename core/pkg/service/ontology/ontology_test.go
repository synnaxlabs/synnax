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
	"context"
	"io"
	"iter"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/observe"
	. "github.com/synnaxlabs/x/testutil"
)

// emittingService is an ontology.Service whose change observable can be driven
// directly, used to exercise resource-change propagation.
type emittingService struct {
	observe.Observer[iter.Seq[ontology.Change]]
}

func (*emittingService) Type() ontology.ResourceType { return ontology.ResourceTypeChannel }

func (*emittingService) RetrieveResource(
	context.Context, string, gorp.Tx,
) (ontology.Resource, error) {
	return ontology.Resource{}, nil
}

func (*emittingService) OpenNexter(
	context.Context,
) (iter.Seq[ontology.Resource], io.Closer, error) {
	return slices.Values([]ontology.Resource{}), xio.NopCloser, nil
}

var _ = Describe("Ontology", func() {
	Describe("Config", func() {
		Describe("Validate", func() {
			It("Should return an error when the DB is not set", func() {
				Expect(
					ontology.Config{}.Validate(),
				).To(MatchError(ContainSubstring("db")))
			})
			It("Should succeed when the DB is set", func() {
				Expect(ontology.Config{DB: db}.Validate()).To(Succeed())
			})
		})
		Describe("Override", func() {
			It("Should adopt the override DB when the base DB is nil", func() {
				Expect(
					ontology.Config{}.Override(ontology.Config{DB: db}).DB,
				).To(Equal(db))
			})
			It("Should retain the base DB when the override DB is nil", func() {
				Expect(
					ontology.Config{DB: db}.Override(ontology.Config{}).DB,
				).To(Equal(db))
			})
			It(
				"Should adopt the override instrumentation when the base is zero",
				func() {
					ins := alamos.New("ontology-override-test")
					Expect(
						ontology.Config{}.Override(
							ontology.Config{Instrumentation: ins},
						).Meta.Path,
					).To(Equal("ontology-override-test"))
				},
			)
		})
	})

	Describe("Open", func() {
		It(
			"Should return an error when opened with an invalid config",
			func(ctx SpecContext) {
				Expect(ontology.Open(ctx, ontology.Config{})).
					Error().To(MatchError(ContainSubstring("db")))
			},
		)
	})

	Describe("RelationshipExists", func() {
		var (
			w             ontology.Writer
			parent, child ontology.ID
		)
		BeforeEach(func(ctx SpecContext) {
			w = otg.NewWriter(tx)
			parent, child = newSampleType(
				"rel-exists-parent",
			), newSampleType(
				"rel-exists-child",
			)
			Expect(w.DefineResources(ctx, parent, child)).To(Succeed())
		})
		It("Should report true when the relationship exists", func(ctx SpecContext) {
			Expect(
				w.DefineRelationships(
					ctx,
					parent,
					ontology.RelationshipTypeParentOf,
					child,
				),
			).
				To(Succeed())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: parent,
				Type: ontology.RelationshipTypeParentOf,
				To:   child,
			})).To(BeTrue())
		})
		It(
			"Should report false when the relationship does not exist",
			func(ctx SpecContext) {
				Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
					From: parent,
					Type: ontology.RelationshipTypeParentOf,
					To:   child,
				})).To(BeFalse())
			},
		)
		It("Should report false for the reverse direction", func(ctx SpecContext) {
			Expect(
				w.DefineRelationships(
					ctx,
					parent,
					ontology.RelationshipTypeParentOf,
					child,
				),
			).
				To(Succeed())
			Expect(otg.RelationshipExists(ctx, tx, ontology.Relationship{
				From: child,
				Type: ontology.RelationshipTypeParentOf,
				To:   parent,
			})).To(BeFalse())
		})
	})

	Describe("ObserveResources", func() {
		It(
			"Should notify subscribers of resource changes emitted by services",
			func(ctx SpecContext) {
				d := DeferClose(gorp.Wrap(memkv.New()))
				o := MustOpen(ontology.Open(ctx, ontology.Config{DB: d}))
				svc := &emittingService{
					Observer: observe.New[iter.Seq[ontology.Change]](),
				}
				o.RegisterService(svc)
				called := false
				o.ObserveResources().
					OnChange(func(context.Context, iter.Seq[ontology.Change]) {
						called = true
					})
				svc.Notify(
					ctx,
					slices.Values([]ontology.Change{{Key: "obs-resource-test"}}),
				)
				Expect(called).To(BeTrue())
			},
		)
	})

	Describe("ObserveRelationships", func() {
		It(
			"Should notify subscribers when a relationship is defined",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				w := otg.NewWriter(tx)
				parent, child := newSampleType(
					"obs-rel-parent",
				), newSampleType(
					"obs-rel-child",
				)
				Expect(w.DefineResources(ctx, parent, child)).To(Succeed())
				called := false
				otg.ObserveRelationships().OnChange(
					func(context.Context, gorp.TxReader[string, ontology.Relationship]) {
						called = true
					},
				)
				Expect(
					w.DefineRelationships(
						ctx,
						parent,
						ontology.RelationshipTypeParentOf,
						child,
					),
				).
					To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(called).To(BeTrue())
			},
		)
	})
})

var _ = Describe("ParentsTraverser", func() {
	retrieveParents := func(ctx SpecContext, id ontology.ID) []ontology.ID {
		GinkgoHelper()
		var parents []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(id).
			TraverseTo(ontology.ParentsTraverser).
			Entries(&parents).
			Exec(ctx, tx)).To(Succeed())
		return ontology.ResourceIDs(parents)
	}
	It("Should return the parents of the given ID", func(ctx SpecContext) {
		w := otg.NewWriter(tx)
		parentA := newSampleType("rp-parent-a")
		parentB := newSampleType("rp-parent-b")
		childOne := newSampleType("rp-child-1")
		childTwo := newSampleType("rp-child-2")
		orphan := newSampleType("rp-orphan")
		Expect(w.DefineResources(
			ctx, parentA, parentB, childOne, childTwo, orphan,
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, parentA, ontology.RelationshipTypeParentOf, childOne,
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, parentB, ontology.RelationshipTypeParentOf, childOne,
		)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, parentA, ontology.RelationshipTypeParentOf, childTwo,
		)).To(Succeed())
		Expect(retrieveParents(ctx, childOne)).To(ConsistOf(parentA, parentB))
		Expect(retrieveParents(ctx, childTwo)).To(ConsistOf(parentA))
		Expect(retrieveParents(ctx, orphan)).To(BeEmpty())
	})
	It("Should ignore non-parent relationships", func(ctx SpecContext) {
		w := otg.NewWriter(tx)
		labeler := newSampleType("rp-labeler")
		labeled := newSampleType("rp-labeled")
		Expect(w.DefineResources(ctx, labeler, labeled)).To(Succeed())
		Expect(w.DefineRelationships(
			ctx, labeler, "labeled_by", labeled,
		)).To(Succeed())
		Expect(retrieveParents(ctx, labeled)).To(BeEmpty())
	})
})

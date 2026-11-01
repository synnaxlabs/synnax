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
	"fmt"
	"iter"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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

func (*emittingService) Search(
	context.Context,
	gorp.Tx,
	string,
	int,
) (search.Result[ontology.ID], error) {
	return search.Result[ontology.ID]{}, nil
}

// textService is an ontology.Service whose resources are keyed by their text. It
// records the tx that its last Search received.
type textService struct {
	observe.Noop[iter.Seq[ontology.Change]]
	typ  ontology.ResourceType
	keys []string
	err  error
	tx   gorp.Tx
}

func (s *textService) Type() ontology.ResourceType { return s.typ }

func (*textService) RetrieveResource(
	context.Context, string, gorp.Tx,
) (ontology.Resource, error) {
	return ontology.Resource{}, nil
}

func (s *textService) Search(
	_ context.Context,
	tx gorp.Tx,
	term string,
	limit int,
) (search.Result[ontology.ID], error) {
	s.tx = tx
	if s.err != nil {
		return search.Result[ontology.ID]{}, s.err
	}
	idx := search.New[string]()
	for _, k := range s.keys {
		idx.Set(k, k)
	}
	return search.MapKeys(idx.Search(term, limit), func(k string) ontology.ID {
		return ontology.ID{Type: s.typ, Key: k}
	}), nil
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

	Describe("Search", func() {
		var (
			d *gorp.DB
			o *ontology.Ontology
		)
		BeforeEach(func(ctx SpecContext) {
			d = DeferClose(gorp.Wrap(memkv.New()))
			o = MustOpen(ontology.Open(ctx, ontology.Config{DB: d}))
		})

		valves := func(n int) []string {
			keys := make([]string, n)
			for i := range keys {
				keys[i] = fmt.Sprintf("valve %d", i)
			}
			return keys
		}

		It("Should rank the matches of every service together", func(ctx SpecContext) {
			o.RegisterService(&textService{
				typ:  ontology.ResourceTypeUser,
				keys: []string{"turbopump inlet"},
			})
			o.RegisterService(&textService{
				typ:  ontology.ResourceTypeRack,
				keys: []string{"turbopump", "valve"},
			})
			Expect(o.Search(ctx, nil, "turbopump")).To(Equal([]ontology.ID{
				{Type: ontology.ResourceTypeRack, Key: "turbopump"},
				{Type: ontology.ResourceTypeUser, Key: "turbopump inlet"},
			}))
		})

		It("Should return at most 100 IDs", func(ctx SpecContext) {
			o.RegisterService(
				&textService{typ: ontology.ResourceTypeUser, keys: valves(80)},
			)
			o.RegisterService(
				&textService{typ: ontology.ResourceTypeRack, keys: valves(80)},
			)
			Expect(o.Search(ctx, nil, "valve")).To(HaveLen(100))
		})

		It("Should match no builtin resource", func(ctx SpecContext) {
			Expect(o.Search(ctx, nil, "root")).To(BeEmpty())
		})

		It("Should pass tx to every service, or the DB when tx is nil",
			func(ctx SpecContext) {
				svc := &textService{typ: ontology.ResourceTypeUser}
				o.RegisterService(svc)
				MustSucceed(o.Search(ctx, nil, "valve"))
				Expect(svc.tx).To(BeIdenticalTo(d))
				tx := DeferClose(d.OpenTx())
				MustSucceed(o.Search(ctx, tx, "valve"))
				Expect(svc.tx).To(BeIdenticalTo(tx))
			})

		It("Should search only the first service registered for a type",
			func(ctx SpecContext) {
				o.RegisterService(&textService{
					typ:  ontology.ResourceTypeUser,
					keys: []string{"valve"},
				})
				o.RegisterService(&textService{
					typ:  ontology.ResourceTypeUser,
					keys: []string{"valve inlet"},
				})
				Expect(o.Search(ctx, nil, "valve")).To(Equal([]ontology.ID{
					{Type: ontology.ResourceTypeUser, Key: "valve"},
				}))
			})

		It("Should return the error of a service", func(ctx SpecContext) {
			err := errors.New("search failed")
			o.RegisterService(&textService{typ: ontology.ResourceTypeUser, err: err})
			Expect(o.Search(ctx, nil, "valve")).Error().To(MatchError(err))
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
		It(
			"Should only notify subscribers of replaced relationships that change",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				w := otg.NewWriter(tx)
				from := newSampleType("replace-from")
				kept := newSampleType("replace-kept")
				removed := newSampleType("replace-removed")
				added := newSampleType("replace-added")
				Expect(w.DefineResources(ctx, from, kept, removed, added)).To(Succeed())
				Expect(w.DefineRelationships(
					ctx, from, ontology.RelationshipTypeParentOf, kept, removed,
				)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				var changes []change.Change[string, ontology.Relationship]
				disconnect := otg.ObserveRelationships().OnChange(
					func(_ context.Context, r gorp.TxReader[string, ontology.Relationship]) {
						changes = append(changes, slices.Collect(r)...)
					},
				)
				defer disconnect()
				tx = db.OpenTx()
				defer func() { Expect(tx.Close()).To(Succeed()) }()
				Expect(otg.NewWriter(tx).ReplaceOutgoingRelationshipsOfType(
					ctx, from, ontology.RelationshipTypeParentOf, kept, added,
				)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				rel := func(to ontology.ID) string {
					return ontology.Relationship{
						From: from, Type: ontology.RelationshipTypeParentOf, To: to,
					}.GorpKey()
				}
				type op struct {
					variant change.Variant
					key     string
				}
				Expect(lo.Map(
					changes,
					func(c change.Change[string, ontology.Relationship], _ int) op {
						return op{variant: c.Variant, key: c.Key}
					},
				)).To(ConsistOf(
					op{variant: change.VariantDelete, key: rel(removed)},
					op{variant: change.VariantSet, key: rel(added)},
				))
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

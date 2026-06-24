// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a new range", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should return a validation error when the name is empty", func(ctx SpecContext) {
			Expect(w.Create(ctx, &ranger.Range{})).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("should return an error if the time range is invalid", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(10 * telem.Second),
					End:   telem.TimeStamp(5 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).
				To(MatchError(
					ContainSubstring("time_range.start cannot be after time_range.end"),
				))
		})
		It("should create a range with start equal to end", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(5 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil))
		})
		It("Should not override the UUID if it is already set", func(ctx SpecContext) {
			k := uuid.New()
			r := &ranger.Range{
				Key:  k,
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).To(Equal(k))
		})
		Context("Parent Management", func() {
			It("Should set a custom parent for the range", func(ctx SpecContext) {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent,
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should NOT re-set the custom parent when the range exists but no parent is provided", func(ctx SpecContext) {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent,
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				r.Parent = nil
				Expect(w.Create(ctx, r)).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should change the custom parent when the range exists and a new parent is provided", func(ctx SpecContext) {
				parent1 := ranger.Range{
					Name:      "Parent1",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent1)).To(Succeed())
				parent2 := ranger.Range{
					Name:      "Parent2",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent2)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent1,
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				r.Parent = &parent2
				Expect(w.Create(ctx, r)).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent2.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
				var res2 ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent1.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res2).
					Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			})
			It("Should create multiple ranges with the same parent", func(ctx SpecContext) {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r1 := ranger.Range{
					Name:      "Range1",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent,
				}
				r2 := ranger.Range{
					Name:      "Range2",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent,
				}
				Expect(w.CreateMany(ctx, &[]ranger.Range{r1, r2})).To(Succeed())
				var res []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(2))
			})
			It("Should resolve the parent's name and time range on create", func(ctx SpecContext) {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				child := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &ranger.Range{Key: parent.Key},
				}
				Expect(w.Create(ctx, child)).To(Succeed())
				Expect(child.Parent.Name).To(Equal("Parent"))
				Expect(child.Parent.TimeRange).To(Equal(parent.TimeRange))
			})
			It("Should NOT resolve the parent's labels on create", func(ctx SpecContext) {
				l := &label.Label{Name: "L"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Label{*l},
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				child := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &ranger.Range{Key: parent.Key},
				}
				Expect(w.Create(ctx, child)).To(Succeed())
				Expect(child.Parent.Labels).To(BeEmpty())
			})
			It("Should not resolve the parent's own parent on create", func(ctx SpecContext) {
				grandparent := ranger.Range{
					Name:      "Grandparent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &grandparent)).To(Succeed())
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &grandparent,
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				child := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &ranger.Range{Key: parent.Key},
				}
				Expect(w.Create(ctx, child)).To(Succeed())
				Expect(child.Parent.Parent).To(BeNil())
			})
		})
		Context("Label Management", func() {
			It("Should define LabeledBy relationships for labels on the range", func(ctx SpecContext) {
				l1 := &label.Label{Name: "L1"}
				l2 := &label.Label{Name: "L2"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l1)).To(Succeed())
				Expect(labelSvc.NewWriter(tx).Create(ctx, l2)).To(Succeed())
				r := &ranger.Range{
					Name:      "Labeled",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Label{*l1, *l2},
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				resolved := MustSucceed(labelSvc.RetrieveFor(ctx, r.OntologyID(), tx))
				resolvedKeys := lo.Map(resolved, func(l label.Label, _ int) label.Key { return l.Key })
				Expect(resolvedKeys).To(ConsistOf(l1.Key, l2.Key))
			})
			It("Should leave label relationships untouched when r.Labels is empty", func(ctx SpecContext) {
				l := &label.Label{Name: "Existing"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(labelSvc.NewWriter(tx).Label(ctx, r.OntologyID(), []label.Key{l.Key})).To(Succeed())
				Expect(w.Create(ctx, r)).To(Succeed())
				resolved := MustSucceed(labelSvc.RetrieveFor(ctx, r.OntologyID(), tx))
				Expect(resolved).To(HaveLen(1))
				Expect(resolved[0].Key).To(Equal(l.Key))
			})
			It("Should add to existing label relationships when a range is re-created with new labels", func(ctx SpecContext) {
				l1 := &label.Label{Name: "First"}
				l2 := &label.Label{Name: "Second"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l1)).To(Succeed())
				Expect(labelSvc.NewWriter(tx).Create(ctx, l2)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Label{*l1},
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				r.Labels = []label.Label{*l2}
				Expect(w.Create(ctx, r)).To(Succeed())
				resolved := MustSucceed(labelSvc.RetrieveFor(ctx, r.OntologyID(), tx))
				keys := lo.Map(resolved, func(l label.Label, _ int) label.Key { return l.Key })
				Expect(keys).To(ConsistOf(l1.Key, l2.Key))
			})
		})
	})

	Describe("Delete", func() {
		It("Should delete a range by its key", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, r.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r.Key)).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should delete all child ranges when a range is deleted", func(ctx SpecContext) {
			parent := ranger.Range{
				Name: "Parent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			r := ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				},
				Parent: &parent,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, parent.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r.Key)).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should delete multiple ranges in a single Delete call", func(ctx SpecContext) {
			r1 := ranger.Range{
				Name:      "Range1",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			r2 := ranger.Range{
				Name:      "Range2",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			r3 := ranger.Range{
				Name:      "Range3",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).CreateMany(ctx, &[]ranger.Range{r1, r2, r3})).
				To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, r1.Key, r2.Key, r3.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r1.Key)).Exists(ctx, tx)).
				To(BeFalse())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r2.Key)).Exists(ctx, tx)).
				To(BeFalse())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r3.Key)).Exists(ctx, tx)).
				To(BeFalse())
		})
		It("Should be a no-op when called with no keys", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx)).To(Succeed())
		})
		It("Should recursively delete grandchild ranges when a range is deleted", func(ctx SpecContext) {
			grandparent := ranger.Range{
				Name:      "Grandparent",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).Create(ctx, &grandparent)).To(Succeed())
			parent := ranger.Range{
				Name:      "Parent",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
				Parent:    &grandparent,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
			child := ranger.Range{
				Name:      "Child",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
				Parent:    &parent,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &child)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, grandparent.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(grandparent.Key)).Exists(ctx, tx)).
				To(BeFalse())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(parent.Key)).Exists(ctx, tx)).
				To(BeFalse())
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(child.Key)).Exists(ctx, tx)).
				To(BeFalse())
		})
	})

	Describe("SetEnd", func() {
		It("Should set the end bound while preserving the start", func(ctx SpecContext) {
			r := ranger.Range{
				Name:      "set_end",
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.TimeStampMax},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			end := telem.SecondTS * 10
			Expect(svc.NewWriter(tx).SetEnd(ctx, r.Key, end)).To(Succeed())
			var updated ranger.Range
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r.Key)).Entry(&updated).
				Exec(ctx, tx)).To(Succeed())
			Expect(updated.TimeRange.Start).To(Equal(telem.SecondTS))
			Expect(updated.TimeRange.End).To(Equal(end))
		})

		It("Should overwrite a previously set end bound", func(ctx SpecContext) {
			r := ranger.Range{
				Name:      "set_end",
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.SecondTS * 5},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(svc.NewWriter(tx).SetEnd(ctx, r.Key, telem.SecondTS*20)).To(Succeed())
			var updated ranger.Range
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r.Key)).Entry(&updated).
				Exec(ctx, tx)).To(Succeed())
			Expect(updated.TimeRange.End).To(Equal(telem.SecondTS * 20))
		})

		It("Should return query.ErrNotFound when the range does not exist", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).SetEnd(ctx, uuid.New(), telem.Now())).
				To(MatchError(query.ErrNotFound))
		})

		It("Should leave other fields untouched", func(ctx SpecContext) {
			r := ranger.Range{
				Name:      "set_end_preserve",
				TimeRange: telem.TimeRange{Start: telem.SecondTS, End: telem.TimeStampMax},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
			Expect(svc.NewWriter(tx).SetEnd(ctx, r.Key, telem.SecondTS*3)).To(Succeed())
			var updated ranger.Range
			Expect(svc.NewRetrieve().Where(ranger.MatchKeys(r.Key)).Entry(&updated).
				Exec(ctx, tx)).To(Succeed())
			Expect(updated.Name).To(Equal(r.Name))
			Expect(updated.Key).To(Equal(r.Key))
		})
	})
})

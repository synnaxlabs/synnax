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
	"context"
	"fmt"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	BeforeEach(openTx)
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
			Expect(r.Key).ToNot(Equal(uuid.Nil()))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				Expect(w.Create(ctx, &ranger.Range{})).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It(
			"should return an error if the time range is invalid",
			func(ctx SpecContext) {
				r := &ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(10 * telem.Second),
						End:   telem.TimeStamp(5 * telem.Second),
					},
				}
				Expect(w.Create(ctx, r)).
					To(MatchError(
						ContainSubstring(
							"time_range.start cannot be after time_range.end",
						),
					))
			},
		)
		It("should create a range with start equal to end", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(5 * telem.Second),
				},
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			Expect(r.Key).ToNot(Equal(uuid.Nil()))
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
		It("Should refuse a range whose key exists", func(ctx SpecContext) {
			r := &ranger.Range{
				Name:      "Range",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(w.Create(ctx, r)).To(Succeed())
			again := ranger.Range{Key: r.Key, Name: "Renamed", TimeRange: r.TimeRange}
			Expect(w.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("range %s already exists", r.Key)),
			))
		})
		Context("Parent", func() {
			create := func(ctx SpecContext, name string, parent *ranger.Key) ranger.Range {
				GinkgoHelper()
				r := ranger.Range{
					Name:      name,
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    parent,
				}
				Expect(w.Create(ctx, &r)).To(Succeed())
				return r
			}
			retrieve := func(ctx SpecContext, key ranger.Key) ranger.Range {
				GinkgoHelper()
				var r ranger.Range
				Expect(svc.NewRetrieve().
					Where(ranger.MatchKeys(key)).
					Entry(&r).
					Exec(ctx, tx)).To(Succeed())
				return r
			}
			setParent := func(key ranger.Key) ranger.Action {
				return ranger.NewSetAction(
					ranger.SetPayload{Path: []string{"parent"}, Value: key},
				)
			}
			It("Should store the parent of the range", func(ctx SpecContext) {
				parent := create(ctx, "Parent", nil)
				r := create(ctx, "Range", &parent.Key)
				Expect(retrieve(ctx, r.Key).Parent).To(HaveValue(Equal(parent.Key)))
			})
			It("Should move the range to a new parent", func(ctx SpecContext) {
				first, second := create(ctx, "First", nil), create(ctx, "Second", nil)
				r := create(ctx, "Range", &first.Key)
				Expect(w.Apply(ctx, r.Key, setParent(second.Key))).To(Succeed())
				Expect(retrieve(ctx, r.Key).Parent).To(HaveValue(Equal(second.Key)))
			})
			It("Should find the children of a range", func(ctx SpecContext) {
				parent := create(ctx, "Parent", nil)
				a, b := create(ctx, "A", &parent.Key), create(ctx, "B", &parent.Key)
				create(ctx, "Other", nil)
				var children []ranger.Range
				Expect(svc.NewRetrieve().
					Where(ranger.MatchParents(parent.Key)).
					Entries(&children).
					Exec(ctx, tx)).To(Succeed())
				Expect(lo.Map(children, func(r ranger.Range, _ int) ranger.Key {
					return r.Key
				})).To(ConsistOf(a.Key, b.Key))
			})
			It("Should reject a parent that does not exist", func(ctx SpecContext) {
				missing := uuid.New()
				r := ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &missing,
				}
				err := w.Create(ctx, &r)
				Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
				Expect(err).To(MatchError(
					"parent: range " + missing.String() + " does not exist: validation error",
				))
			})
			It("Should reject a parent that is a descendant", func(ctx SpecContext) {
				grandparent := create(ctx, "Grandparent", nil)
				parent := create(ctx, "Parent", &grandparent.Key)
				child := create(ctx, "Child", &parent.Key)
				err := w.Apply(ctx, grandparent.Key, setParent(child.Key))
				Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
				Expect(err).To(MatchError(
					"parent: range " + grandparent.Key.String() +
						" would form a cycle: validation error",
				))
			})
		})
		Context("Labels", func() {
			createLabel := func(ctx SpecContext, name string) label.Key {
				GinkgoHelper()
				l := &label.Label{Name: name}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
				return l.Key
			}
			labelsOf := func(ctx SpecContext, key ranger.Key) []label.Key {
				GinkgoHelper()
				var r ranger.Range
				Expect(svc.NewRetrieve().
					Where(ranger.MatchKeys(key)).
					Entry(&r).
					Exec(ctx, tx)).To(Succeed())
				return r.Labels
			}
			It("Should reject a label that does not exist", func(ctx SpecContext) {
				missing := uuid.New()
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Key{missing},
				}
				err := w.Create(ctx, r)
				Expect(err).To(BeAssignableToTypeOf(validate.PathError{}))
				Expect(err).To(MatchError(
					"labels: label " + missing.String() + " does not exist: validation error",
				))
			})
			It("Should add and replace labels", func(ctx SpecContext) {
				a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Key{a},
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(w.AddLabels(ctx, r.Key, []label.Key{b})).To(Succeed())
				Expect(labelsOf(ctx, r.Key)).To(ConsistOf(a, b))
				Expect(w.ReplaceLabels(ctx, r.Key, []label.Key{b})).To(Succeed())
				Expect(labelsOf(ctx, r.Key)).To(Equal([]label.Key{b}))
			})
			It("Should remove labels", func(ctx SpecContext) {
				a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Key{a, b},
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				Expect(w.RemoveLabels(ctx, r.Key, []label.Key{a})).To(Succeed())
				Expect(labelsOf(ctx, r.Key)).To(Equal([]label.Key{b}))
			})
			It(
				"Should remove a deleted label from the ranges that hold it",
				func(ctx SpecContext) {
					a, b := createLabel(ctx, "A"), createLabel(ctx, "B")
					r := &ranger.Range{
						Name:      "Range",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
						Labels:    []label.Key{a, b},
					}
					Expect(w.Create(ctx, r)).To(Succeed())
					Expect(labelSvc.NewWriter(tx).Delete(ctx, a)).To(Succeed())
					Expect(labelsOf(ctx, r.Key)).To(Equal([]label.Key{b}))
				},
			)
			It("Should find the ranges that hold a label", func(ctx SpecContext) {
				a := createLabel(ctx, "A")
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Labels:    []label.Key{a},
				}
				Expect(w.Create(ctx, r)).To(Succeed())
				var res []ranger.Range
				Expect(svc.NewRetrieve().
					Where(ranger.MatchLabels(a)).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Key).To(Equal(r.Key))
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
			Expect(
				svc.NewRetrieve().
					Where(ranger.MatchKeys(r.Key)).
					Entry(&retrieveR).
					Exec(ctx, tx),
			).To(MatchError(query.ErrNotFound))
		})
		It(
			"Should delete all child ranges when a range is deleted",
			func(ctx SpecContext) {
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
					Parent: &parent.Key,
				}
				Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, parent.Key)).To(Succeed())
				var retrieveR ranger.Range
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchKeys(r.Key)).
						Entry(&retrieveR).
						Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should delete multiple ranges in a single Delete call",
			func(ctx SpecContext) {
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
				Expect(
					svc.NewWriter(tx).Delete(ctx, r1.Key, r2.Key, r3.Key),
				).To(Succeed())
				Expect(
					svc.NewRetrieve().Where(ranger.MatchKeys(r1.Key)).Exists(ctx, tx),
				).
					To(BeFalse())
				Expect(
					svc.NewRetrieve().Where(ranger.MatchKeys(r2.Key)).Exists(ctx, tx),
				).
					To(BeFalse())
				Expect(
					svc.NewRetrieve().Where(ranger.MatchKeys(r3.Key)).Exists(ctx, tx),
				).
					To(BeFalse())
			},
		)
		It("Should be a no-op when called with no keys", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Delete(ctx)).To(Succeed())
		})
		It(
			"Should recursively delete grandchild ranges when a range is deleted",
			func(ctx SpecContext) {
				grandparent := ranger.Range{
					Name:      "Grandparent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(svc.NewWriter(tx).Create(ctx, &grandparent)).To(Succeed())
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &grandparent.Key,
				}
				Expect(svc.NewWriter(tx).Create(ctx, &parent)).To(Succeed())
				child := ranger.Range{
					Name:      "Child",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
					Parent:    &parent.Key,
				}
				Expect(svc.NewWriter(tx).Create(ctx, &child)).To(Succeed())
				Expect(svc.NewWriter(tx).Delete(ctx, grandparent.Key)).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchKeys(grandparent.Key)).
						Exists(ctx, tx),
				).
					To(BeFalse())
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchKeys(parent.Key)).
						Exists(ctx, tx),
				).
					To(BeFalse())
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchKeys(child.Key)).
						Exists(ctx, tx),
				).
					To(BeFalse())
			},
		)
	})
})

var _ = Describe("Writer broadcasts", func() {
	// commit commits the spec's transaction and deletes the given ranges after the
	// spec.
	commit := func(ctx context.Context, keys ...ranger.Key) {
		GinkgoHelper()
		Expect(tx.Commit(ctx)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, keys...)
			})).To(Succeed())
		})
	}
	newRange := func() *ranger.Range {
		return &ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	}
	// framesOf returns the recorded frames that broadcast on the range with the key.
	framesOf := func(
		rec *Recorder[ranger.Key, ranger.Action],
		key ranger.Key,
	) []actions.Scoped[ranger.Key, ranger.Action] {
		var res []actions.Scoped[ranger.Key, ranger.Action]
		for _, f := range rec.Snapshot() {
			if f.Key == key {
				res = append(res, f)
			}
		}
		return res
	}

	It("Should broadcast a create with labels and parent", func(ctx SpecContext) {
		openTx()
		parent := newRange()
		Expect(w.Create(ctx, parent)).To(Succeed())
		l := label.Label{Key: uuid.New(), Name: "Label"}
		Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		r := newRange()
		r.Parent, r.Labels = &parent.Key, []label.Key{l.Key}
		Expect(w.WithDispatchKey("dk-create").Create(ctx, r)).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
		commit(ctx, parent.Key, r.Key)
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return labelSvc.NewWriter(tx).Delete(ctx, l.Key)
			})).To(Succeed())
		})
		created := framesOf(rec, r.Key)
		Expect(created).To(HaveLen(1))
		f := created[0]
		Expect(f.Key).To(Equal(r.Key))
		Expect(f.DispatchKey).To(Equal("dk-create"))
		Expect(f.Actions).To(Equal([]ranger.Action{
			ranger.NewCreateAction(ranger.CreatePayload{Range: ranger.Range{
				Key:       r.Key,
				Name:      r.Name,
				TimeRange: r.TimeRange,
				Labels:    []label.Key{l.Key},
				Parent:    &parent.Key,
			}}),
		}))
	})

	It("Should broadcast a label edit as a set of the labels", func(ctx SpecContext) {
		openTx()
		l := label.Label{Key: uuid.New(), Name: "Label"}
		Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		r := newRange()
		Expect(w.Create(ctx, r)).To(Succeed())
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		Expect(w.AddLabels(ctx, r.Key, []label.Key{l.Key})).To(Succeed())
		commit(ctx, r.Key)
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return labelSvc.NewWriter(tx).Delete(ctx, l.Key)
			})).To(Succeed())
		})
		frames := framesOf(rec, r.Key)
		Expect(frames).To(HaveLen(2))
		Expect(frames[1].Actions).To(HaveLen(1))
		Expect(frames[1].Actions[0].Set.Path).To(Equal([]string{"labels"}))
	})

	It(
		"Should broadcast the labels a range keeps when a label delete detaches one",
		func(ctx SpecContext) {
			openTx()
			deleted := label.Label{Key: uuid.New(), Name: "Deleted"}
			kept := label.Label{Key: uuid.New(), Name: "Kept"}
			Expect(labelSvc.NewWriter(tx).Create(ctx, &deleted)).To(Succeed())
			Expect(labelSvc.NewWriter(tx).Create(ctx, &kept)).To(Succeed())
			r := newRange()
			r.Labels = []label.Key{deleted.Key, kept.Key}
			Expect(w.Create(ctx, r)).To(Succeed())
			commit(ctx, r.Key)
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return labelSvc.NewWriter(tx).Delete(ctx, kept.Key)
				})).To(Succeed())
			})
			rec := &Recorder[ranger.Key, ranger.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return labelSvc.NewWriter(tx).Delete(ctx, deleted.Key)
			})).To(Succeed())
			frames := framesOf(rec, r.Key)
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].Actions).To(Equal([]ranger.Action{
				ranger.NewSetAction(ranger.SetPayload{
					Path:  []string{"labels"},
					Value: []label.Key{kept.Key},
				}),
			}))
		},
	)

	It("Should not broadcast a create that never commits", func(ctx SpecContext) {
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		uncommitted := db.OpenTx()
		Expect(svc.NewWriter(uncommitted).Create(ctx, newRange())).To(Succeed())
		Expect(uncommitted.Close()).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
	})
})

var _ = Describe("Writer in concurrent transactions", func() {
	It(
		"Should keep the labels that adds write during concurrent edits",
		func(ctx SpecContext) {
			labels := make([]label.Key, 20)
			for i := range labels {
				l := label.Label{Name: fmt.Sprintf("Concurrent %d", i)}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return labelSvc.NewWriter(tx).Create(ctx, &l)
				})).To(Succeed())
				labels[i] = l.Key
			}
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return labelSvc.NewWriter(tx).Delete(ctx, labels...)
				})).To(Succeed())
			})
			r := ranger.Range{
				Name: "Concurrent",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &r)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, r.Key)
				})).To(Succeed())
			})
			var wg sync.WaitGroup
			wg.Go(func() {
				defer GinkgoRecover()
				for i := range 100 {
					rename := ranger.NewSetAction(ranger.SetPayload{
						Path:  []string{"name"},
						Value: fmt.Sprintf("Concurrent %d", i),
					})
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Apply(ctx, r.Key, rename)
					})).To(Succeed())
				}
			})
			wg.Go(func() {
				defer GinkgoRecover()
				for _, l := range labels {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).AddLabels(
							ctx,
							r.Key,
							[]label.Key{l},
						)
					})).To(Succeed())
				}
			})
			wg.Wait()
			var stored ranger.Range
			Expect(svc.NewRetrieve().
				Where(ranger.MatchKeys(r.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Labels).To(ConsistOf(labels))
		},
	)
})

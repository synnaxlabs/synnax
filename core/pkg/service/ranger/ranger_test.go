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
	"fmt"
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ranger", func() {
	BeforeEach(openTx)
	Describe("Derive", func() {
		var parent, child, leaf ranger.Range
		BeforeEach(func(ctx SpecContext) {
			tr := telem.SecondTS.SpanRange(telem.Second)
			parent = ranger.Range{Name: "Parent", TimeRange: tr}
			Expect(w.Create(ctx, &parent)).To(Succeed())
			child = ranger.Range{Name: "Child", TimeRange: tr, Parent: &parent.Key}
			Expect(w.Create(ctx, &child)).To(Succeed())
			leaf = ranger.Range{Name: "Leaf", TimeRange: tr}
			Expect(w.Create(ctx, &leaf)).To(Succeed())
		})
		all := func(ranger.Key) bool { return true }
		// withChildren returns the names of the ranges that Derive marks.
		withChildren := func(
			ctx SpecContext,
			snap gorp.Snapshot,
			visible func(ranger.Key) bool,
		) set.Set[string] {
			GinkgoHelper()
			ranges := []ranger.Range{parent, child, leaf}
			Expect(svc.Derive(ctx, snap, ranges, visible)).To(Succeed())
			names := make(set.Set[string])
			for _, r := range ranges {
				if r.HasChildren {
					names.Add(r.Name)
				}
			}
			return names
		}

		It("Should mark the ranges that have nested ranges", func(ctx SpecContext) {
			Expect(withChildren(ctx, tx, all)).To(Equal(set.New("Parent")))
		})

		It("Should see a range moved in the same transaction", func(ctx SpecContext) {
			Expect(w.Apply(ctx, child.Key, ranger.NewSetAction(ranger.SetPayload{
				Path:  []string{"parent"},
				Value: leaf.Key,
			}))).To(Succeed())
			Expect(withChildren(ctx, tx, all)).To(Equal(set.New("Leaf")))
		})

		It("Should count only the nested ranges that visible accepts", func(
			ctx SpecContext,
		) {
			hidden := func(k ranger.Key) bool { return k != child.Key }
			Expect(withChildren(ctx, tx, hidden)).To(BeEmpty())
			sibling := ranger.Range{
				Name:      "Sibling",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
				Parent:    &parent.Key,
			}
			Expect(w.Create(ctx, &sibling)).To(Succeed())
			Expect(withChildren(ctx, tx, hidden)).To(Equal(set.New("Parent")))
		})

		It("Should neither store nor echo the derived field", func(ctx SpecContext) {
			stored := ranger.Range{
				Name:        "Stored",
				TimeRange:   telem.SecondTS.SpanRange(telem.Second),
				HasChildren: true,
			}
			Expect(w.Create(ctx, &stored)).To(Succeed())
			Expect(stored.HasChildren).To(BeFalse())
			var res ranger.Range
			Expect(svc.NewRetrieve().
				Where(ranger.MatchKeys(stored.Key)).
				Entry(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(res.HasChildren).To(BeFalse())
		})

		It(
			"Should mark a range that its batch nests a range in",
			func(ctx SpecContext) {
				tr := telem.SecondTS.SpanRange(telem.Second)
				parentKey := uuid.New()
				batch := []ranger.Range{
					{Key: parentKey, Name: "Parent", TimeRange: tr, HasChildren: true},
					{
						Name:        "Child",
						TimeRange:   tr,
						Parent:      &parentKey,
						HasChildren: true,
					},
					{Name: "Other", TimeRange: tr, HasChildren: true},
				}
				Expect(w.CreateMany(ctx, &batch)).To(Succeed())
				Expect(lo.Map(batch, func(r ranger.Range, _ int) bool {
					return r.HasChildren
				})).To(Equal([]bool{true, false, false}))
			},
		)
	})

	Describe("Retrieve", func() {
		It("Should retrieve a range by its key", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(
				svc.NewRetrieve().
					Where(ranger.MatchKeys(r.Key)).
					Entry(&retrieveR).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve a range by its name", func(ctx SpecContext) {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(
				svc.NewRetrieve().
					Where(ranger.MatchNames(r.Name)).
					Entry(&retrieveR).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It(
			"Should retrieve only the exact name of names that differ in case",
			func(ctx SpecContext) {
				tr := telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				}
				upper := &ranger.Range{Name: "CaseRange", TimeRange: tr}
				lower := &ranger.Range{Name: "caserange", TimeRange: tr}
				Expect(svc.NewWriter(tx).Create(ctx, upper)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, lower)).To(Succeed())
				var res []ranger.Range
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchNames(upper.Name)).
						Entries(&res).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0].Key).To(Equal(upper.Key))
			},
		)
		It(
			"Should retrieve any ranges that overlap a given time range",
			func(ctx SpecContext) {
				r := &ranger.Range{
					Name: "Range",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
				var retrieveR ranger.Range
				Expect(svc.NewRetrieve().Where(ranger.MatchOverlaps(telem.TimeRange{
					Start: telem.TimeStamp(7 * telem.Second),
					End:   telem.TimeStamp(9 * telem.Second),
				})).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
				Expect(retrieveR.Key).To(Equal(r.Key))
			},
		)
		It("Should retrieve ranges that have a specific label", func(ctx SpecContext) {
			l := &label.Label{Name: "TestLabel"}
			Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
			r1 := &ranger.Range{
				Name:      "LabeledRange",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			r2 := &ranger.Range{
				Name:      "UnlabeledRange",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).Create(ctx, r1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, r2)).To(Succeed())
			Expect(
				svc.NewWriter(tx).AddLabels(ctx, r1.Key, []label.Key{l.Key}),
			).To(Succeed())
			var results []ranger.Range
			Expect(
				svc.NewRetrieve().
					Where(ranger.MatchLabels(l.Key)).
					Entries(&results).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Key).To(Equal(r1.Key))
		})
		It(
			"Should return empty when no ranges have the specified label",
			func(ctx SpecContext) {
				l := &label.Label{Name: "UnusedLabel"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
				r := &ranger.Range{
					Name:      "RangeWithoutLabels",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
				var results []ranger.Range
				Expect(
					svc.NewRetrieve().
						Where(ranger.MatchLabels(l.Key)).
						Entries(&results).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(results).To(BeEmpty())
			},
		)
		It(
			"Should match ranges that have any of multiple provided labels",
			func(ctx SpecContext) {
				la := &label.Label{Name: "LabelA"}
				lb := &label.Label{Name: "LabelB"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, la)).To(Succeed())
				Expect(labelSvc.NewWriter(tx).Create(ctx, lb)).To(Succeed())
				rA := &ranger.Range{
					Name:      "RA",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				rB := &ranger.Range{
					Name:      "RB",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				rNone := &ranger.Range{
					Name:      "RNone",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(svc.NewWriter(tx).Create(ctx, rA)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rB)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rNone)).To(Succeed())
				Expect(
					svc.NewWriter(tx).
						AddLabels(ctx, rA.Key, []label.Key{la.Key}),
				).To(Succeed())
				Expect(
					svc.NewWriter(tx).
						AddLabels(ctx, rB.Key, []label.Key{lb.Key}),
				).To(Succeed())
				var results []ranger.Range
				Expect(
					svc.NewRetrieve().
						Where(ranger.Or(ranger.MatchLabels(la.Key), ranger.MatchLabels(lb.Key))).
						Entries(&results).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(results).To(HaveLen(2))
				keys := []uuid.UUID{results[0].Key, results[1].Key}
				Expect(keys).To(ContainElements(rA.Key, rB.Key))
			},
		)
		It(
			"Should compose MatchOverlaps and MatchLabels under And via Where",
			func(ctx SpecContext) {
				l := &label.Label{Name: "AndLabel"}
				Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
				rHit := &ranger.Range{
					Name: "AndHit",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				rWrongTime := &ranger.Range{
					Name: "AndWrongTime",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(20 * telem.Second),
						End:   telem.TimeStamp(25 * telem.Second),
					},
				}
				rWrongLabel := &ranger.Range{
					Name: "AndWrongLabel",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, rHit)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rWrongTime)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rWrongLabel)).To(Succeed())
				Expect(
					svc.NewWriter(tx).
						AddLabels(ctx, rHit.Key, []label.Key{l.Key}),
				).To(Succeed())
				Expect(
					svc.NewWriter(tx).
						AddLabels(ctx, rWrongTime.Key, []label.Key{l.Key}),
				).To(Succeed())
				var results []ranger.Range
				Expect(svc.NewRetrieve().Where(ranger.And(
					ranger.MatchOverlaps(
						telem.TimeRange{
							Start: telem.TimeStamp(7 * telem.Second),
							End:   telem.TimeStamp(9 * telem.Second),
						},
					),
					ranger.MatchLabels(l.Key),
				)).Entries(&results).Exec(ctx, tx)).To(Succeed())
				Expect(results).To(HaveLen(1))
				Expect(results[0].Key).To(Equal(rHit.Key))
			},
		)
		It(
			"Should compose MatchOverlaps and MatchNames under Or via Where",
			func(ctx SpecContext) {
				rOverlap := &ranger.Range{
					Name: "OrOverlap",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(5 * telem.Second),
						End:   telem.TimeStamp(10 * telem.Second),
					},
				}
				rByName := &ranger.Range{
					Name: "OrByName",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(100 * telem.Second),
						End:   telem.TimeStamp(110 * telem.Second),
					},
				}
				rNeither := &ranger.Range{
					Name: "OrNeither",
					TimeRange: telem.TimeRange{
						Start: telem.TimeStamp(200 * telem.Second),
						End:   telem.TimeStamp(210 * telem.Second),
					},
				}
				Expect(svc.NewWriter(tx).Create(ctx, rOverlap)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rByName)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, rNeither)).To(Succeed())
				var results []ranger.Range
				Expect(svc.NewRetrieve().Where(ranger.Or(
					ranger.MatchOverlaps(
						telem.TimeRange{
							Start: telem.TimeStamp(7 * telem.Second),
							End:   telem.TimeStamp(9 * telem.Second),
						},
					),
					ranger.MatchNames("OrByName"),
				)).Entries(&results).Exec(ctx, tx)).To(Succeed())
				Expect(results).To(HaveLen(2))
				keys := []uuid.UUID{results[0].Key, results[1].Key}
				Expect(keys).To(ContainElements(rOverlap.Key, rByName.Key))
			},
		)
		It("Should invert a filter with Not", func(ctx SpecContext) {
			l := &label.Label{Name: "NotLabel"}
			Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
			rLabeled := &ranger.Range{
				Name:      "NotLabeled",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			rPlain := &ranger.Range{
				Name:      "NotPlain",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).Create(ctx, rLabeled)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, rPlain)).To(Succeed())
			Expect(
				svc.NewWriter(tx).
					AddLabels(ctx, rLabeled.Key, []label.Key{l.Key}),
			).To(Succeed())
			var results []ranger.Range
			Expect(svc.NewRetrieve().Where(
				ranger.And(
					ranger.MatchNames("NotLabeled", "NotPlain"),
					ranger.Not(ranger.MatchLabels(l.Key)),
				),
			).Entries(&results).Exec(ctx, tx)).To(Succeed())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Key).To(Equal(rPlain.Key))
		})
		It("Should count ranges with Count", func(ctx SpecContext) {
			for i := range 3 {
				r := &ranger.Range{
					Name:      fmt.Sprintf("CountRange%d", i),
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			}
			n := MustSucceed(svc.NewRetrieve().
				Where(ranger.MatchNames("CountRange0", "CountRange1", "CountRange2")).
				Count(ctx, tx))
			Expect(n).To(Equal(3))
		})
		It("Should report existence with Exists", func(ctx SpecContext) {
			r := &ranger.Range{
				Name:      "ExistsRange",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			Expect(MustSucceed(svc.NewRetrieve().
				Where(ranger.MatchNames("ExistsRange")).
				Exists(ctx, tx))).To(BeTrue())
			Expect(MustSucceed(svc.NewRetrieve().
				Where(ranger.MatchNames("NopeRange")).
				Exists(ctx, tx))).To(BeFalse())
		})
		It("Should page with Limit and After", func(ctx SpecContext) {
			names := make([]string, 5)
			created := make([]ranger.Key, 5)
			for i := range names {
				names[i] = fmt.Sprintf("PageRange%d", i)
				r := &ranger.Range{
					Name:      names[i],
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
				created[i] = r.Key
			}
			page := func(q ranger.Retrieve) []ranger.Key {
				var res []ranger.Range
				Expect(q.Where(ranger.MatchNames(names...)).Limit(2).Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				return lo.Map(
					res,
					func(r ranger.Range, _ int) ranger.Key { return r.Key },
				)
			}
			first := page(svc.NewRetrieve())
			second := page(svc.NewRetrieve().After(first[1]))
			third := page(svc.NewRetrieve().After(second[1]))
			Expect(third).To(HaveLen(1))
			Expect(slices.Concat(first, second, third)).To(ConsistOf(created))
		})
		Describe("Search", func() {
			It(
				"Should execute the search path via Exec without error",
				func(ctx SpecContext) {
					r := &ranger.Range{
						Name:      "SearchableRangeAlpha",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
					var results []ranger.Range
					Expect(svc.NewRetrieve().
						Search("SearchableRangeAlpha").
						Entries(&results).
						Exec(ctx, tx)).To(Succeed())
				},
			)
			It("Should execute Count through the search path", func(ctx SpecContext) {
				MustSucceed(svc.NewRetrieve().
					Search("SearchableRangeAlpha").
					Count(ctx, tx))
			})
			It("Should execute Exists through the search path", func(ctx SpecContext) {
				MustSucceed(svc.NewRetrieve().
					Search("SearchableRangeAlpha").
					Exists(ctx, tx))
			})
		})
	})
})

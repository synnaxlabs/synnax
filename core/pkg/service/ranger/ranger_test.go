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
	"io"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ranger", Ordered, func() {
	var (
		db       *gorp.DB
		svc      *ranger.Service
		ctx      context.Context
		w        ranger.Writer
		otg      *ontology.Ontology
		tx       gorp.Tx
		closer   io.Closer
		labelSvc *label.Service
	)
	BeforeAll(func() {
		db = gorp.Wrap(memkv.New())
		ctx = context.Background()
		otg = MustSucceed(ontology.Open(ctx, ontology.Config{
			DB:           db,
			EnableSearch: config.True(),
		}))
		g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}))
		labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{DB: db, Ontology: otg, Group: g}))
		svc = MustSucceed(ranger.OpenService(ctx, ranger.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    labelSvc,
		}))
		Expect(otg.InitializeSearchIndex(ctx)).To(Succeed())
		closer = xio.MultiCloser{db, otg, g, svc}
	})
	AfterAll(func() {
		Expect(closer.Close()).To(Succeed())
	})
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
	})
	AfterEach(func() {
		Expect(tx.Close()).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a new range", func() {
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
		It("should return an error if the time range is invalid", func() {
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
		It("should create a range with start equal to end", func() {
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
		It("Should not override the UUID if it is already set", func() {
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
			It("Should set a custom parent for the range", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateWithParent(ctx, r, parent.OntologyID())).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should NOT re-set the custom parent when the range exists but no parent is provided", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r := &ranger.Range{
					Name:      "Range",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateWithParent(ctx, r, parent.OntologyID())).To(Succeed())
				Expect(w.Create(ctx, r)).To(Succeed())
				var res ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entry(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res.ID.Key).To(Equal(r.Key.String()))
			})
			It("Should change the custom parent when the range exists and a new parent is provided", func() {
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
				}
				Expect(w.CreateWithParent(ctx, r, parent1.OntologyID())).To(Succeed())
				Expect(w.CreateWithParent(ctx, r, parent2.OntologyID())).To(Succeed())
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
					Exec(ctx, tx)).To(HaveOccurredAs(query.ErrNotFound))
			})
			It("Should create multiple ranges with the same parent", func() {
				parent := ranger.Range{
					Name:      "Parent",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.Create(ctx, &parent)).To(Succeed())
				r1 := ranger.Range{
					Name:      "Range1",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				r2 := ranger.Range{
					Name:      "Range2",
					TimeRange: telem.SecondTS.SpanRange(telem.Second),
				}
				Expect(w.CreateManyWithParent(ctx, &[]ranger.Range{r1, r2}, parent.OntologyID())).To(Succeed())
				var res []ontology.Resource
				Expect(otg.NewRetrieve().
					WhereIDs(parent.OntologyID()).
					TraverseTo(ontology.ChildrenTraverser).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(HaveLen(2))
			})
			Context("RetrieveParentKey Method", func() {
				It("Should get the parent key of the range", func() {
					parent := ranger.Range{
						Name:      "Parent",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.Create(ctx, &parent)).To(Succeed())
					r := ranger.Range{
						Name:      "Range",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
					pKey := MustSucceed(svc.RetrieveParentKey(ctx, r.Key, tx))
					Expect(pKey).To(Equal(parent.Key))
				})
				It("Should return an error if the range has no parent", func() {
					p := ranger.Range{
						Name:      "Parent",
						TimeRange: telem.SecondTS.SpanRange(telem.Second),
					}
					Expect(w.Create(ctx, &p)).To(Succeed())
					_, err := svc.RetrieveParentKey(ctx, p.Key, tx)
					Expect(err).To(HaveOccurredAs(query.ErrNotFound))
				})
			})
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve a range by its key", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve a range by its name", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereNames(r.Name).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve any ranges that overlap a given time range", func() {
			r := &ranger.Range{
				Name: "Range",
				TimeRange: telem.TimeRange{
					Start: telem.TimeStamp(5 * telem.Second),
					End:   telem.TimeStamp(10 * telem.Second),
				},
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereOverlapsWith(telem.TimeRange{
				Start: telem.TimeStamp(7 * telem.Second),
				End:   telem.TimeStamp(9 * telem.Second),
			}).Entry(&retrieveR).Exec(ctx, tx)).To(Succeed())
			Expect(retrieveR.Key).To(Equal(r.Key))
		})
		It("Should retrieve ranges that have a specific label", func() {
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
			Expect(labelSvc.NewWriter(tx).Label(ctx, r1.OntologyID(), []uuid.UUID{l.Key})).To(Succeed())
			var results []ranger.Range
			Expect(svc.NewRetrieve().WhereHasLabels(l.Key).Entries(&results).Exec(ctx, tx)).To(Succeed())
			Expect(results).To(HaveLen(1))
			Expect(results[0].Key).To(Equal(r1.Key))
		})
		It("Should return empty when no ranges have the specified label", func() {
			l := &label.Label{Name: "UnusedLabel"}
			Expect(labelSvc.NewWriter(tx).Create(ctx, l)).To(Succeed())
			r := &ranger.Range{
				Name:      "RangeWithoutLabels",
				TimeRange: telem.SecondTS.SpanRange(telem.Second),
			}
			Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			var results []ranger.Range
			Expect(svc.NewRetrieve().WhereHasLabels(l.Key).Entries(&results).Exec(ctx, tx)).To(Succeed())
			Expect(results).To(BeEmpty())
		})
	})

	Describe("Delete", func() {
		It("Should delete a range by its key", func() {
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
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
		It("Should delete all child ranges when a range is deleted", func() {
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
			}
			Expect(svc.NewWriter(tx).CreateWithParent(ctx, &r, parent.OntologyID())).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, parent.Key)).To(Succeed())
			var retrieveR ranger.Range
			Expect(svc.NewRetrieve().WhereKeys(r.Key).Entry(&retrieveR).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func rangeView(name string) view.View {
	return view.View{
		Variant: view.RangeView{
			Base:   view.Base{Key: uuid.New(), Name: name},
			Sort:   view.RangeSortNewest,
			Layout: view.RangeLayoutTree,
		},
	}
}

func statusView(name string) view.View {
	return view.View{
		Variant: view.StatusView{Base: view.Base{Key: uuid.New(), Name: name}},
	}
}

var _ = Describe("View", func() {
	var (
		svc *view.Service
		w   view.Writer
		tx  gorp.Tx
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(view.OpenService(ctx, view.ServiceConfig{
			DB: db,
		}))
	})
	BeforeEach(func() {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})

	Describe("Writer", func() {
		Describe("Create", func() {
			It(
				"Should create a new view with an auto-generated key",
				func(ctx SpecContext) {
					s := &view.View{Variant: view.RangeView{
						Base:   view.Base{Name: "Test"},
						Sort:   view.RangeSortNewest,
						Layout: view.RangeLayoutTree,
					}}
					Expect(w.Create(ctx, s)).To(Succeed())
					Expect(s.GorpKey()).ToNot(Equal(uuid.Nil()))
				},
			)
			It("Should reject a view with no variant", func(ctx SpecContext) {
				Expect(w.Create(ctx, &view.View{})).
					To(MatchError(ContainSubstring("type: required")))
			})
			It(
				"Should reject a status view with an invalid variant",
				func(ctx SpecContext) {
					s := view.View{Variant: view.StatusView{
						Base: view.Base{Name: "Bogus"},
						Where: &status.Where{Or: []status.Where{
							{Variant: &filter.Set[status.Variant]{
								In: []status.Variant{"bogus"},
							}},
						}},
					}}
					Expect(w.Create(ctx, &s)).To(MatchError(ContainSubstring(
						`where.or.0.variant.in.0: invalid variant "bogus"`,
					)))
				},
			)
			It(
				"Should reject a range view with an invalid regex",
				func(ctx SpecContext) {
					s := view.View{Variant: view.RangeView{
						Base:   view.Base{Name: "Bad"},
						Where:  &ranger.Where{Name: &filter.String{Regex: new("(")}},
						Sort:   view.RangeSortNewest,
						Layout: view.RangeLayoutTree,
					}}
					Expect(w.Create(ctx, &s)).To(MatchError(ContainSubstring(
						"where.name.regex: error parsing regexp: missing closing )",
					)))
				},
			)
			It(
				"Should reject a range view with an invalid sort",
				func(ctx SpecContext) {
					s := view.View{Variant: view.RangeView{
						Base:   view.Base{Name: "Bad"},
						Sort:   "random",
						Layout: view.RangeLayoutTree,
					}}
					Expect(w.Create(ctx, &s)).To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("sort: invalid sort: random")),
					))
				},
			)
			It("Should reject a view with no name", func(ctx SpecContext) {
				s := rangeView("")
				Expect(
					w.Create(ctx, &s),
				).To(MatchError(ContainSubstring("name: required")))
			})
			It("Should refuse a view whose key exists", func(ctx SpecContext) {
				stored := rangeView("Test View")
				Expect(w.Create(ctx, &stored)).To(Succeed())
				again := view.View{Variant: view.StatusView{
					Base: view.Base{Key: stored.GorpKey(), Name: "Updated Name"},
				}}
				Expect(w.Create(ctx, &again)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"view %s already exists",
						stored.GorpKey(),
					)),
				))
				var retrieved view.View
				Expect(
					svc.NewRetrieve().
						Where(view.MatchKeys(stored.GorpKey())).
						Entry(&retrieved).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(retrieved).To(Equal(stored))
			})
		})

		Describe("CreateMany", func() {
			It("Should create multiple views", func(ctx SpecContext) {
				views := []view.View{rangeView("View 1"), statusView("View 2")}
				Expect(w.CreateMany(ctx, &views)).To(Succeed())

				var retrieved []view.View
				Expect(svc.NewRetrieve().Where(view.MatchKeys(
					views[0].GorpKey(),
					views[1].GorpKey(),
				)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
				Expect(retrieved).To(HaveLen(2))
			})
		})

		Describe("Rename", func() {
			It(
				"Should rename a view, keeping every other field",
				func(ctx SpecContext) {
					s := view.View{Variant: view.StatusView{
						Base: view.Base{Key: uuid.New(), Name: "Before"},
						Where: &status.Where{Variant: &filter.Set[status.Variant]{
							In: []status.Variant{status.VariantSuccess},
						}},
					}}
					Expect(w.Create(ctx, &s)).To(Succeed())
					Expect(w.Rename(ctx, s.GorpKey(), "After")).To(Succeed())
					var res view.View
					Expect(svc.NewRetrieve().
						Where(view.MatchKeys(s.GorpKey())).
						Entry(&res).
						Exec(ctx, tx)).To(Succeed())
					s.SetBase(view.Base{Key: s.GorpKey(), Name: "After"})
					Expect(res).To(Equal(s))
				},
			)
			It(
				"Should return a validation error for an empty name",
				func(ctx SpecContext) {
					s := rangeView("Before")
					Expect(w.Create(ctx, &s)).To(Succeed())
					Expect(w.Rename(ctx, s.GorpKey(), "")).
						To(MatchError(ContainSubstring("name: required")))
				},
			)
			It(
				"Should return a not found error for a missing view",
				func(ctx SpecContext) {
					Expect(w.Rename(ctx, uuid.New(), "After")).
						To(MatchError(query.ErrNotFound))
				},
			)
		})

		Describe("Delete", func() {
			It("Should delete a view", func(ctx SpecContext) {
				s := rangeView("To Delete")
				Expect(w.Create(ctx, &s)).To(Succeed())
				Expect(w.Delete(ctx, s.GorpKey())).To(Succeed())

				Expect(
					svc.NewRetrieve().
						Where(view.MatchKeys(s.GorpKey())).
						Entry(&view.View{}).
						Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
			})

			It("Should be idempotent", func(ctx SpecContext) {
				Expect(w.Delete(ctx, uuid.New())).To(Succeed())
			})

			It("Should delete multiple views", func(ctx SpecContext) {
				views := []view.View{rangeView("Del 1"), rangeView("Del 2")}
				Expect(w.CreateMany(ctx, &views)).To(Succeed())
				Expect(
					w.Delete(ctx, views[0].GorpKey(), views[1].GorpKey()),
				).To(Succeed())

				Expect(
					svc.NewRetrieve().
						Where(view.MatchKeys(views[0].GorpKey(), views[1].GorpKey())).
						Exec(ctx, tx),
				).To(MatchError(query.ErrNotFound))
			})
		})
	})

	Describe("Retrieve", func() {
		var views []view.View
		BeforeEach(func(ctx SpecContext) {
			views = []view.View{
				rangeView("View A"),
				rangeView("View B"),
				statusView("View C"),
			}
			Expect(w.CreateMany(ctx, &views)).To(Succeed())
		})

		Describe("WhereKeys", func() {
			It("Should retrieve view by key", func(ctx SpecContext) {
				var s view.View
				Expect(
					svc.NewRetrieve().
						Where(view.MatchKeys(views[0].GorpKey())).
						Entry(&s).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(s.GorpKey()).To(Equal(views[0].GorpKey()))
				Expect(s.Base().Name).To(Equal("View A"))
			})

			It("Should retrieve multiple views by keys", func(ctx SpecContext) {
				var resViews []view.View
				Expect(
					svc.NewRetrieve().
						Where(view.MatchKeys(views[0].GorpKey(), views[1].GorpKey())).
						Entries(&resViews).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(resViews).To(HaveLen(2))
			})
		})

		Describe("WhereTypes", func() {
			It("Should retrieve views by type", func(ctx SpecContext) {
				var resViews []view.View
				Expect(
					svc.NewRetrieve().
						Where(view.MatchTypes(view.TypeRange)).
						Entries(&resViews).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(resViews).To(HaveLen(2))
			})
			It("Should also retrieve views by type and key", func(ctx SpecContext) {
				var resViews []view.View
				Expect(
					svc.NewRetrieve().
						Where(view.MatchTypes(view.TypeRange)).
						Where(view.MatchKeys(views[0].GorpKey())).
						Entries(&resViews).
						Exec(ctx, tx),
				).To(Succeed())
				Expect(resViews).To(HaveLen(1))
				Expect(resViews[0].GorpKey()).To(Equal(views[0].GorpKey()))
			})
		})

		Describe("MatchWhere", func() {
			It("Should match views by type and name", func(ctx SpecContext) {
				var resViews []view.View
				Expect(svc.NewRetrieve().
					Where(MustSucceed(view.MatchWhere(view.Where{
						Type: &filter.Set[view.Type]{Eq: new(view.TypeRange)},
						Name: &filter.String{Regex: new("B$")},
					}))).
					Entries(&resViews).
					Exec(ctx, tx),
				).To(Succeed())
				Expect(resViews).To(HaveLen(1))
				Expect(resViews[0].GorpKey()).To(Equal(views[1].GorpKey()))
			})
		})

		Describe("Limit", func() {
			It("Should limit results", func(ctx SpecContext) {
				var views []view.View
				Expect(
					svc.NewRetrieve().Limit(2).Entries(&views).Exec(ctx, tx),
				).To(Succeed())
				Expect(views).To(HaveLen(2))
			})
		})

		Describe("Search", func() {
			It("Should search for views", func(ctx SpecContext) {
				var resViews []view.View
				Expect(tx.Commit(ctx)).To(Succeed())
				DeferCleanup(func(ctx SpecContext) {
					keys := make([]view.Key, len(views))
					for i, v := range views {
						keys[i] = v.GorpKey()
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx).Delete(ctx, keys...)
					})).To(Succeed())
				})
				Expect(
					svc.NewRetrieve().Search("View A").Entries(&resViews).Exec(ctx, db),
				).To(Succeed())
				Expect(resViews).To(HaveLen(1))
				Expect(resViews[0].GorpKey()).To(Equal(views[0].GorpKey()))
			})
		})
	})

	Describe("Search", func() {
		create := func(ctx SpecContext, name string) resource.ID {
			GinkgoHelper()
			v := rangeView(name)
			Expect(w.Create(ctx, &v)).To(Succeed())
			return view.ResourceID(v.GorpKey())
		}

		It(
			"Should return the matching views, closest match first",
			func(ctx SpecContext) {
				a := create(ctx, "turbopump test")
				b := create(ctx, "turbopump")
				q := search.Query[resource.ID]{Term: "turbopump"}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b, a}))
				q.Limit = 1
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{b}))
			},
		)

		It(
			"Should not match a staged view outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})
})

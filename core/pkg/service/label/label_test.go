// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Label", Ordered, func() {
	var (
		db  *gorp.DB
		svc *label.Service
		w   label.Writer
		tx  gorp.Tx
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
	})
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})
	Describe("Create", func() {
		It("Should create a new label", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: MustSucceed(color.FromHex("#000000")),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(l.Key).ToNot(Equal(label.Key(uuid.Nil())))
		})
		It(
			"Should return a validation error when the name is empty",
			func(ctx SpecContext) {
				Expect(w.Create(ctx, &label.Label{})).
					To(MatchError(ContainSubstring("name: required")))
			},
		)
		It("Should refuse a label whose key exists", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: MustSucceed(color.FromHex("#000000")),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			again := label.Label{Key: l.Key, Name: "Renamed"}
			Expect(w.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("label %s already exists", l.Key)),
			))
		})
		It("Should create many labels", func(ctx SpecContext) {
			ls := []label.Label{
				{
					Name:  "Label1",
					Color: MustSucceed(color.FromHex("#000000")),
				},
				{
					Name:  "label",
					Color: MustSucceed(color.FromHex("#000000")),
				},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
			for _, l := range ls {
				Expect(l.Key).ToNot(Equal(label.Key(uuid.Nil())))
			}
		})
	})
	Describe("Delete", func() {
		It("Should delete a label", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Label",
				Color: MustSucceed(color.FromHex("#000000")),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(w.Delete(ctx, l.Key)).To(Succeed())
			Expect(
				svc.NewRetrieve().Where(label.MatchKeys(l.Key)).Exec(ctx, db),
			).To(MatchError(query.ErrNotFound))
		})
		It("Should delete many labels", func(ctx SpecContext) {
			ls := []label.Label{
				{
					Name:  "Label1",
					Color: MustSucceed(color.FromHex("#000000")),
				},
				{
					Name:  "label",
					Color: MustSucceed(color.FromHex("#000000")),
				},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
			Expect(w.Delete(ctx, ls[0].Key, ls[1].Key)).To(Succeed())
			for _, l := range ls {
				Expect(
					svc.NewRetrieve().Where(label.MatchKeys(l.Key)).Exec(ctx, db),
				).To(MatchError(query.ErrNotFound))
			}
		})
	})
	Describe("MatchNames", func() {
		It("Should retrieve a label by its name", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "match-name-target",
				Color: MustSucceed(color.FromHex("#000000")),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			var got label.Label
			Expect(svc.NewRetrieve().
				Where(label.MatchNames("match-name-target")).
				Entry(&got).
				Exec(ctx, tx)).To(Succeed())
			Expect(got.Key).To(Equal(l.Key))
		})
		It(
			"Should retrieve labels matching any of the provided names",
			func(ctx SpecContext) {
				ls := []label.Label{
					{Name: "mn-a", Color: MustSucceed(color.FromHex("#000000"))},
					{Name: "mn-b", Color: MustSucceed(color.FromHex("#000000"))},
					{Name: "mn-c", Color: MustSucceed(color.FromHex("#000000"))},
				}
				Expect(w.CreateMany(ctx, &ls)).To(Succeed())
				var got []label.Label
				Expect(svc.NewRetrieve().
					Where(label.MatchNames("mn-a", "mn-b")).
					Entries(&got).
					Exec(ctx, tx)).To(Succeed())
				Expect(got).To(HaveLen(2))
			},
		)
	})
	Describe("Limit", func() {
		BeforeEach(func(ctx SpecContext) {
			ls := []label.Label{
				{Name: "page-a", Color: MustSucceed(color.FromHex("#000000"))},
				{Name: "page-b", Color: MustSucceed(color.FromHex("#000000"))},
				{Name: "page-c", Color: MustSucceed(color.FromHex("#000000"))},
			}
			Expect(w.CreateMany(ctx, &ls)).To(Succeed())
		})
		It("Should limit the number of returned labels", func(ctx SpecContext) {
			var got []label.Label
			Expect(svc.NewRetrieve().Limit(2).Entries(&got).Exec(ctx, tx)).To(Succeed())
			Expect(got).To(HaveLen(2))
		})
	})
	Describe("Search", func() {
		It("Should fuzzy search labels by name", func(ctx SpecContext) {
			l := &label.Label{
				Name:  "Searchable Critical Label",
				Color: MustSucceed(color.FromHex("#000000")),
			}
			Expect(w.Create(ctx, l)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			tx = DeferClose(db.OpenTx())
			w = svc.NewWriter(tx)
			var got []label.Label
			Expect(svc.NewRetrieve().
				Search("Searchable Critical").
				Entries(&got).
				Exec(ctx, db)).To(Succeed())
			Expect(got).ToNot(BeEmpty())
			Expect(got[0].Name).To(ContainSubstring("Searchable"))
		})
	})
	Describe("Search", func() {
		create := func(ctx SpecContext, name string) resource.ID {
			GinkgoHelper()
			l := label.Label{Name: name, Color: MustSucceed(color.FromHex("#000000"))}
			Expect(w.Create(ctx, &l)).To(Succeed())
			return l.ResourceID()
		}

		It(
			"Should return the matching labels, closest match first",
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
			"Should not match a staged label outside its transaction",
			func(ctx SpecContext) {
				create(ctx, "injector")
				q := search.Query[resource.ID]{Term: "injector"}
				Expect(
					MustSucceed(svc.Resources().Search(ctx, db, q)).Keys,
				).To(BeEmpty())
			},
		)
	})

	Describe("Changes", func() {
		It("Should return the label that a commit creates", func(ctx SpecContext) {
			var names []string
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range svc.Changes(ctx, c) {
						names = append(names, ch.Value.Name)
					}
				}),
			)
			Expect(w.Create(ctx, &label.Label{Name: "observed"})).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(names).To(Equal([]string{"observed"}))
		})
	})
})

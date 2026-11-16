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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/view"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[view.Key, view.Action]
	var (
		db  *gorp.DB
		svc *view.Service
		v   view.View
	)
	write := func(ctx context.Context, fn func(view.Writer) error) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return fn(svc.NewWriter(tx).WithDispatchKey("dk"))
		})
	}
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(view.OpenService(ctx, view.ServiceConfig{DB: db}))
		v = statusView("Before")
		Expect(write(ctx, func(w view.Writer) error { return w.Create(ctx, &v) })).
			To(Succeed())
	})
	set := func(path string, value any) view.Action {
		return view.NewSetAction(view.SetPayload{Path: []string{path}, Value: value})
	}
	apply := func(ctx context.Context, acts ...view.Action) error {
		return write(ctx, func(w view.Writer) error {
			return w.Apply(ctx, v.GorpKey(), acts...)
		})
	}
	record := func() *Recorder[view.Key, view.Action] {
		rec := &Recorder[view.Key, view.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	retrieve := func(ctx context.Context) view.View {
		GinkgoHelper()
		var res view.View
		Expect(svc.NewRetrieve().
			Where(view.MatchKeys(v.GorpKey())).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}
	where := status.Where{Variant: &filter.Set[status.Variant]{
		In: []status.Variant{status.VariantSuccess},
	}}

	It("Should store the edited name and where", func(ctx SpecContext) {
		Expect(apply(ctx, set("name", "After"), set("where", where))).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(view.View{Variant: view.StatusView{
			Base:  view.Base{Key: v.GorpKey(), Name: "After"},
			Where: &where,
		}}))
	})

	It("Should store the edited icon and color", func(ctx SpecContext) {
		c := MustSucceed(color.Parse("#00ff00"))
		Expect(apply(ctx, set("icon", "bell"), set("color", c))).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(view.View{Variant: view.StatusView{
			Base: view.Base{
				Key:   v.GorpKey(),
				Name:  "Before",
				Icon:  new("bell"),
				Color: &c,
			},
		}}))
	})

	DescribeTable("Should refuse an edit", func(
		ctx SpecContext,
		act view.Action,
		matcher string,
	) {
		err := apply(ctx, act)
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(ContainSubstring(matcher)))
		Expect(retrieve(ctx)).To(Equal(v))
	},
		Entry("of the type", set("type", "range"), `unknown object member name "type"`),
		Entry(
			"to a where that does not decode",
			set("where", 1),
			"unmarshal JSON number into Go v2.Where",
		),
		Entry("to an invalid where", set("where", status.Where{
			Variant: &filter.Set[status.Variant]{In: []status.Variant{"bogus"}},
		}), `where.variant.in.0: invalid variant "bogus"`),
	)

	Describe("Range view", func() {
		BeforeEach(func(ctx SpecContext) {
			v = rangeView("Ranges")
			Expect(write(ctx, func(w view.Writer) error { return w.Create(ctx, &v) })).
				To(Succeed())
		})

		It(
			"Should store the edited icon, color, sort, and layout",
			func(ctx SpecContext) {
				c := MustSucceed(color.Parse("#ff0000"))
				Expect(apply(
					ctx,
					set("icon", "star"),
					set("color", c),
					set("sort", view.RangeSortOldest),
					set("layout", view.RangeLayoutTimeline),
				)).To(Succeed())
				Expect(retrieve(ctx)).To(Equal(view.View{Variant: view.RangeView{
					Base: view.Base{
						Key:   v.GorpKey(),
						Name:  "Ranges",
						Icon:  new("star"),
						Color: &c,
					},
					Sort:   view.RangeSortOldest,
					Layout: view.RangeLayoutTimeline,
				}}))
			},
		)

		DescribeTable(
			"Should refuse an invalid edit",
			func(ctx SpecContext, act view.Action, matcher string) {
				Expect(apply(ctx, act)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(matcher)),
				))
				Expect(retrieve(ctx)).To(Equal(v))
			},
			Entry("of the sort", set("sort", "random"), "sort: invalid sort: random"),
			Entry(
				"of the sort to null",
				set("sort", nil),
				"[sort]: the document requires a value",
			),
			Entry(
				"that removes the sort",
				view.NewRemoveAction(view.RemovePayload{Key: "sort"}),
				"[sort]: the document requires a value",
			),
			Entry(
				"of the layout",
				set("layout", "grid"),
				"layout: invalid layout: grid",
			),
		)
	})

	It("Should broadcast a create and an applied batch", func(ctx SpecContext) {
		rec := record()
		created := rangeView("Created")
		Expect(write(ctx, func(w view.Writer) error {
			return w.Create(ctx, &created)
		})).To(Succeed())
		Expect(apply(ctx, set("name", "After"))).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{
				Key:         created.GorpKey(),
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions: []view.Action{
					view.NewCreateAction(view.CreatePayload{View: created}),
				},
			},
			{
				Key:         v.GorpKey(),
				DispatchKey: "dk",
				Seq:         frames[1].Seq,
				Actions:     []view.Action{set("name", "After")},
			},
		}))
	})

	It("Should broadcast the set that a rename applies", func(ctx SpecContext) {
		rec := record()
		Expect(write(ctx, func(w view.Writer) error {
			return w.Rename(ctx, v.GorpKey(), "After")
		})).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		Expect(frames[0].Actions).To(Equal([]view.Action{set("name", "After")}))
	})
})

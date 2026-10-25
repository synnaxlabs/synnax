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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[ranger.Key, ranger.Action]
	collect := func() *[]frame {
		GinkgoHelper()
		var frames []frame
		DeferCleanup(svc.OnAction(func(_ context.Context, f frame) {
			frames = append(frames, f)
		}))
		return &frames
	}
	set := func(path []string, value any) ranger.Action {
		return ranger.NewSetAction(ranger.SetPayload{Path: path, Value: value})
	}
	retrieve := func(ctx context.Context, key ranger.Key) ranger.Range {
		GinkgoHelper()
		var res ranger.Range
		Expect(svc.NewRetrieve().
			Where(ranger.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, nil)).To(Succeed())
		return res
	}
	// Dispatch commits on its own, so its ranges commit too and are deleted after each
	// spec.
	var hotFire ranger.Range
	BeforeEach(func(ctx SpecContext) {
		hotFire = ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
		Expect(svc.NewWriter(db).Create(ctx, &hotFire)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(db).Delete(ctx, hotFire.Key)).To(Succeed())
		})
	})

	It("Should store the edited name, time range, and color", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			set([]string{"name"}, "Cold flow"),
			set([]string{"time_range", "end"}, "9000000000"),
			set([]string{"color"}, "#ff0000"),
		})).To(Succeed())
		res := retrieve(ctx, hotFire.Key)
		Expect(res.Name).To(Equal("Cold flow"))
		Expect(res.TimeRange).To(Equal(telem.NewRangeSeconds(1, 9)))
		Expect(res.Color).To(Equal(new(color.MustFromHex("#ff0000"))))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		frames := collect()
		acts := []ranger.Action{set([]string{"name"}, "Cold flow")}
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk-1", acts)).To(Succeed())
		Expect(*frames).To(HaveLen(1))
		f := (*frames)[0]
		Expect(f.Key).To(Equal(hotFire.Key))
		Expect(f.DispatchKey).To(Equal("dk-1"))
		Expect(f.Seq).To(BeNumerically(">", 0))
		Expect(f.Actions).To(Equal(acts))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		frames := collect()
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			set([]string{"name"}, "Cold flow"),
			set([]string{"labels"}, []any{}),
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx, hotFire.Key).Name).To(Equal("Hot fire"))
		Expect(*frames).To(BeEmpty())
	})

	It("Should reject an empty name", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			set([]string{"name"}, ""),
		})).To(MatchError(ContainSubstring("name: required")))
	})

	It("Should reject an end before the start", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			set([]string{"time_range", "end"}, "0"),
		})).To(MatchError(
			ContainSubstring("time_range.start cannot be after time_range.end"),
		))
	})

	It("Should reject a create", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			ranger.NewCreateAction(ranger.CreatePayload{Range: hotFire}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("only the Core dispatches a range create")),
		))
	})

	It("Should return not found for a missing range", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, uuid.New(), "dk", []ranger.Action{
			set([]string{"name"}, "Cold flow"),
		})).To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Writer broadcasts", func() {
	type frame = actions.Scoped[ranger.Key, ranger.Action]
	collect := func() *[]frame {
		GinkgoHelper()
		var frames []frame
		DeferCleanup(svc.OnAction(func(_ context.Context, f frame) {
			frames = append(frames, f)
		}))
		return &frames
	}
	// commit commits the spec's transaction, deleting the given ranges after the spec,
	// and opens another transaction for the cleanup to close.
	commit := func(ctx context.Context, keys ...ranger.Key) {
		GinkgoHelper()
		Expect(tx.Commit(ctx)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(db).Delete(ctx, keys...)).To(Succeed())
		})
		tx = db.OpenTx()
		DeferCleanup(func() { Expect(tx.Close()).To(Succeed()) })
	}
	newRange := func() *ranger.Range {
		return &ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 5)}
	}
	// framesOf returns the frames that broadcast on the range with the given key.
	framesOf := func(frames *[]frame, key ranger.Key) []frame {
		var res []frame
		for _, f := range *frames {
			if f.Key == key {
				res = append(res, f)
			}
		}
		return res
	}

	It("Should broadcast a create without labels or parent", func(ctx SpecContext) {
		parent := newRange()
		Expect(w.Create(ctx, parent)).To(Succeed())
		l := label.Label{Key: uuid.New(), Name: "Label"}
		Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		frames := collect()
		r := newRange()
		r.Parent, r.Labels = &ranger.Range{Key: parent.Key}, []label.Label{l}
		Expect(w.WithDispatchKey("dk-create").Create(ctx, r)).To(Succeed())
		Expect(*frames).To(BeEmpty())
		commit(ctx, parent.Key, r.Key)
		DeferCleanup(func(ctx SpecContext) {
			Expect(labelSvc.NewWriter(db).Delete(ctx, l.Key)).To(Succeed())
		})
		created := framesOf(frames, r.Key)
		Expect(created).To(HaveLen(1))
		f := created[0]
		Expect(f.Key).To(Equal(r.Key))
		Expect(f.DispatchKey).To(Equal("dk-create"))
		Expect(f.Actions).To(Equal([]ranger.Action{
			ranger.NewCreateAction(ranger.CreatePayload{Range: ranger.Range{
				Key:       r.Key,
				Name:      r.Name,
				TimeRange: r.TimeRange,
			}}),
		}))
	})

	It("Should not broadcast a create that never commits", func(ctx SpecContext) {
		frames := collect()
		Expect(w.Create(ctx, newRange())).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		tx = db.OpenTx()
		DeferCleanup(func() { Expect(tx.Close()).To(Succeed()) })
		Expect(*frames).To(BeEmpty())
	})

	It("Should broadcast a set of the name on rename", func(ctx SpecContext) {
		r := newRange()
		Expect(w.Create(ctx, r)).To(Succeed())
		frames := collect()
		Expect(w.Rename(ctx, r.Key, "Cold flow")).To(Succeed())
		commit(ctx, r.Key)
		renamed := framesOf(frames, r.Key)
		Expect(renamed).To(HaveLen(2))
		Expect(renamed[1].Actions).To(Equal([]ranger.Action{
			ranger.NewSetAction(ranger.SetPayload{
				Path:  []string{"name"},
				Value: "Cold flow",
			}),
		}))
	})

	It("Should broadcast a set of the end on set end", func(ctx SpecContext) {
		r := newRange()
		Expect(w.Create(ctx, r)).To(Succeed())
		frames := collect()
		Expect(w.SetEnd(ctx, r.Key, telem.SecondTS*9)).To(Succeed())
		commit(ctx, r.Key)
		ended := framesOf(frames, r.Key)
		Expect(ended).To(HaveLen(2))
		Expect(ended[1].Actions).To(Equal([]ranger.Action{
			ranger.NewSetAction(ranger.SetPayload{
				Path:  []string{"time_range", "end"},
				Value: telem.SecondTS * 9,
			}),
		}))
	})
})

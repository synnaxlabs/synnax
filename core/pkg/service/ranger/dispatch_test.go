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
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
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
		Expect(res.Color).To(Equal(new(MustSucceed(color.FromHex("#ff0000")))))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		acts := []ranger.Action{set([]string{"name"}, "Cold flow")}
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk-1", acts)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		f := frames[0]
		Expect(f.Key).To(Equal(hotFire.Key))
		Expect(f.DispatchKey).To(Equal("dk-1"))
		Expect(f.Seq).To(BeNumerically(">", 0))
		Expect(f.Actions).To(Equal(acts))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		rec := &Recorder[ranger.Key, ranger.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		Expect(svc.Dispatch(ctx, hotFire.Key, "dk", []ranger.Action{
			set([]string{"name"}, "Cold flow"),
			set([]string{"labels"}, []any{}),
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx, hotFire.Key).Name).To(Equal("Hot fire"))
		Expect(rec.Snapshot()).To(BeEmpty())
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

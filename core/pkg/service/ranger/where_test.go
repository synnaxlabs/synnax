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
	"encoding/json/v2"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/testutil/vectors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", func() {
	v := MustSucceed(vectors.Read[ranger.Range, ranger.Where]("ranges"))
	keys := lo.MapEntries(v.Records, func(
		alias string,
		r ranger.Range,
	) (ranger.Key, string) {
		return r.Key, alias
	})

	BeforeEach(func(ctx SpecContext) {
		openTx()
		for _, k := range lo.Uniq(lo.FlatMap(
			lo.Values(v.Records),
			func(r ranger.Range, _ int) []label.Key { return r.Labels },
		)) {
			l := label.Label{Key: k, Name: k.String()}
			Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		}
		for _, r := range v.Records {
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, w ranger.Where, expected []string) {
			f := MustSucceed(ranger.MatchWhere(w))
			var res []ranger.Range
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(lo.Map(res, func(r ranger.Range, _ int) string {
				return keys[r.Key]
			})).To(ConsistOf(expected))
		},
		lo.Map(v.Cases, func(c vectors.Case[ranger.Where], _ int) TableEntry {
			return Entry(c.Name, c.Where, c.Expected)
		}),
	)

	Describe("Time", func() {
		var running, planned, done ranger.Range
		BeforeEach(func(ctx SpecContext) {
			now := telem.Now()
			running = ranger.Range{
				Name:      "Running",
				TimeRange: now.Add(-telem.Hour).Range(telem.TimeStampMax),
			}
			planned = ranger.Range{
				Name:      "Planned",
				TimeRange: now.Add(telem.Hour).SpanRange(telem.Hour),
			}
			done = ranger.Range{
				Name:      "Done",
				TimeRange: now.Add(-3 * telem.Hour).SpanRange(telem.Hour),
			}
			for _, r := range []*ranger.Range{&running, &planned, &done} {
				Expect(svc.NewWriter(tx).Create(ctx, r)).To(Succeed())
			}
		})

		match := func(ctx SpecContext, w ranger.Where) []string {
			GinkgoHelper()
			var res []ranger.Range
			Expect(svc.NewRetrieve().
				Where(MustSucceed(ranger.MatchWhere(w))).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			return lo.FilterMap(res, func(r ranger.Range, _ int) (string, bool) {
				return r.Name, keys[r.Key] == ""
			})
		}

		DescribeTable(
			"Should match the ranges in the stages when the query opens",
			func(ctx SpecContext, stages []ranger.Stage, expected []string) {
				Expect(match(ctx, ranger.Where{Stages: stages})).
					To(ConsistOf(expected))
			},
			Entry("in progress", []ranger.Stage{ranger.StageInProgress},
				[]string{"Running"}),
			Entry("to do", []ranger.Stage{ranger.StageToDo}, []string{"Planned"}),
			Entry("completed", []ranger.Stage{ranger.StageCompleted}, []string{"Done"}),
			Entry(
				"two stages",
				[]ranger.Stage{ranger.StageToDo, ranger.StageInProgress},
				[]string{"Planned", "Running"},
			),
			Entry("no stages", []ranger.Stage{}, []string{}),
		)

		It(
			"Should match the ranges that overlap the recent span",
			func(ctx SpecContext) {
				Expect(match(ctx, ranger.Where{Recent: new(telem.Minute)})).
					To(ConsistOf("Running"))
				Expect(match(ctx, ranger.Where{Recent: new(4 * telem.Hour)})).
					To(ConsistOf("Running", "Done"))
			},
		)

		It("Should match the ranges that a negative recent span reaches", func(
			ctx SpecContext,
		) {
			Expect(match(ctx, ranger.Where{Recent: new(-2 * telem.Hour)})).
				To(ConsistOf("Running", "Planned"))
			Expect(match(ctx, ranger.Where{Recent: new(-30 * telem.Minute)})).
				To(ConsistOf("Running"))
		})

		It("Should match all past ranges for the maximum recent span", func(
			ctx SpecContext,
		) {
			span := telem.TimeSpan(math.MaxInt64)
			Expect(match(ctx, ranger.Where{Recent: &span})).
				To(ConsistOf("Running", "Done"))
		})

		It("Should match all future ranges for the minimum recent span", func(
			ctx SpecContext,
		) {
			span := telem.TimeSpan(math.MinInt64)
			Expect(match(ctx, ranger.Where{Recent: &span})).
				To(ConsistOf("Running", "Planned"))
		})

		It("Should match the ranges that overlap a time range", func(ctx SpecContext) {
			tr := done.TimeRange.End.Add(-telem.Minute).SpanRange(2 * telem.Hour)
			Expect(match(ctx, ranger.Where{Overlaps: &tr})).
				To(ConsistOf("Done", "Running"))
			Expect(match(ctx, ranger.Where{Overlaps: new(tr.Swap())})).
				To(ConsistOf("Done", "Running"))
		})

		It("Should see a range moved in the same transaction", func(ctx SpecContext) {
			Expect(svc.NewWriter(tx).Apply(ctx, planned.Key, ranger.NewSetAction(
				ranger.SetPayload{
					Path:  []string{"time_range"},
					Value: telem.Now().Add(-telem.Minute).Range(telem.TimeStampMax),
				},
			))).To(Succeed())
			Expect(match(ctx, ranger.Where{
				Stages: []ranger.Stage{ranger.StageInProgress},
			})).To(ConsistOf("Running", "Planned"))
		})

		It("Should return a path error for an invalid stage", func() {
			Expect(ranger.MatchWhere(ranger.Where{
				Stages: []ranger.Stage{ranger.StageToDo, "paused"},
			})).Error().To(MatchError(
				`stages.1: invalid stages "paused": validation error`,
			))
		})

		It("Should panic on an invalid stage from Go", func() {
			Expect(func() { ranger.MatchStages("paused")(svc.NewRetrieve()) }).
				To(PanicWith(`ranger: invalid stage "paused"`))
		})
	})

	It("Should return a path error for an invalid regex", func() {
		var w ranger.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(ranger.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError("or.1.name.regex: error parsing regexp: missing closing ): "+
				"`(a`: validation error"),
		))
	})
})

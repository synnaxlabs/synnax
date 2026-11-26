// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Stop", func() {
	var (
		db      *gorp.DB
		stopped [][]resource.ID
		stopErr error
		svc     *impact.Service
	)
	label, rng := resource.TypeLabel, resource.TypeRange
	labelOf := func(k string) resource.ID { return resource.ID{Type: label, Key: k} }
	rangeOf := func(k string) resource.ID { return resource.ID{Type: rng, Key: k} }
	// deleter returns a Deleter whose records each restrict holds one of the same key
	// and type holder, or none for an empty holder.
	deleter := func(holder resource.Type) impact.Deleter {
		return impact.Deleter{
			Plan: func(
				_ context.Context,
				_ gorp.Snapshot,
				ids []resource.ID,
			) ([]resource.ID, gorp.DeleteEffects, error) {
				var effects gorp.DeleteEffects
				if holder != "" {
					for _, id := range ids {
						effects.Restricts = append(effects.Restricts, gorp.DeleteEffect{
							Holder: gorp.Target{Table: string(holder), Key: id.Key},
							Target: gorp.Target{Table: string(id.Type), Key: id.Key},
						})
					}
				}
				return ids, effects, nil
			},
			Delete: func(
				context.Context,
				gorp.Tx,
				[]resource.ID,
				gorp.DeleteEffects,
			) error {
				return nil
			},
		}
	}
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		stopped, stopErr = nil, nil
		ranges := deleter("")
		ranges.Stop = func(_ context.Context, removed []resource.ID) error {
			stopped = append(stopped, removed)
			return stopErr
		}
		svc = MustSucceed(impact.OpenService(impact.ServiceConfig{
			Deleters: map[resource.Type]impact.Deleter{
				label: deleter(rng),
				rng:   ranges,
			},
			RecordOf: func(t gorp.Target) resource.ID {
				return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
			},
		}))
	})

	Describe("Stopping", func() {
		It("Should list the records that the delete removes of a type that stops", func(
			ctx SpecContext,
		) {
			Expect(
				svc.Stopping(
					ctx,
					db,
					[]resource.ID{rangeOf("1")},
					impact.ApplyOptions{},
				),
			).
				To(Equal([]resource.ID{rangeOf("1")}))
		})

		It("Should list a record that a delete pick adds to the delete", func(
			ctx SpecContext,
		) {
			Expect(
				svc.Stopping(ctx, db, []resource.ID{labelOf("1")}, impact.ApplyOptions{
					Picks: map[resource.ID]impact.Choice{
						rangeOf("1"): impact.ChoiceDelete,
					},
				}),
			).To(Equal([]resource.ID{rangeOf("1")}))
		})

		It(
			"Should list nothing when an entry cancels the delete",
			func(ctx SpecContext) {
				Expect(
					svc.Stopping(
						ctx,
						db,
						[]resource.ID{labelOf("1")},
						impact.ApplyOptions{},
					),
				).
					To(BeEmpty())
			},
		)

		It("Should list nothing for the token of another plan", func(ctx SpecContext) {
			Expect(
				svc.Stopping(ctx, db, []resource.ID{rangeOf("1")}, impact.ApplyOptions{
					Token: "0000000000000000",
				}),
			).To(BeEmpty())
		})

		It("Should list nothing of a type that does not stop", func(ctx SpecContext) {
			Expect(
				svc.Stopping(ctx, db, []resource.ID{labelOf("1")}, impact.ApplyOptions{
					Picks: map[resource.ID]impact.Choice{
						rangeOf("1"): impact.ChoiceCancel,
					},
				}),
			).To(BeEmpty())
		})

		It("Should refuse a pick that is not a choice", func(ctx SpecContext) {
			Expect(
				svc.Stopping(ctx, db, []resource.ID{labelOf("1")}, impact.ApplyOptions{
					Picks: map[resource.ID]impact.Choice{rangeOf("1"): "keep"},
				}),
			).Error().
				To(MatchError(ContainSubstring("keep is not a choice")))
			Expect(stopped).To(BeEmpty())
		})
	})

	Describe("Stop", func() {
		It("Should run the Stop of a type on its records", func(ctx SpecContext) {
			ids := []resource.ID{rangeOf("1"), rangeOf("2")}
			Expect(svc.Stop(ctx, ids)).To(Succeed())
			Expect(stopped).To(Equal([][]resource.ID{ids}))
		})

		It("Should return the error of a Stop", func(ctx SpecContext) {
			stopErr = errors.New("driver unreachable")
			Expect(svc.Stop(ctx, []resource.ID{rangeOf("1")})).
				To(MatchError("driver unreachable"))
		})
	})
})

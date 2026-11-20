// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// ownedStatuses returns the statuses that the calculation with key owns.
func ownedStatuses(ctx context.Context, key calculation.Key) []calculation.Status {
	GinkgoHelper()
	var statuses []calculation.Status
	Expect(statusSvc.NewRetrieve[struct{}]().
		Where(status.MatchOwners[struct{}](calculation.ResourceID(key))).
		Entries(&statuses).
		Exec(ctx, db)).To(Succeed())
	return statuses
}

// statusesNamed returns the statuses with the given name.
func statusesNamed(ctx context.Context, name string) []calculation.Status {
	GinkgoHelper()
	var all []calculation.Status
	Expect(statusSvc.NewRetrieve[struct{}]().Entries(&all).Exec(ctx, db)).
		To(Succeed())
	var named []calculation.Status
	for _, s := range all {
		if s.Name == name {
			named = append(named, s)
		}
	}
	return named
}

var _ = Describe("Graph breaker", func() {
	It(
		"Should delete the status of a calculation when its record is deleted",
		func(ctx SpecContext) {
			openGraph(ctx)
			calc := createBrokenCalc(ctx, "brk_rec_del", "brk_rec_del_dep")
			expectStatus(ctx, calc.Key())
			c := MustSucceed(creator.CalculationOf(ctx, calc.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return creator.Calculation.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
			Eventually(func() []calculation.Status {
				return ownedStatuses(ctx, c.Key)
			}).Should(BeEmpty())
			Consistently(func() []calculation.Status {
				return statusesNamed(ctx, "brk_rec_del")
			}, 300*time.Millisecond).Should(BeEmpty())
		},
	)

	It(
		"Should delete the status of a calculation when its output is deleted",
		func(ctx SpecContext) {
			openGraph(ctx)
			calc := createBrokenCalc(ctx, "brk_out_del", "brk_out_del_dep")
			expectStatus(ctx, calc.Key())
			c := MustSucceed(creator.CalculationOf(ctx, calc.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, calc.Key(), false)
			})).To(Succeed())
			Eventually(func() []calculation.Status {
				return ownedStatuses(ctx, c.Key)
			}).Should(BeEmpty())
			Consistently(func() []calculation.Status {
				return statusesNamed(ctx, "brk_out_del")
			}, 300*time.Millisecond).Should(BeEmpty())
		},
	)

	It(
		"Should clear the status of a broken calculation that the calculation API "+
			"edits to a valid expression",
		func(ctx SpecContext) {
			openGraph(ctx)
			createDep(ctx, "brk_api_fix_other")
			calc := createBrokenCalc(ctx, "brk_api_fix", "brk_api_fix_dep")
			expectStatus(ctx, calc.Key())
			c := MustSucceed(creator.CalculationOf(ctx, calc.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return creator.Calculation.NewWriter(tx).Apply(
					ctx,
					c.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return brk_api_fix_other + 1",
					}),
				)
			})).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())
		},
	)

	It(
		"Should give a dependent the new type when the record of its input is "+
			"replaced under another key",
		func(ctx SpecContext) {
			openGraph(ctx)
			createDep(ctx, "brk_repl_base")
			floatBase := channel.Channel{
				Name:     "brk_repl_fbase",
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &floatBase)
			})).To(Succeed())
			inner := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "brk_repl_inner",
				Expression: "return brk_repl_base * 2",
			}))
			outer := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "brk_repl_outer",
				Expression: "return brk_repl_inner + 1",
			}))
			Expect(retrieveChannelDataType(ctx, outer.Key())).To(Equal(telem.Int64T))
			old := MustSucceed(creator.CalculationOf(ctx, inner.Key()))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := creator.Calculation.NewWriter(tx)
				if err := w.Delete(ctx, old.Key); err != nil {
					return err
				}
				return w.Create(ctx, &calculation.Calculation{
					Output:     inner.Key(),
					Expression: "return brk_repl_fbase * 2",
				})
			})).To(Succeed())
			Expect(retrieveChannelDataType(ctx, inner.Key())).To(Equal(telem.Float64T))
			Eventually(func() telem.DataType {
				return retrieveChannelDataType(ctx, outer.Key())
			}).Should(Equal(telem.Float64T))
		},
	)
})

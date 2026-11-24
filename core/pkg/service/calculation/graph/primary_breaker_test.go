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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Primary breaker", func() {
	virtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	// upgrade leaves the calculation that writes output with no primary and no reads,
	// as the v0.62 upgrade stores it.
	upgrade := func(ctx SpecContext, output channel.Key) {
		GinkgoHelper()
		c := MustSucceed(creator.CalculationOf(ctx, output))
		Expect(gorp.NewUpdate[calculation.Key, calculation.Calculation]().
			Where(gorp.MatchKeys[calculation.Key, calculation.Calculation](c.Key)).
			Change(func(
				_ gorp.Context,
				c calculation.Calculation,
			) calculation.Calculation {
				c.Primary, c.Reads = nil, nil
				return c
			}).
			Exec(ctx, db)).To(Succeed())
	}
	rename := func(ctx SpecContext, key channel.Key, name string) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Rename(ctx, key, name, false)
		})).To(Succeed())
	}
	calculationOf := func(
		ctx context.Context,
		output channel.Key,
	) calculation.Calculation {
		GinkgoHelper()
		return MustSucceed(creator.CalculationOf(ctx, output))
	}

	It("Should announce the first save after an upgrade that drops the primary "+
		"picked at start", func(ctx SpecContext) {
		a := virtual(ctx, "brk15b_up_a")
		b := virtual(ctx, "brk15b_up_b")
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "brk15b_up_out",
			Expression: "return brk15b_up_a + brk15b_up_b",
		}))
		upgrade(ctx, out.Key())
		g := openGraph(ctx)
		Expect(calculationOf(ctx, out.Key()).Primary).To(HaveValue(Equal(a.Key())))
		handled := waitForBatch(g, out.Name)
		Expect(creator.Apply(
			ctx,
			out.Key(),
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"expression"},
				Value: "return brk15b_up_b",
			}),
		)).To(Succeed())
		Expect(calculationOf(ctx, out.Key()).Primary).To(HaveValue(Equal(b.Key())))
		// The graph commits its statuses before it notifies the batch.
		Eventually(handled).Should(BeClosed())
		var statuses []calculation.Status
		Expect(statusSvc.NewRetrieve[struct{}]().
			Entries(&statuses).
			Exec(ctx, db)).To(Succeed())
		Expect(statuses).To(ContainElement(And(
			HaveField("Name", out.Name),
			HaveField("Message", "Changed the primary input"),
		)))
	})

	It("Should keep the stored primary among the reads that the start pass "+
		"stores", func(ctx SpecContext) {
		a := virtual(ctx, "brk15b_rn_a")
		b := virtual(ctx, "brk15b_rn_b")
		c := virtual(ctx, "brk15b_rn_c")
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "brk15b_rn_out",
			Expression: "return brk15b_rn_a + brk15b_rn_b",
		}))
		Expect(calculationOf(ctx, out.Key()).Primary).To(HaveValue(Equal(a.Key())))
		// The text stays, but brk15b_rn_a now names another channel.
		rename(ctx, a.Key(), "brk15b_rn_x")
		rename(ctx, c.Key(), "brk15b_rn_a")
		openGraph(ctx)
		stored := calculationOf(ctx, out.Key())
		Expect(stored.Reads).To(ConsistOf(b.Key(), c.Key()))
		Expect(stored.Primary).ToNot(BeNil())
		Expect(stored.Reads).To(ContainElement(*stored.Primary))
	})
})

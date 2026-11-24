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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Primary check", func() {
	const changed = "Changed the primary input"
	virtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	announced := func(ctx SpecContext, name string) []calculation.Status {
		GinkgoHelper()
		var out []calculation.Status
		for _, s := range statusesNamed(ctx, name) {
			if s.Message == changed {
				out = append(out, s)
			}
		}
		return out
	}
	setExpression := func(ctx SpecContext, output channel.Key, expression string) {
		GinkgoHelper()
		Expect(creator.Apply(
			ctx,
			output,
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"expression"},
				Value: expression,
			}),
		)).To(Succeed())
	}

	It("Should keep one primary status for a calculation whose primary moves twice",
		func(ctx SpecContext) {
			virtual(ctx, "prc_twice_a")
			virtual(ctx, "prc_twice_b")
			virtual(ctx, "prc_twice_c")
			out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "prc_twice_out",
				Expression: "return prc_twice_a + prc_twice_b + prc_twice_c",
			}))
			openGraph(ctx)
			setExpression(ctx, out.Key(), "return prc_twice_b + prc_twice_c")
			Eventually(func() []calculation.Status {
				return announced(ctx, out.Name)
			}).Should(HaveLen(1))
			setExpression(ctx, out.Key(), "return prc_twice_c")
			Eventually(func() []calculation.Status {
				return announced(ctx, out.Name)
			}).Should(ContainElement(HaveField("Description", ContainSubstring(
				"no longer reads prc_twice_b",
			))))
			Expect(announced(ctx, out.Name)).To(HaveLen(1))
		})

	It("Should delete the primary status with its calculation", func(ctx SpecContext) {
		virtual(ctx, "prc_del_a")
		virtual(ctx, "prc_del_b")
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "prc_del_out",
			Expression: "return prc_del_a + prc_del_b",
		}))
		openGraph(ctx)
		setExpression(ctx, out.Key(), "return prc_del_b")
		Eventually(func() []calculation.Status {
			return announced(ctx, out.Name)
		}).Should(HaveLen(1))
		c := MustSucceed(creator.CalculationOf(ctx, out.Key()))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return creator.Calculation.NewWriter(tx).Delete(ctx, c.Key)
		})).To(Succeed())
		Eventually(func() []calculation.Status {
			return announced(ctx, out.Name)
		}, 2*time.Second).Should(BeEmpty())
	})

	It("Should announce, with its true text, a primary that the save sets while its "+
		"expression drops the held one", func(ctx SpecContext) {
		// The graph sees only the stored record, so it announces a set primary as it
		// does a pick.
		virtual(ctx, "prc_set_a")
		virtual(ctx, "prc_set_b")
		chosen := virtual(ctx, "prc_set_c")
		out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       "prc_set_out",
			Expression: "return prc_set_a + prc_set_b + prc_set_c",
		}))
		openGraph(ctx)
		Expect(creator.Apply(
			ctx,
			out.Key(),
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"expression"},
				Value: "return prc_set_b + prc_set_c",
			}),
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"primary"},
				Value: chosen.Key(),
			}),
		)).To(Succeed())
		Expect(MustSucceed(creator.CalculationOf(ctx, out.Key())).Primary).
			To(HaveValue(Equal(chosen.Key())))
		Eventually(func() []calculation.Status {
			return announced(ctx, out.Name)
		}).Should(ConsistOf(HaveField("Description",
			"The expression no longer reads prc_set_a, so the outputs now use the "+
				"timestamps of prc_set_c.",
		)))
	})
	It("Should delete the primary status when the calculation's output is deleted",
		func(ctx SpecContext) {
			virtual(ctx, "prc_out_del_a")
			virtual(ctx, "prc_out_del_b")
			out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "prc_out_del_out",
				Expression: "return prc_out_del_a + prc_out_del_b",
			}))
			openGraph(ctx)
			setExpression(ctx, out.Key(), "return prc_out_del_b")
			Eventually(func() []calculation.Status {
				return announced(ctx, out.Name)
			}).Should(HaveLen(1))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, out.Key(), false)
			})).To(Succeed())
			Eventually(func() []calculation.Status {
				return announced(ctx, out.Name)
			}, 2*time.Second).Should(BeEmpty())
		})

	It("Should rename the primary status with the calculation's output",
		func(ctx SpecContext) {
			virtual(ctx, "prc_ren_a")
			virtual(ctx, "prc_ren_b")
			out := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "prc_ren_out",
				Expression: "return prc_ren_a + prc_ren_b",
			}))
			openGraph(ctx)
			setExpression(ctx, out.Key(), "return prc_ren_b")
			Eventually(func() []calculation.Status {
				return announced(ctx, out.Name)
			}).Should(HaveLen(1))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).
					Rename(ctx, out.Key(), "prc_ren_out_2", false)
			})).To(Succeed())
			Eventually(func() []calculation.Status {
				return announced(ctx, "prc_ren_out_2")
			}, 2*time.Second).Should(HaveLen(1))
			Expect(announced(ctx, out.Name)).To(BeEmpty())
		})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation access checks", func() {
	retrieves := []access.Action{access.ActionRetrieve}
	createAsCore := func(
		ctx SpecContext,
		c calculation.Calculation,
	) calculation.Calculation {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Calculation.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}

	It("Should leave out a calculation whose primary the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		out, primary := createVirtual(ctx), createVirtual(ctx)
		createAsCore(ctx, calculation.Calculation{
			Output:     out.Key(),
			Expression: "return " + primary.Name,
			Primary:    new(primary.Key()),
		})
		grant(ctx, retrieves, u.ResourceID(), anyCalculation)
		grant(ctx, retrieves, u.ResourceID(), channel.ResourceID(out.Key()))
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, u), apicalculation.RetrieveRequest{}),
		)
		Expect(res.Calculations).To(BeEmpty())
	})

	It("Should leave out a calculation whose reset the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		out, reset := createVirtual(ctx), createVirtual(ctx)
		createAsCore(ctx, calculation.Calculation{
			Output:     out.Key(),
			Expression: "return 2",
			Operations: []calculation.Operation{{Variant: calculation.MaxOperation{
				Window: calculation.Window{Reset: new(reset.Key())},
			}}},
		})
		grant(ctx, retrieves, u.ResourceID(), anyCalculation)
		grant(ctx, retrieves, u.ResourceID(), channel.ResourceID(out.Key()))
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, u), apicalculation.RetrieveRequest{}),
		)
		Expect(res.Calculations).To(BeEmpty())
	})

	It("Should leave out a calculation whose read the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		out, read := createVirtual(ctx), createVirtual(ctx)
		createAsCore(ctx, calculation.Calculation{
			Output:     out.Key(),
			Expression: "return 2",
			Reads:      []channel.Key{read.Key()},
		})
		grant(ctx, retrieves, u.ResourceID(), anyCalculation)
		grant(ctx, retrieves, u.ResourceID(), channel.ResourceID(out.Key()))
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, u), apicalculation.RetrieveRequest{}),
		)
		Expect(res.Calculations).To(BeEmpty())
	})

	It("Should refuse a create without update on the output", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation)
		grant(ctx, retrieves, u.ResourceID(), anyChannel)
		out := createVirtual(ctx)
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []calculation.Calculation{
				{Output: out.Key(), Expression: "return 2"},
			},
		})).Error().To(MatchError(access.ErrDenied))
	})

	It(
		"Should refuse a set of the primary to a channel the subject cannot retrieve",
		func(
			ctx SpecContext,
		) {
			u := createUser(ctx)
			out, hidden := createVirtual(ctx), createVirtual(ctx)
			c := createAsCore(ctx, calculation.Calculation{
				Output:     out.Key(),
				Expression: "return " + hidden.Name,
			})
			grant(ctx, access.AllActions, u.ResourceID(), anyCalculation)
			grant(ctx, access.AllActions, u.ResourceID(), channel.ResourceID(out.Key()))
			Expect(dispatch(AuthedCtx(ctx, u), apicalculation.DispatchRequest{
				Key: c.Key,
				Actions: []calculation.Action{
					calculation.NewSetAction(
						calculation.SetPayload{
							Path:  []string{"primary"},
							Value: hidden.Key(),
						},
					),
				},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)

	// Question: bind.Read denies a named record the subject cannot retrieve. A
	// calculation named by key whose output is hidden reads as absent instead.
	It("Should deny a retrieve by key of a calculation whose output is hidden", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		c := createAsCore(ctx, calculation.Calculation{
			Output:     createVirtual(ctx).Key(),
			Expression: "return 2",
		})
		grant(ctx, retrieves, u.ResourceID(), anyCalculation)
		Expect(retrieve(AuthedCtx(ctx, u), apicalculation.RetrieveRequest{
			Where: &calculation.Where{Key: &filter.Set[calculation.Key]{Eq: &c.Key}},
		})).Error().To(MatchError(access.ErrDenied))
	})
})

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
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Calculation access breaker", func() {
	It(
		"Should refuse a create whose expression reads a channel the subject cannot "+
			"retrieve",
		func(ctx SpecContext) {
			u := createUser(ctx)
			hidden := channel.Channel{
				Name:     "calc_hidden_wide",
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Channel.NewWriter(tx).Create(ctx, &hidden)
			})).To(Succeed())
			out := createVirtual(ctx)
			grant(ctx, access.AllActions, u.ResourceID(), anyCalculation)
			grant(ctx, access.AllActions, u.ResourceID(), channel.ResourceID(out.Key()))
			_, err := create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
				Calculations: []calculation.Calculation{
					{Output: out.Key(), Expression: "return " + hidden.Name},
				},
			})
			var stored channel.Channel
			Expect(svc.Channel.NewRetrieve().
				Where(channel.MatchKeys(out.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(err).To(
				MatchError(access.ErrDenied),
				"the output now has the data type %s of the hidden channel",
				stored.DataType,
			)
		},
	)

	It(
		"Should refuse an edit whose expression reads a channel the subject cannot "+
			"retrieve",
		func(ctx SpecContext) {
			u := createUser(ctx)
			out, visible, hidden := createVirtual(
				ctx,
			), createVirtual(
				ctx,
			), createVirtual(
				ctx,
			)
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return " + visible.Name,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Calculation.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			grant(ctx, access.AllActions, u.ResourceID(), anyCalculation)
			grant(
				ctx,
				access.AllActions,
				u.ResourceID(),
				channel.ResourceID(out.Key()),
				channel.ResourceID(visible.Key()),
			)
			Expect(dispatch(AuthedCtx(ctx, u), apicalculation.DispatchRequest{
				Key: c.Key,
				Actions: []calculation.Action{calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return " + hidden.Name,
					},
				)},
			})).Error().To(MatchError(access.ErrDenied))
		},
	)
})

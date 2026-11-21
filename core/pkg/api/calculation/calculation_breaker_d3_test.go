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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// readFramer lets u read the samples of channels.
func readFramer(ctx SpecContext, u user.User, channels ...channel.Channel) {
	GinkgoHelper()
	grant(
		ctx,
		[]access.Action{access.ActionRetrieve},
		u.ResourceID(),
		framer.ResourceIDs(channel.KeysFromChannels(channels))...,
	)
}

var _ = Describe("Calculation breaker d3", func() {
	// A client settles its own echo against the actions it sent and the actions the
	// dispatch response says the Core added, as the channel, schematic, and task
	// dispatches return them.
	It("Should return each action that the Core adds to the echo of a dispatch", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		first, second := createVirtual(ctx), createVirtual(ctx)
		readFramer(ctx, u, first, second)
		out := createVirtual(ctx)
		c := calculation.Calculation{
			Output:     out.Key(),
			Expression: "return " + first.Name,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Calculation.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		rec := &Recorder[calculation.Key, calculation.Action]{}
		DeferCleanup(svc.Calculation.Actions().OnChange(rec.Record))
		sent := []calculation.Action{calculation.NewSetAction(calculation.SetPayload{
			Path:  []string{"expression"},
			Value: "return " + second.Name,
		})}
		dispatchKey := uuid.New().String()
		res := MustSucceed(dispatch(AuthedCtx(ctx, u), apicalculation.DispatchRequest{
			Key:         c.Key,
			DispatchKey: dispatchKey,
			Actions:     sent,
		}))
		var echo actions.Scoped[calculation.Key, calculation.Action]
		Eventually(func(g Gomega) {
			for _, s := range rec.Snapshot() {
				if s.DispatchKey == dispatchKey {
					echo = s
				}
			}
			g.Expect(echo.Actions).ToNot(BeEmpty())
		}).Should(Succeed())
		// The response of the calculation dispatch holds no actions.
		Expect(res).To(Equal(struct{}{}))
		Expect(echo.Actions).To(Equal(sent))
	})

	It("Should name the field of a made output whose name is taken", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		taken := createVirtual(ctx)
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{
				{
					Calculation: calculation.Calculation{Expression: "return 2"},
					Name:        "made_" + uuid.New().String()[:8],
				},
				{
					Calculation: calculation.Calculation{Expression: "return 2"},
					Name:        taken.Name,
				},
			},
		})).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("calculations.1.name")),
		))
	})

	It("Should type a made output by the derivative of its expression", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		input := createVirtualOf(ctx, telem.Int32T)
		readFramer(ctx, u, input)
		res := MustSucceed(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{{
				Calculation: calculation.Calculation{
					Expression: "return " + input.Name,
					Operations: []calculation.Operation{
						{Variant: calculation.DerivativeOperation{}},
					},
				},
				Name: "made_" + uuid.New().String()[:8],
			}},
		}))
		var output channel.Channel
		Expect(svc.Channel.NewRetrieve().
			Where(channel.MatchKeys(res.Calculations[0].Output)).
			Entry(&output).
			Exec(ctx, db)).To(Succeed())
		Expect(output.DataType).To(Equal(telem.Float64T))
		Expect(res.Calculations[0].Reads).To(Equal([]channel.Key{input.Key()}))
	})

	It("Should leave no output when the record of a made output fails", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		input, other := createVirtual(ctx), createVirtual(ctx)
		readFramer(ctx, u, input, other)
		name := "made_" + uuid.New().String()[:8]
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{{
				Calculation: calculation.Calculation{
					Expression: "return " + input.Name,
					Primary:    new(other.Key()),
				},
				Name: name,
			}},
		})).Error().To(MatchError(ContainSubstring("calculations.0.primary")))
		Expect(svc.Channel.NewRetrieve().
			Where(channel.MatchNames(name, name+"_time")).
			Exists(ctx, db)).To(BeFalse())
		res := MustSucceed(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{
				{
					Calculation: calculation.Calculation{
						Expression: "return " + input.Name,
					},
					Name: name,
				},
			},
		}))
		Expect(res.Calculations).To(HaveLen(1))
	})
	It("Should refuse to make an output in a group the subject cannot update", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		g := group.Group{
			Name:  "made_" + uuid.New().String()[:8],
			Scope: resource.TypeChannel,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		name := "made_" + uuid.New().String()[:8]
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{{
				Calculation: calculation.Calculation{Expression: "return 2"},
				Name:        name,
				Group:       &g.Key,
			}},
		})).Error().To(MatchError(access.ErrDenied))
		Expect(svc.Channel.NewRetrieve().
			Where(channel.MatchNames(name)).
			Exists(ctx, db)).To(BeFalse())
	})
})

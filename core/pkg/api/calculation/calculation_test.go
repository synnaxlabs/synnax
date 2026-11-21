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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apicalculation "github.com/synnaxlabs/synnax/pkg/api/calculation"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	anyCalculation = resource.ID{Type: resource.TypeCalculation}
	anyChannel     = resource.ID{Type: resource.TypeChannel}
)

var _ = Describe("Calculation", func() {
	// createCalculation creates a calculation of out as the Core, with no subject.
	createCalculation := func(ctx SpecContext, out channel.Channel) calculation.Calculation {
		GinkgoHelper()
		c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Calculation.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}
	byKey := func(c calculation.Calculation) apicalculation.RetrieveRequest {
		return apicalculation.RetrieveRequest{
			Where: &calculation.Where{Key: &filter.Set[calculation.Key]{Eq: &c.Key}},
		}
	}

	It("Should create a calculation on a virtual channel", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		out := createVirtual(ctx)
		res := MustSucceed(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{
				{
					Calculation: calculation.Calculation{
						Output:     out.Key(),
						Expression: "return 2",
					},
				},
			},
		}))
		Expect(res.Calculations).To(HaveLen(1))
		Expect(res.Calculations[0].Key).ToNot(BeZero())
		got := MustSucceed(retrieve(AuthedCtx(ctx, u), byKey(res.Calculations[0])))
		Expect(got.Calculations).To(Equal(res.Calculations))
	})

	Describe("Made output", func() {
		var u user.User
		BeforeEach(func(ctx SpecContext) {
			u = createUser(ctx)
			grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		})
		createInput := func(
			ctx SpecContext,
			in apicalculation.Input,
		) (apicalculation.CreateResponse, error) {
			return create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
				Calculations: []apicalculation.Input{in},
			})
		}
		createGroup := func(ctx SpecContext) group.Group {
			GinkgoHelper()
			g := group.Group{
				Name:  "made_" + uuid.New().String()[:8],
				Scope: resource.TypeChannel,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Group.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			return g
		}
		retrieveChannel := func(ctx SpecContext, key channel.Key) channel.Channel {
			GinkgoHelper()
			var ch channel.Channel
			Expect(svc.Channel.NewRetrieve().
				Where(channel.MatchKeys(key)).
				Entry(&ch).
				Exec(ctx, db)).To(Succeed())
			return ch
		}

		It("Should make a virtual output with the name and group of the input", func(
			ctx SpecContext,
		) {
			input := createVirtual(ctx)
			grant(
				ctx,
				[]access.Action{access.ActionRetrieve},
				u.ResourceID(),
				framer.ResourceIDs(channel.Keys{
					input.Key(),
				})...,
			)
			g := createGroup(ctx)
			grant(
				ctx,
				[]access.Action{access.ActionUpdate},
				u.ResourceID(),
				group.ResourceID(g.Key),
			)
			name := "made_" + uuid.New().String()[:8]
			res := MustSucceed(createInput(ctx, apicalculation.Input{
				Calculation: calculation.Calculation{
					Expression: "return " + input.Name,
				},
				Name:  name,
				Group: &g.Key,
			}))
			Expect(res.Calculations).To(HaveLen(1))
			output := retrieveChannel(ctx, res.Calculations[0].Output)
			Expect(output.Name).To(Equal(name))
			Expect(output.Virtual).To(BeTrue())
			Expect(output.DataType).To(Equal(input.DataType))
			Expect(output.Group).To(Equal(&g.Key))
		})

		Describe("Index", func() {
			var output channel.Channel
			BeforeEach(func(ctx SpecContext) {
				res := MustSucceed(createInput(ctx, apicalculation.Input{
					Calculation: calculation.Calculation{Expression: "return 2"},
					Name:        "made_" + uuid.New().String()[:8],
				}))
				output = retrieveChannel(ctx, res.Calculations[0].Output)
				Expect(output.Index()).ToNot(BeZero())
			})

			It("Should delete the index of a made output with the output", func(
				ctx SpecContext,
			) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Channel.NewWriter(tx).Delete(ctx, output.Key(), false)
				})).To(Succeed())
				Expect(svc.Channel.NewRetrieve().
					Where(channel.MatchKeys(output.Index())).
					Exists(ctx, db)).To(BeFalse())
			})

			It("Should move the index of a made output with the output", func(
				ctx SpecContext,
			) {
				g := createGroup(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Channel.NewWriter(tx).SetGroup(ctx, &g.Key, output.Key())
				})).To(Succeed())
				Expect(retrieveChannel(ctx, output.Index()).Group).To(Equal(&g.Key))
			})
		})

		It("Should refuse an input that names its output and sets a name", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx)
			Expect(createInput(ctx, apicalculation.Input{
				Calculation: calculation.Calculation{
					Output:     out.Key(),
					Expression: "return 2",
				},
				Name: "made",
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("calculations.0.name")),
			))
		})

		It("Should refuse an input that names its output and sets a group", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx)
			Expect(createInput(ctx, apicalculation.Input{
				Calculation: calculation.Calculation{
					Output:     out.Key(),
					Expression: "return 2",
				},
				Group: new(group.Key(uuid.New())),
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("calculations.0.group")),
			))
		})

		It("Should refuse an input that names no output and no name", func(
			ctx SpecContext,
		) {
			Expect(createInput(ctx, apicalculation.Input{
				Calculation: calculation.Calculation{Expression: "return 2"},
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("calculations.0.name")),
			))
		})
	})

	It("Should refuse a create without create on the calculation", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, []access.Action{access.ActionRetrieve}, u.ResourceID(), anyChannel)
		out := createVirtual(ctx)
		Expect(create(AuthedCtx(ctx, u), apicalculation.CreateRequest{
			Calculations: []apicalculation.Input{
				{
					Calculation: calculation.Calculation{
						Output:     out.Key(),
						Expression: "return 2",
					},
				},
			},
		})).Error().To(MatchError(access.ErrDenied))
	})

	It("Should leave out a calculation whose output the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		readable, hidden := createVirtual(ctx), createVirtual(ctx)
		shown := createCalculation(ctx, readable)
		createCalculation(ctx, hidden)
		retrieves := []access.Action{access.ActionRetrieve}
		grant(ctx, retrieves, u.ResourceID(), anyCalculation)
		grant(ctx, retrieves, u.ResourceID(), channel.ResourceID(readable.Key()))
		res := MustSucceed(
			retrieve(AuthedCtx(ctx, u), apicalculation.RetrieveRequest{}),
		)
		Expect(res.Calculations).To(Equal([]calculation.Calculation{shown}))
	})

	It("Should set the expression of a calculation", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		c := createCalculation(ctx, createVirtual(ctx))
		Expect(dispatch(AuthedCtx(ctx, u), apicalculation.DispatchRequest{
			Key: c.Key,
			Actions: []calculation.Action{calculation.NewSetAction(
				calculation.SetPayload{Path: []string{"expression"}, Value: "return 4"},
			)},
		})).Error().To(Succeed())
		got := MustSucceed(retrieve(AuthedCtx(ctx, u), byKey(c)))
		Expect(got.Calculations[0].Expression).To(Equal("return 4"))
	})

	It("Should delete a calculation", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.AllActions, u.ResourceID(), anyCalculation, anyChannel)
		c := createCalculation(ctx, createVirtual(ctx))
		Expect(del(AuthedCtx(ctx, u), apicalculation.DeleteRequest{
			Keys: []calculation.Key{c.Key},
		})).Error().To(Succeed())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), byKey(c)))
		Expect(res.Calculations).To(BeEmpty())
	})
})

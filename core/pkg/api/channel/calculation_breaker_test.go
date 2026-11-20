// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation records breaker", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "breaker-" + uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeChannel},
			resource.ID{Type: resource.TypeCalculation},
			resource.ID{Type: resource.TypeFramer},
		)
	})
	recordsOf := func(ctx SpecContext, key channel.Key) []calculation.Calculation {
		GinkgoHelper()
		var out []calculation.Calculation
		Expect(layer.Calculation.NewRetrieve().
			Where(calculation.MatchOutputs(key)).
			Entries(&out).
			Exec(ctx, db)).To(Succeed())
		return out
	}
	// createRecord creates a calculation of out through the calculation service, as the
	// new calculation API does.
	createRecord := func(ctx SpecContext, out channel.Key) calculation.Calculation {
		GinkgoHelper()
		c := calculation.Calculation{Output: out, Expression: "return 2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Calculation.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}
	rename := func(ctx SpecContext, key channel.Key) error {
		_, err := dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
			Key: key,
			Actions: []channel.Action{channel.NewRenameAction(
				channel.RenamePayload{Name: UniqueChannelName()},
			)},
		})
		return err
	}

	It("Should keep a new-API calculation when the old API renames its output", func(
		ctx SpecContext,
	) {
		out := createChannel(ctx, telem.Float32T)
		c := createRecord(ctx, out.Key())
		Expect(rename(ctx, out.Key())).To(Succeed())
		Expect(recordsOf(ctx, out.Key())).To(ConsistOf(c))
	})

	It(
		"Should keep a new-API calculation when an old-API create finds its output",
		func(
			ctx SpecContext,
		) {
			out := createChannel(ctx, telem.Float32T)
			c := createRecord(ctx, out.Key())
			Expect(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
				Channels: []apichannel.Channel{{
					Name:     out.Name,
					DataType: telem.Float32T,
					Virtual:  true,
				}},
				RetrieveIfNameExists: true,
			})).Error().To(Succeed())
			Expect(recordsOf(ctx, out.Key())).To(ConsistOf(c))
		},
	)

	It(
		"Should keep a new-API expression edit when the old API renames the output",
		func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			ch := MustSucceed(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
				Channels: []apichannel.Channel{{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: fmt.Sprintf("return %s * 2", input.Name),
				}},
			})).Channels[0]
			edited := fmt.Sprintf("return %s * 5", input.Name)
			rec := recordsOf(ctx, ch.Key)[0]
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Calculation.NewWriter(tx).Apply(
					ctx,
					rec.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: edited,
					}),
				)
			})).To(Succeed())
			Expect(rename(ctx, ch.Key)).To(Succeed())
			Expect(recordsOf(ctx, ch.Key)[0].Expression).To(Equal(edited))
		},
	)

	It("Should keep a new-API delete when the old API renames the output", func(
		ctx SpecContext,
	) {
		input := createChannel(ctx, telem.Float32T)
		ch := MustSucceed(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", input.Name),
			}},
		})).Channels[0]
		rec := recordsOf(ctx, ch.Key)[0]
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Calculation.NewWriter(tx).Delete(ctx, rec.Key)
		})).To(Succeed())
		Expect(rename(ctx, ch.Key)).To(Succeed())
		Expect(recordsOf(ctx, ch.Key)).To(BeEmpty())
	})

	It(
		"Should let a create-and-retrieve subject find a calculated channel by name",
		func(
			ctx SpecContext,
		) {
			input := createChannel(ctx, telem.Float32T)
			req := apichannel.CreateRequest{
				Channels: []apichannel.Channel{{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: fmt.Sprintf("return %s * 2", input.Name),
				}},
				RetrieveIfNameExists: true,
			}
			first := MustSucceed(createChannels(AuthedCtx(ctx, u), req)).Channels[0]
			// The grants that v63_grant_calculations gives a v0.62 policy of create and
			// retrieve on channels.
			v := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "creator-" + uuid.New().String()},
			))
			grantOn(
				ctx,
				v.ResourceID(),
				[]access.Action{access.ActionCreate, access.ActionRetrieve},
				resource.ID{Type: resource.TypeChannel},
				resource.ID{Type: resource.TypeCalculation},
				resource.ID{Type: resource.TypeFramer},
			)
			found := MustSucceed(createChannels(AuthedCtx(ctx, v), req)).Channels[0]
			Expect(found.Key).To(Equal(first.Key))
		},
	)
})

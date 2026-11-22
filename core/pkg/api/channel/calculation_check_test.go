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

var _ = Describe("Calculation records checks", func() {
	It(
		"Should keep a calculation API calculation when the old API moves its output",
		func(
			ctx SpecContext,
		) {
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "records-check-" + uuid.New().String()},
			))
			grantOn(
				ctx,
				u.ResourceID(),
				access.AllActions,
				resource.ID{Type: resource.TypeChannel},
				resource.ID{Type: resource.TypeCalculation},
				resource.ID{Type: resource.TypeGroup},
			)
			g := createChannelGroup(ctx, "Calculation output move")
			out := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			c := calculation.Calculation{Expression: "return 2"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := channelSvc.NewWriter(tx).Create(ctx, &out); err != nil {
					return err
				}
				c.Output = out.Key()
				return layer.Calculation.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
				Key: out.Key(),
				Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
					Path:  []string{"group"},
					Value: g.Key.String(),
				})},
			})).Error().To(Succeed())
			Expect(layer.Calculation.NewRetrieve().
				Where(calculation.MatchKeys(c.Key)).
				Exists(ctx, db)).To(BeTrue())
		},
	)

	// v63_grant_calculations gives a v0.62 policy of channel create and retrieve the
	// same actions on calculations, and no update.
	It(
		"Should find a calculated channel by name for a subject with create and retrieve",
		func(ctx SpecContext) {
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "records-check-" + uuid.New().String()},
			))
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionCreate, access.ActionRetrieve},
				resource.ID{Type: resource.TypeChannel},
				resource.ID{Type: resource.TypeCalculation},
			)
			req := apichannel.CreateRequest{
				Channels: []apichannel.Channel{{
					Name:       UniqueChannelName(),
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: "return 2",
				}},
				RetrieveIfNameExists: true,
			}
			first := MustSucceed(createChannels(AuthedCtx(ctx, u), req)).Channels[0]
			again := MustSucceed(createChannels(AuthedCtx(ctx, u), req)).Channels[0]
			Expect(again.Key).To(Equal(first.Key))
		},
	)
})

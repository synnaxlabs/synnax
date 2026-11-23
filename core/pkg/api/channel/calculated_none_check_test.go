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
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculated line on a Core with no calculation", func() {
	It(
		"Should keep no channel for calculated and each channel for not calculated",
		func(
			ctx SpecContext,
		) {
			n := testutil.NewNode(ctx)
			l := MustOpen(svctestutil.OpenLayer(ctx, n))
			svc := MustSucceed(apichannel.NewService(apichannel.ServiceConfig{
				DB:          n.DB,
				RBAC:        l.RBAC,
				Channel:     l.Channel,
				Status:      l.Status,
				Calculation: l.Calculation,
			}))
			binder := MustSucceed(bind.NewService(bind.ServiceConfig{
				DB:   n.DB,
				RBAC: l.RBAC,
				Held: apiauth.HeldTables(),
			}))
			// A Core makes one metrics calculation, and a user can delete it.
			var calcs []calculation.Calculation
			Expect(
				l.Calculation.NewRetrieve().Entries(&calcs).Exec(ctx, n.DB),
			).To(Succeed())
			Expect(calcs).ToNot(BeEmpty())
			Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return l.Calculation.NewWriter(tx).Delete(
					ctx,
					lo.Map(calcs, func(c calculation.Calculation, _ int) calculation.Key {
						return c.Key
					})...,
				)
			})).To(Succeed())
			create := bind.Write(binder, svc.Create)
			retrieve := bind.Read(binder, svc.Retrieve)
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				n.DB,
				l.RBAC,
				l.User,
				user.User{Name: "none-" + uuid.New().String()},
			))
			MustSucceed(rbactestutil.Grant(
				ctx,
				n.DB,
				l.RBAC,
				l.User,
				u.Key,
				access.AllActions,
				resource.ID{Type: resource.TypeChannel},
				resource.ID{Type: resource.TypeCalculation},
			))
			created := MustSucceed(create(
				AuthedCtx(ctx, u),
				apichannel.CreateRequest{Channels: []apichannel.Channel{
					{
						Name:     UniqueChannelName(),
						DataType: telem.Float32T,
						Virtual:  true,
					},
					{
						Name:     UniqueChannelName(),
						DataType: telem.Float32T,
						Virtual:  true,
					},
				}},
			)).Channels
			keys := lo.Map(created, func(ch apichannel.Channel, _ int) channel.Key {
				return ch.Key
			})
			matches := func(w channel.Where) []channel.Key {
				GinkgoHelper()
				res := MustSucceed(retrieve(
					AuthedCtx(ctx, u),
					apichannel.RetrieveRequest{Where: &w},
				))
				return lo.Intersect(
					keys,
					lo.Map(
						res.Channels,
						func(ch apichannel.Channel, _ int) channel.Key {
							return ch.Key
						},
					),
				)
			}
			within := func(w channel.Where) channel.Where {
				return channel.Where{And: []channel.Where{
					w,
					{Key: &filter.Set[channel.Key]{In: keys}},
				}}
			}
			Expect(matches(channel.Where{Calculated: new(true)})).To(BeEmpty())
			Expect(matches(within(channel.Where{Calculated: new(true)}))).To(BeEmpty())
			Expect(matches(channel.Where{Calculated: new(false)})).To(ConsistOf(keys))
			Expect(matches(within(channel.Where{Calculated: new(false)}))).
				To(ConsistOf(keys))
			Expect(matches(within(channel.Where{
				Not: &channel.Where{Calculated: new(true)},
			}))).To(ConsistOf(keys))
			Expect(matches(within(channel.Where{Or: []channel.Where{
				{Virtual: new(false)},
				{Calculated: new(true)},
			}}))).To(BeEmpty())
		},
	)
})

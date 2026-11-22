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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Dispatch access check", func() {
	It("Should refuse a dispatch that reads an unreadable channel through a stored "+
		"calculation", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "dispatch-" + uuid.New().String()},
		))
		public := createChannel(ctx, telem.Float32T)
		secret := createChannel(ctx, telem.Float32T)
		creator := calctestutil.Creator{
			DB:          db,
			Channel:     channelSvc,
			Calculation: layer.Calculation,
		}
		stored := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Expression: fmt.Sprintf("return %s * 2", secret.Name),
		}))
		calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			DataType:   telem.Float32T,
			Expression: fmt.Sprintf("return %s * 2", public.Name),
		}))
		grantOn(
			ctx,
			u.ResourceID(),
			access.AllActions,
			resource.ID{Type: resource.TypeCalculation},
		)
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			channel.ResourceID(calc.Key()),
		)
		readable := channel.Keys{public.Key(), stored.Key()}
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			append(
				framer.ResourceIDs(readable),
				channel.ResourceID(public.Key()),
				channel.ResourceID(stored.Key()),
			)...,
		)
		edit := func(input string) error {
			_, err := dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
				Key: calc.Key(),
				Actions: []channel.Action{channel.NewSetAction(channel.SetPayload{
					Path:  []string{"expression"},
					Value: fmt.Sprintf("return %s * 3", input),
				})},
			})
			return err
		}
		Expect(edit(public.Name)).To(Succeed())
		Expect(edit(stored.Name)).To(MatchError(access.ErrDenied))
		Expect(expressionOf(ctx, calc.Key())).
			To(Equal(fmt.Sprintf("return %s * 3", public.Name)))
	})
})

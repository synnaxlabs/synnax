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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Named retrieves", Ordered, func() {
	var (
		reader               user.User
		readable, unreadable channel.Channel
	)
	BeforeAll(func(ctx SpecContext) {
		chs := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).CreateMany(ctx, &chs)
		})).To(Succeed())
		readable, unreadable = chs[0], chs[1]
		reader = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "named-reader-" + uuid.New().String()}))
		grantOn(
			ctx,
			reader.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			channel.ResourceID(readable.Key()),
		)
	})
	ask := func(ctx SpecContext, w channel.Where) ([]channel.Key, error) {
		res, err := retrieveChannels(
			AuthedCtx(ctx, reader),
			apichannel.RetrieveRequest{Where: &w},
		)
		if err != nil {
			return nil, err
		}
		return lo.Map(res.Channels, func(ch apichannel.Channel, _ int) channel.Key {
			return ch.Key
		}), nil
	}
	eq := func(name string) *filter.String { return &filter.String{Eq: new(name)} }

	It("Should deny a name that an unreadable channel holds", func(ctx SpecContext) {
		Expect(ask(ctx, channel.Where{Name: eq(unreadable.Name)})).Error().
			To(MatchError(access.ErrDenied))
	})

	It("Should deny a name that an unreadable channel holds with the virtual line",
		func(ctx SpecContext) {
			Expect(ask(ctx, channel.Where{
				Name:    eq(unreadable.Name),
				Virtual: new(true),
			})).Error().To(MatchError(access.ErrDenied))
		},
	)

	It("Should return a readable channel by name with the virtual line", func(
		ctx SpecContext,
	) {
		Expect(ask(ctx, channel.Where{Name: eq(readable.Name), Virtual: new(true)})).
			To(Equal([]channel.Key{readable.Key()}))
	})
})

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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Kept stored names", func() {
	var (
		u  user.User
		ch channel.Channel
		// decomposed is the name that the stored channel holds, not in NFC, as a v0.62
		// Core with name validation off stored it. The upgrade keeps it.
		decomposed string
	)
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			resource.ID{Type: resource.TypeChannel},
		)
		name := UniqueChannelName()
		ch = channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		decomposed = name + "_é"
		stored := ch
		stored.Name = decomposed
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, stored)).
			To(Succeed())
	})

	It("Should retrieve a kept stored name by the exact name that it holds", func(
		ctx SpecContext,
	) {
		where := channel.Where{Name: &filter.String{Eq: &decomposed}}
		res := MustSucceed(retrieveChannels(
			AuthedCtx(ctx, u),
			apichannel.RetrieveRequest{Where: &where},
		))
		Expect(res.Channels).To(HaveLen(1))
		Expect(res.Channels[0].Key).To(Equal(ch.Key()))
	})

	It("Should delete a kept stored name by the exact name that it holds", func(
		ctx SpecContext,
	) {
		MustSucceed(deleteChannels(
			AuthedCtx(ctx, u),
			apichannel.DeleteRequest{Names: []string{decomposed}},
		))
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Exists(ctx, db)).To(BeFalse())
	})
})

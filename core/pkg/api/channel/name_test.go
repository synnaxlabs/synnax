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

var _ = Describe("Names", func() {
	It("Should find a channel by the decomposed form of its name", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
		)
		decomposed := UniqueChannelName() + "_e\u0301"
		ch := channel.Channel{Name: decomposed, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		for _, where := range []channel.Where{
			{Name: &filter.String{Eq: &decomposed}},
			{Or: []channel.Where{{Name: &filter.String{In: []string{decomposed}}}}},
		} {
			res := MustSucceed(retrieveChannels(
				AuthedCtx(ctx, u),
				apichannel.RetrieveRequest{Where: &where},
			))
			Expect(res.Channels).To(HaveLen(1))
			Expect(res.Channels[0].Key).To(Equal(ch.Key()))
		}
	})
})

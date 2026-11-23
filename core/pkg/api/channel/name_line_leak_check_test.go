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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Name line leak check", func() {
	It("Should not tell a subject the data type of an unreadable channel", func(
		ctx SpecContext,
	) {
		owner := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "leak-owner-" + uuid.New().String()}))
		grantOn(ctx, owner.ResourceID(), access.AllActions,
			resource.ID{Type: resource.TypeChannel})
		outsider := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc,
			user.User{Name: "leak-outsider-" + uuid.New().String()}))
		created := MustSucceed(createChannels(
			AuthedCtx(ctx, owner),
			apichannel.CreateRequest{Channels: []apichannel.Channel{
				{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
				{Name: UniqueChannelName(), DataType: telem.Int64T, Virtual: true},
			}},
		)).Channels
		readable := MustSucceed(createChannels(
			AuthedCtx(ctx, owner),
			apichannel.CreateRequest{Channels: []apichannel.Channel{
				{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
			}},
		)).Channels[0]
		grantOn(ctx, outsider.ResourceID(), []access.Action{access.ActionRetrieve},
			channel.ResourceID(readable.Key))
		ask := func(name string) error {
			where := channel.Where{
				Name: &filter.String{Eq: new(name)},
				DataType: &filter.Set[telem.DataType]{
					In: []telem.DataType{telem.Float32T},
				},
			}
			_, err := retrieveChannels(
				AuthedCtx(ctx, outsider),
				apichannel.RetrieveRequest{Where: &where},
			)
			return err
		}
		floatErr, intErr := ask(created[0].Name), ask(created[1].Name)
		Expect(errors.Is(floatErr, access.ErrDenied)).
			To(Equal(errors.Is(intErr, access.ErrDenied)),
				"float32: %v, int64: %v", floatErr, intErr)
	})
})

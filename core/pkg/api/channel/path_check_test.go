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
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Path checks", func() {
	// D-214: a path through a folder that the subject cannot retrieve matches nothing,
	// so a lookup cannot probe whether a hidden folder's name exists.
	It("Should match nothing for a path through a folder the subject cannot retrieve",
		func(ctx SpecContext) {
			g := createChannelGroup(ctx, "hidden-"+uuid.New().String()[:8])
			ch := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &g.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			u := MustSucceed(rbactestutil.CreateUser(
				ctx,
				db,
				rbacSvc,
				userSvc,
				user.User{Name: "path-check-" + uuid.New().String()},
			))
			grantOn(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				channel.ResourceID(ch.Key()),
			)
			path := g.Name + "." + ch.Name
			res := MustSucceed(
				retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
					Where: &channel.Where{Path: &filter.Set[string]{Eq: &path}},
				}),
			)
			Expect(res.Channels).To(BeEmpty())
		},
	)
})

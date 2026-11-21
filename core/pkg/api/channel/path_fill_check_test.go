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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// D-214: a path shows only when the subject can retrieve every folder on it, so a
// visible folder under a hidden one shows no path, and a visible chain shows all of it.
var _ = Describe("Path fill checks", func() {
	It("Should check every folder on a path, not only the nearest", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		top := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		mid := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		hiddenTop := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		shownMid := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		var shownChain, underHidden channel.Channel
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := groupSvc.NewWriter(tx)
			for _, g := range []*group.Group{&top, &hiddenTop} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			mid.Parent = new(group.ResourceID(top.Key))
			shownMid.Parent = new(group.ResourceID(hiddenTop.Key))
			for _, g := range []*group.Group{&mid, &shownMid} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			virtual := func(g *group.Key) channel.Channel {
				return channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float32T,
					Virtual:  true,
					Group:    g,
				}
			}
			shownChain, underHidden = virtual(&mid.Key), virtual(&shownMid.Key)
			return channelSvc.NewWriter(tx).CreateMany(ctx, &[]channel.Channel{
				shownChain, underHidden,
			})
		})).To(Succeed())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			top.ResourceID(),
			mid.ResourceID(),
			shownMid.ResourceID(),
		)
		res := MustSucceed(
			retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
				Where: &channel.Where{Name: &filter.String{
					In: []string{shownChain.Name, underHidden.Name},
				}},
				IncludePath: true,
			}),
		)
		paths := make(map[string]*string, len(res.Channels))
		for _, ch := range res.Channels {
			paths[ch.Name] = ch.Path
		}
		Expect(paths).To(Equal(map[string]*string{
			shownChain.Name:  new(top.Name + "." + mid.Name + "." + shownChain.Name),
			underHidden.Name: nil,
		}))
	})
})

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

var _ = Describe("Paths", func() {
	var (
		u                user.User
		shown, hidden    group.Group
		atTop, inShown   channel.Channel
		inHidden         channel.Channel
		shownDir, hidDir string
	)
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		shownDir, hidDir = UniqueChannelName(), UniqueChannelName()
		shown = group.Group{Name: shownDir, Scope: resource.TypeChannel}
		hidden = group.Group{Name: hidDir, Scope: resource.TypeChannel}
		virtual := func(g *group.Key) channel.Channel {
			return channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    g,
			}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range []*group.Group{&shown, &hidden} {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			atTop, inShown, inHidden = virtual(nil), virtual(&shown.Key),
				virtual(&hidden.Key)
			return channelSvc.NewWriter(tx).CreateMany(ctx, &[]channel.Channel{
				atTop, inShown, inHidden,
			})
		})).To(Succeed())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			shown.ResourceID(),
		)
	})
	keyOf := func(ctx SpecContext, name string) channel.Key {
		GinkgoHelper()
		var ch channel.Channel
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Key()
	}
	retrieve := func(ctx SpecContext, req apichannel.RetrieveRequest) map[string]*string {
		GinkgoHelper()
		res := MustSucceed(retrieveChannels(AuthedCtx(ctx, u), req))
		paths := make(map[string]*string, len(res.Channels))
		for _, ch := range res.Channels {
			paths[ch.Name] = ch.Path
		}
		return paths
	}
	byKeys := func(ctx SpecContext) *channel.Where {
		return &channel.Where{Key: &filter.Set[channel.Key]{In: []channel.Key{
			keyOf(ctx, atTop.Name), keyOf(ctx, inShown.Name), keyOf(ctx, inHidden.Name),
		}}}
	}

	It("Should fill no path when the request does not ask for one", func(
		ctx SpecContext,
	) {
		Expect(retrieve(ctx, apichannel.RetrieveRequest{Where: byKeys(ctx)})).
			To(Equal(map[string]*string{
				atTop.Name:    nil,
				inShown.Name:  nil,
				inHidden.Name: nil,
			}))
	})

	It("Should fill each path, and none through a folder the subject cannot retrieve",
		func(ctx SpecContext) {
			Expect(retrieve(ctx, apichannel.RetrieveRequest{
				Where:       byKeys(ctx),
				IncludePath: true,
			})).To(Equal(map[string]*string{
				atTop.Name:    new(atTop.Name),
				inShown.Name:  new(shownDir + "." + inShown.Name),
				inHidden.Name: nil,
			}))
		},
	)
})

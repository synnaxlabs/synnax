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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
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

var _ = Describe("Strict retrieve breaker", func() {
	var (
		u             user.User
		shown, hidden group.Group
		name, solo    string
		inShown       channel.Channel
		inHidden      channel.Channel
		alone         channel.Channel
		extra         []channel.Key
	)
	stored := func(n string, key channel.LocalKey, g *group.Key) channel.Channel {
		return channel.Channel{
			Name:        n,
			Leaseholder: node.KeyFree,
			LocalKey:    key,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       g,
		}
	}
	// store writes chs past the writer, as a v0.62 Core stored them.
	store := func(ctx SpecContext, chs ...channel.Channel) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
			Set(ctx, chs...)).To(Succeed())
		extra = append(extra, channel.KeysFromChannels(chs)...)
	}
	BeforeEach(func(ctx SpecContext) {
		extra = nil
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		shown = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		hidden = group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := groupSvc.NewWriter(tx)
			for _, g := range []*group.Group{&shown, &hidden} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		name, solo = UniqueChannelName(), UniqueChannelName()
		inShown = stored(name, 1_000_901, &shown.Key)
		inHidden = stored(name, 1_000_902, &hidden.Key)
		alone = stored(solo, 1_000_903, &shown.Key)
		store(ctx, inShown, inHidden, alone)
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, extra...)).To(Succeed())
		})
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve},
			resource.ID{Type: resource.TypeChannel},
			shown.ResourceID(),
		)
	})
	retrieve := func(
		ctx SpecContext,
		where channel.Where,
	) (apichannel.RetrieveResponse, error) {
		return retrieveChannels(AuthedCtx(ctx, u), apichannel.RetrieveRequest{
			Where:  &where,
			Strict: true,
		})
	}
	named := func(res apichannel.RetrieveResponse, n string) int {
		count := 0
		for _, ch := range res.Channels {
			if ch.Name == n {
				count++
			}
		}
		return count
	}
	refused := ContainSubstring(
		"where: a strict retrieve takes only key, name, and path matches",
	)

	It("Should refuse a strict where that holds no match", func(ctx SpecContext) {
		res, err := retrieve(ctx, channel.Where{})
		Expect(err).To(MatchError(refused),
			"a strict empty where returned %d channels, %d of them named %s",
			len(res.Channels), named(res, name), name,
		)
	})

	It("Should refuse a strict or with an empty branch", func(ctx SpecContext) {
		res, err := retrieve(ctx, channel.Where{Or: []channel.Where{
			{Name: &filter.String{In: []string{solo}}},
			{},
		}})
		Expect(err).To(MatchError(refused),
			"a strict or with an empty branch returned %d channels, %d of them named %s",
			len(res.Channels), named(res, name), name,
		)
	})

	It("Should refuse a strict where with an empty and", func(ctx SpecContext) {
		res, err := retrieve(ctx, channel.Where{And: []channel.Where{}})
		Expect(err).To(MatchError(refused),
			"a strict where with an empty and returned %d channels, %d of them named %s",
			len(res.Channels), named(res, name), name,
		)
	})
})

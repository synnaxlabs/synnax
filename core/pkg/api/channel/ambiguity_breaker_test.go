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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// D-214: no hidden folder's name shows to a subject that cannot retrieve it. A store
// from before the sibling rule holds one name in a shown and a hidden folder.
var _ = Describe("Ambiguity breaker", func() {
	var (
		u              user.User
		shown, hidden  group.Group
		name           string
		inShown, inHid channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
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
		name = UniqueChannelName()
		stored := func(key channel.LocalKey, g *group.Key) channel.Channel {
			return channel.Channel{
				Name:        name,
				Leaseholder: node.KeyFree,
				LocalKey:    key,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			}
		}
		inShown, inHid = stored(1_000_101, &shown.Key), stored(1_000_102, &hidden.Key)
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
			Set(ctx, inShown, inHid)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, inShown.Key(), inHid.Key())).To(Succeed())
		})
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionRetrieve, access.ActionDelete},
			resource.ID{Type: resource.TypeChannel},
			shown.ResourceID(),
		)
	})

	It("Should not name a hidden folder when a delete by name is ambiguous", func(
		ctx SpecContext,
	) {
		_, err := deleteChannels(AuthedCtx(ctx, u), apichannel.DeleteRequest{
			Names: []string{name},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).ToNot(ContainSubstring(hidden.Name))
	})

	It("Should not name a hidden folder when a retrieve-if-exists create is "+
		"ambiguous", func(ctx SpecContext) {
		_, err := createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:     name,
				DataType: telem.Float32T,
				Virtual:  true,
			}},
			RetrieveIfNameExists: true,
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).ToNot(ContainSubstring(hidden.Name))
	})
})

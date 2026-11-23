// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel link folders access check", func() {
	var (
		folder group.Group
		inside channel.Key
		path   string
		u      user.User
	)
	grant := func(ctx SpecContext, action access.Action, id resource.ID) {
		GinkgoHelper()
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			u.Key,
			[]access.Action{action},
			id,
		))
	}
	BeforeEach(func(ctx SpecContext) {
		folder = group.Group{
			Name:  channeltestutil.UniqueChannelName(),
			Scope: resource.TypeChannel,
		}
		ch := channel.Channel{
			Name:     channeltestutil.UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := groupSvc.NewWriter(tx).Create(ctx, &folder); err != nil {
				return err
			}
			ch.Group = new(folder.Key)
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		inside = ch.Key()
		path = folder.Name + string(validate.PathSeparator) + ch.Name
		u = MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "importer-" + uuid.New().String(),
		}))
		grant(ctx, access.ActionCreate, resource.ID{Type: resource.TypeLog})
		grant(ctx, access.ActionUpdate, project.ResourceID(proj.Key))
		grant(ctx, access.ActionRetrieve, channel.ResourceID(inside))
	})

	It("Should bind a path through a folder that the importer can retrieve", func(
		ctx SpecContext,
	) {
		grant(ctx, access.ActionRetrieve, folder.ResourceID())
		id := MustSucceed(importLogAs(subjectCtx(ctx, u), fmt.Sprintf("%q", path)))
		Expect(logChannels(ctx, id)).To(Equal([]channel.Key{inside}))
	})

	It(
		"Should not bind a path through a folder that the importer cannot retrieve",
		func(ctx SpecContext) {
			Expect(importLogAs(subjectCtx(ctx, u), fmt.Sprintf("%q", path))).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("no channel is at %s", path)),
				))
		},
	)
})

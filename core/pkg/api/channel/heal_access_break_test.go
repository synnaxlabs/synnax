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
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Heal access break", Ordered, func() {
	var proj project.Project
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		proj = project.Project{Name: "heal-access-break"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	It("Should name no record in the denial of a heal of a schematic that the caller "+
		"cannot retrieve", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
		dead := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &dead)
		})).To(Succeed())
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = dead.Key()
		hidden := schematic.Schematic{
			Name:    "hidden",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Schematic.NewWriter(tx).Create(ctx, &hidden); err != nil {
				return err
			}
			return channelSvc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionCreate},
			resource.ID{Type: resource.TypeChannel},
		)
		Expect(rbacSvc.NewEnforcer(db).Enforce(ctx, access.Request{
			Subject: u.ResourceID(),
			Action:  access.ActionRetrieve,
			Objects: []resource.ID{schematic.ResourceID(hidden.Key)},
		})).To(MatchError(access.ErrDenied))
		Expect(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:     dead.Name,
				DataType: telem.Float32T,
				Virtual:  true,
			}},
		})).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			Not(MatchError(ContainSubstring(hidden.Key.String()))),
		), "the caller cannot retrieve the schematic, so a denial names no record")
	})
})

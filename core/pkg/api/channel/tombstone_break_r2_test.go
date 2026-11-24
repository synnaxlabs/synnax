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
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Tombstone break r2", Ordered, ContinueOnFailure, func() {
	var (
		proj project.Project
		u    user.User
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		proj = project.Project{Name: "tombstone-break-r2"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})
	BeforeEach(func(ctx SpecContext) {
		u = MustSucceed(rbactestutil.CreateUser(
			ctx, db, rbacSvc, userSvc, user.User{Name: uuid.New().String()},
		))
	})
	createChannel := func(ctx SpecContext, g *group.Key) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    g,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	display := func(ctx SpecContext, k channel.Key) resource.ID {
		GinkgoHelper()
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		value.Channel = k
		sch := schematic.Schematic{
			Name:    "pid",
			Project: &proj.Key,
			Nodes: []schematic.Node{{
				Key:    "value",
				Config: schematic.NodeConfig{Variant: value},
			}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.NewWriter(tx).Create(ctx, &sch)
		})).To(Succeed())
		return schematic.ResourceID(sch.Key)
	}
	tombstoned := func(ctx SpecContext, k channel.Key) bool {
		GinkgoHelper()
		var ts []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&ts).
			Exec(ctx, db)).To(Succeed())
		for _, t := range ts {
			if t.Key == k {
				return true
			}
		}
		return false
	}
	deleteAs := func(ctx SpecContext, k channel.Key) error {
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionDelete},
			channel.ResourceID(k),
		)
		_, err := deleteChannels(
			AuthedCtx(ctx, u),
			apichannel.DeleteRequest{Keys: channel.Keys{k}},
		)
		return err
	}

	It("Should delete a root channel that nothing holds through the route", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx, nil)
		Expect(deleteAs(ctx, ch.Key())).To(Succeed())
	})

	It("Should delete a root channel that a schematic holds through the route", func(
		ctx SpecContext,
	) {
		ch := createChannel(ctx, nil)
		display(ctx, ch.Key())
		Expect(deleteAs(ctx, ch.Key())).To(Succeed())
		Expect(tombstoned(ctx, ch.Key())).To(BeTrue())
	})

	It("Should delete a folder channel that a schematic holds with the grants of an "+
		"unheld one", func(ctx SpecContext) {
		folder := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &folder)
		})).To(Succeed())
		ch := createChannel(ctx, &folder.Key)
		display(ctx, ch.Key())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			folder.ResourceID(),
		)
		Expect(deleteAs(ctx, ch.Key())).To(Succeed())
		Expect(tombstoned(ctx, ch.Key())).To(BeTrue())
	})

	It("Should create a root channel at the path of a root tombstone through the "+
		"route", func(ctx SpecContext) {
		dead := createChannel(ctx, nil)
		healed := display(ctx, dead.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionCreate},
			resource.ID{Type: resource.TypeChannel},
		)
		// The heal moves the link of the schematic onto the new channel.
		grantOn(ctx, u.ResourceID(), []access.Action{access.ActionUpdate}, healed)
		Expect(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:     dead.Name,
				DataType: telem.Float32T,
				Virtual:  true,
			}},
		})).Error().ToNot(HaveOccurred())
		Expect(tombstoned(ctx, dead.Key())).To(BeFalse())
	})

	It("Should rename a root channel onto the path of a root tombstone through the "+
		"route", func(ctx SpecContext) {
		dead := createChannel(ctx, nil)
		healed := display(ctx, dead.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		ch := createChannel(ctx, nil)
		// The heal adds a use of the renamed channel to the schematic, which needs
		// retrieve on the channel.
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionUpdate, access.ActionRetrieve},
			channel.ResourceID(ch.Key()),
		)
		// The heal moves the link of the schematic onto the renamed channel.
		grantOn(ctx, u.ResourceID(), []access.Action{access.ActionUpdate}, healed)
		Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
			Key: ch.Key(),
			Actions: []channel.Action{
				channel.NewRenameAction(channel.RenamePayload{Name: dead.Name}),
			},
		})).Error().ToNot(HaveOccurred())
		Expect(tombstoned(ctx, dead.Key())).To(BeFalse())
	})

	It("Should refuse a rename onto the path of a tombstone when the caller cannot "+
		"retrieve the renamed channel", func(ctx SpecContext) {
		dead := createChannel(ctx, nil)
		healed := display(ctx, dead.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		ch := createChannel(ctx, nil)
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionUpdate},
			channel.ResourceID(ch.Key()),
		)
		grantOn(ctx, u.ResourceID(), []access.Action{access.ActionUpdate}, healed)
		Expect(dispatch(AuthedCtx(ctx, u), apichannel.DispatchRequest{
			Key: ch.Key(),
			Actions: []channel.Action{
				channel.NewRenameAction(channel.RenamePayload{Name: dead.Name}),
			},
		})).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(
				"%s cannot retrieve %s",
				u.ResourceID(),
				channel.ResourceID(ch.Key()),
			)),
		))
		Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
	})

	It("Should refuse a create at a tombstone path when the caller cannot update a "+
		"display that the heal moves, and name no hidden display", func(
		ctx SpecContext,
	) {
		dead := createChannel(ctx, nil)
		healed := display(ctx, dead.Key())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Delete(ctx, dead.Key(), false)
		})).To(Succeed())
		grantOn(
			ctx,
			u.ResourceID(),
			[]access.Action{access.ActionCreate},
			resource.ID{Type: resource.TypeChannel},
		)
		Expect(createChannels(AuthedCtx(ctx, u), apichannel.CreateRequest{
			Channels: []apichannel.Channel{{
				Name:     dead.Name,
				DataType: telem.Float32T,
				Virtual:  true,
			}},
		})).Error().To(SatisfyAll(
			MatchError(access.ErrDenied),
			MatchError(ContainSubstring(
				"%s cannot update 1 record that a heal changes",
				u.ResourceID(),
			)),
			Not(MatchError(ContainSubstring(healed.Key))),
		))
		Expect(tombstoned(ctx, dead.Key())).To(BeTrue())
	})
})

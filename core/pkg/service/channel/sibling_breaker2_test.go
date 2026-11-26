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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	dtestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Sibling breaker 2", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		dist     dtestutil.Node
		groupSvc *group.Service
		svc      *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist = dtestutil.NewNode(ctx)
		db = dist.DB
		cfg, gs := serviceConfig(ctx, dist, dist.DB)
		// The sibling rules hold whatever the name rule; free text needs it off.
		cfg.ValidateNames = new(false)
		groupSvc = gs
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	virtualIn := func(name string, g *group.Key) channel.Channel {
		return channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    g,
		}
	}
	folder := func(ctx context.Context, name string, parent *group.Key) group.Group {
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(group.ResourceID(*parent))
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	create := func(ctx context.Context, chs ...*channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, ch := range chs {
				if err := svc.NewWriter(tx).Create(ctx, ch); err != nil {
					return err
				}
			}
			return nil
		})
	}
	pathOf := func(err error) []string {
		GinkgoHelper()
		pe, ok := errors.AsType[validate.PathError](err)
		Expect(ok).To(BeTrue(), "not a path error: %v", err)
		return pe.Path
	}
	stored := func(ctx context.Context, key channel.Key) channel.Channel {
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(key)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	moveTo := func(g *group.Key) channel.Action {
		return channel.NewSetAction(channel.SetPayload{
			Path:  []string{"group"},
			Value: g,
		})
	}
	renameTo := func(name string) channel.Action {
		return channel.NewRenameAction(channel.RenamePayload{Name: name})
	}
	apply := func(ctx context.Context, key channel.Key, acts ...channel.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.NewWriter(tx).Apply(ctx, key, acts...)
			return err
		})
	}

	It("Should name a position in the request when a folder refuses the name of a "+
		"calculated channel's index", func(ctx SpecContext) {
		parent := folder(ctx, UniqueChannelName(), nil)
		name := UniqueChannelName()
		folder(ctx, name+"_time", &parent.Key)
		calc := channel.Channel{
			Name:       name,
			DataType:   telem.Float64T,
			Expression: "return 1.0",
			Group:      &parent.Key,
		}
		err := create(ctx, &calc)
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(pathOf(err)[0]).To(Equal("0"), err.Error())
	})

	It("Should name the holder and the place of a clash in the error", func(
		ctx SpecContext,
	) {
		parent := folder(ctx, UniqueChannelName(), nil)
		inGroup, atRoot := UniqueChannelName(), UniqueChannelName()
		folder(ctx, inGroup, &parent.Key)
		folder(ctx, atRoot, nil)
		ch := virtualIn(inGroup, &parent.Key)
		Expect(create(ctx, &ch)).To(MatchError(ContainSubstring(
			"a group named '" + inGroup + "' already exists in the same group",
		)))
		ch = virtualIn(atRoot, nil)
		Expect(create(ctx, &ch)).To(MatchError(ContainSubstring(
			"a group named '" + atRoot + "' already exists at the top level",
		)))
		held := virtualIn(UniqueChannelName(), &parent.Key)
		Expect(create(ctx, &held)).To(Succeed())
		g := group.Group{Name: held.Name, Scope: resource.TypeChannel}
		g.Parent = new(group.ResourceID(parent.Key))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(MatchError(ContainSubstring(
			"a channel named '" + held.Name + "' already exists in the same group",
		)))
	})

	It("Should refuse a channel and a folder of one name made in one transaction",
		func(ctx SpecContext) {
			name := UniqueChannelName()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				g := group.Group{Name: name, Scope: resource.TypeChannel}
				if err := groupSvc.NewWriter(tx).Create(ctx, &g); err != nil {
					return err
				}
				ch := virtualIn(name, nil)
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(MatchError(ContainSubstring("already exists at the top level")))
			name = UniqueChannelName()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				ch := virtualIn(name, nil)
				if err := svc.NewWriter(tx).Create(ctx, &ch); err != nil {
					return err
				}
				g := group.Group{Name: name, Scope: resource.TypeChannel}
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(MatchError(ContainSubstring("already exists at the top level")))
		},
	)

	It("Should apply a move and a rename in either order onto a name free in the "+
		"new folder", func(ctx SpecContext) {
		name := UniqueChannelName()
		folder(ctx, name, nil)
		dest := folder(ctx, UniqueChannelName(), nil)
		a := virtualIn(UniqueChannelName(), nil)
		b := virtualIn(UniqueChannelName(), nil)
		Expect(create(ctx, &a, &b)).To(Succeed())
		Expect(apply(ctx, a.Key(), moveTo(&dest.Key), renameTo(name))).To(Succeed())
		Expect(stored(ctx, a.Key()).Name).To(Equal(name))
		other := folder(ctx, UniqueChannelName(), nil)
		Expect(apply(
			ctx, b.Key(), renameTo(name+"b"), moveTo(&other.Key), renameTo(name+"c"),
		)).To(Succeed())
		got := stored(ctx, b.Key())
		Expect(got.Name).To(Equal(name + "c"))
		Expect(*got.Group).To(Equal(other.Key))
		storage := MustSucceed(
			dist.Storage.TS.RetrieveChannel(ctx, cesium.ChannelKey(b.Key())),
		)
		Expect(storage.Name).To(Equal(name + "c"))
	})

	It("Should keep the storage name when a rename and a move are refused", func(
		ctx SpecContext,
	) {
		dest := folder(ctx, UniqueChannelName(), nil)
		taken := UniqueChannelName()
		folder(ctx, taken, &dest.Key)
		ch := virtualIn(UniqueChannelName(), nil)
		ch.Virtual = false
		ch.DataType = telem.TimestampT
		ch.IsIndex = true
		Expect(create(ctx, &ch)).To(Succeed())
		Expect(apply(ctx, ch.Key(), renameTo(taken), moveTo(&dest.Key))).
			To(MatchError(ContainSubstring("already exists in the same group")))
		Expect(stored(ctx, ch.Key()).Name).To(Equal(ch.Name))
		storage := MustSucceed(
			dist.Storage.TS.RetrieveChannel(ctx, cesium.ChannelKey(ch.Key())),
		)
		Expect(storage.Name).To(Equal(ch.Name))
	})

	It("Should name the position of a sibling clash in a batch rename", func(
		ctx SpecContext,
	) {
		parent := folder(ctx, UniqueChannelName(), nil)
		taken := UniqueChannelName()
		folder(ctx, taken, &parent.Key)
		a := virtualIn(UniqueChannelName(), nil)
		b := virtualIn(UniqueChannelName(), &parent.Key)
		Expect(create(ctx, &a, &b)).To(Succeed())
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).RenameMany(
				ctx,
				channel.Keys{a.Key(), b.Key()},
				[]string{UniqueChannelName(), taken},
				false,
			)
		})
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(pathOf(err)).To(Equal([]string{"1", "name"}))
	})

	It("Should refuse a folder moved or renamed onto a channel's name", func(
		ctx SpecContext,
	) {
		parent := folder(ctx, UniqueChannelName(), nil)
		held := virtualIn(UniqueChannelName(), &parent.Key)
		Expect(create(ctx, &held)).To(Succeed())
		moved := folder(ctx, held.Name, nil)
		id := group.ResourceID(parent.Key)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).SetParent(ctx, &id, moved.Key)
		})).To(MatchError(ContainSubstring(
			"a channel named '" + held.Name + "' already exists in the same group",
		)))
		root := virtualIn(UniqueChannelName(), nil)
		Expect(create(ctx, &root)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Rename(ctx, moved.Key, root.Name)
		})).To(MatchError(ContainSubstring(
			"a channel named '" + root.Name + "' already exists at the top level",
		)))
	})

	It("Should keep the name when a move fails after the rename is staged", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		calc := channel.Channel{
			Name:       name,
			DataType:   telem.Float64T,
			Expression: "return 1.0",
		}
		Expect(create(ctx, &calc)).To(Succeed())
		dest := folder(ctx, UniqueChannelName(), nil)
		folder(ctx, name+"_time", &dest.Key)
		renamed := UniqueChannelName()
		Expect(apply(ctx, calc.Key(), renameTo(renamed), moveTo(&dest.Key))).
			To(MatchError(ContainSubstring("already exists in the same group")))
		got := stored(ctx, calc.Key())
		Expect(got.Name).To(Equal(name))
		Expect(got.Group).To(BeNil())
		Expect(svc.NewRetrieve().
			Where(channel.MatchNames(renamed)).
			Exists(ctx, db)).To(BeFalse())
	})

	It("Should keep a channel's name free in a folder after the channel leaves it",
		func(ctx SpecContext) {
			parent := folder(ctx, UniqueChannelName(), nil)
			ch := virtualIn(UniqueChannelName(), &parent.Key)
			Expect(create(ctx, &ch)).To(Succeed())
			Expect(apply(ctx, ch.Key(), moveTo(nil))).To(Succeed())
			folder(ctx, ch.Name, &parent.Key)
		},
	)
})

var _ = Describe("Sibling breaker 2 over stored names", func() {
	It("Should refuse a folder beside a stored channel of its name in another normal "+
		"form", func(ctx SpecContext) {
		fresh := dtestutil.NewNode(ctx)
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		// The sibling rules hold whatever the name rule; free text needs it off.
		cfg.ValidateNames = new(false)
		// A v0.62 Core stored a name as sent.
		stored := channel.Channel{
			Name:        "Stand A\u030A brk2",
			Leaseholder: node.KeyFree,
			LocalKey:    7,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
		createStored(ctx, fresh.DB, stored)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		// The name index compares the stored name in NFC.
		ch := channel.Channel{
			Name:     "Stand \u00C5 brk2",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(MatchError(ContainSubstring("channel with name")))
		f := group.Group{Name: "Stand \u00C5 brk2", Scope: resource.TypeChannel}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &f)
		})).To(MatchError(ContainSubstring("already exists at the top level")))
	})
})

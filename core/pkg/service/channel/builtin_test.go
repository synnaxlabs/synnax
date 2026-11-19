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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Built-in channels", func() {
	var (
		db        *gorp.DB
		svc       *channel.Service
		statusSvc *status.Service
		groupSvc  *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		var cfg channel.ServiceConfig
		cfg, groupSvc = serviceConfig(ctx, node, db)
		statusSvc = cfg.Status
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})

	declare := func(ctx SpecContext, ch *channel.Channel) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Declare(ctx, ch)
		})
	}

	create := func(ctx SpecContext, ch *channel.Channel) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, ch)
		})).To(Succeed())
	}

	retrieve := func(ctx SpecContext, name string) []channel.Channel {
		GinkgoHelper()
		var res []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entries(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	command := func() channel.Channel {
		return channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.JSONT,
			Virtual:  true,
		}
	}

	It("Should create a channel when no channel holds the name", func(ctx SpecContext) {
		ch := command()
		Expect(declare(ctx, &ch)).To(Succeed())
		Expect(ch.Key()).ToNot(BeZero())
		Expect(svc.IsBuiltIn(ch.Key())).To(BeTrue())
		Expect(svc.BuiltIns()).To(ContainElement(ch.Key()))
		Expect(retrieve(ctx, ch.Name)).To(HaveLen(1))
	})

	It("Should adopt a channel of the same data type and virtual flag", func(
		ctx SpecContext,
	) {
		held := command()
		create(ctx, &held)
		ch := command()
		ch.Name = held.Name
		Expect(declare(ctx, &ch)).To(Succeed())
		Expect(ch.Key()).To(Equal(held.Key()))
		Expect(svc.IsBuiltIn(held.Key())).To(BeTrue())
	})

	It("Should take an adopted channel out of the license count", func(
		ctx SpecContext,
	) {
		held := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		create(ctx, &held)
		base := svc.CountLimited()
		ch := channel.Channel{
			Name:     held.Name,
			DataType: telem.TimestampT,
			IsIndex:  true,
		}
		Expect(declare(ctx, &ch)).To(Succeed())
		Expect(svc.CountLimited()).To(Equal(base - 1))
	})

	It("Should rename a channel of another shape with the first free suffix", func(
		ctx SpecContext,
	) {
		held := command()
		held.DataType = telem.Float32T
		create(ctx, &held)
		taken := command()
		taken.Name = held.Name + "_2"
		create(ctx, &taken)
		ch := command()
		ch.Name = held.Name
		Expect(declare(ctx, &ch)).To(Succeed())
		Expect(ch.Key()).ToNot(Equal(held.Key()))
		renamed := retrieve(ctx, held.Name+"_3")
		Expect(renamed).To(HaveLen(1))
		Expect(renamed[0].Key()).To(Equal(held.Key()))
		var warnings []status.Status[any]
		Expect(statusSvc.NewRetrieve[any]().
			Where(status.MatchNames[any](held.Name+"_3")).
			Entries(&warnings).
			Exec(ctx, db)).To(Succeed())
		Expect(warnings).To(HaveLen(1))
		Expect(warnings[0].Variant).To(Equal(status.VariantWarning))
		Expect(warnings[0].Message).To(Equal("Renamed channel"))
		Expect(warnings[0].Description).To(Equal(
			"The Core needs the name " + held.Name + " for a built-in channel, so " +
				"the channel " + held.Name + " is now " + held.Name + "_3.",
		))
	})

	It("Should rename every other holder of the name when it adopts one", func(
		ctx SpecContext,
	) {
		// Create refuses a held name, so only a store from before has two holders.
		fresh := testutil.NewNode(ctx)
		name := UniqueChannelName()
		other := channel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    7,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
		held := channel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    8,
			DataType:    telem.JSONT,
			Virtual:     true,
		}
		createStored(ctx, fresh.DB, other, held)
		freshSvc := openService(ctx, fresh)
		ch := channel.Channel{Name: name, DataType: telem.JSONT, Virtual: true}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return freshSvc.NewWriter(tx).Declare(ctx, &ch)
		})).To(Succeed())
		Expect(ch.Key()).To(Equal(held.Key()))
		var renamed []channel.Channel
		Expect(freshSvc.NewRetrieve().
			Where(channel.MatchNames(name+"_2")).
			Entries(&renamed).
			Exec(ctx, fresh.DB)).To(Succeed())
		Expect(renamed).To(HaveLen(1))
		Expect(renamed[0].Key()).To(Equal(other.Key()))
	})

	It(
		"Should rename a channel of the same shape in a folder and create at the root",
		func(
			ctx SpecContext,
		) {
			g := group.Group{Name: "folder", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			held := command()
			held.Group = &g.Key
			create(ctx, &held)
			ch := command()
			ch.Name = held.Name
			Expect(declare(ctx, &ch)).To(Succeed())
			Expect(ch.Key()).ToNot(Equal(held.Key()))
			Expect(ch.Group).To(BeNil())
			renamed := retrieve(ctx, held.Name+"_2")
			Expect(renamed).To(HaveLen(1))
			Expect(renamed[0].Key()).To(Equal(held.Key()))
			// The user can still empty the folder and delete it.
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Delete(ctx, held.Key(), false); err != nil {
					return err
				}
				return groupSvc.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
		},
	)

	It("Should let a renamed user channel be deleted", func(ctx SpecContext) {
		held := command()
		held.DataType = telem.Float32T
		create(ctx, &held)
		ch := command()
		ch.Name = held.Name
		Expect(declare(ctx, &ch)).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, held.Key(), false)
		})).To(Succeed())
	})

	It("Should refuse a name that a built-in of another shape holds", func(
		ctx SpecContext,
	) {
		first := command()
		Expect(declare(ctx, &first)).To(Succeed())
		second := command()
		second.Name = first.Name
		second.DataType = telem.Float32T
		Expect(declare(ctx, &second)).To(MatchError(
			ContainSubstring(
				`built-in channel "` + first.Name +
					`" has another data type or virtual flag`,
			),
		))
	})

	It("Should not mark a channel whose transaction rolls back", func(
		ctx SpecContext,
	) {
		ch := command()
		rollback := errors.New("rollback")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := svc.NewWriter(tx).Declare(ctx, &ch); err != nil {
				return err
			}
			return rollback
		})).To(MatchError(rollback))
		Expect(svc.IsBuiltIn(ch.Key())).To(BeFalse())
	})

	Describe("Guards", func() {
		guardError := func(verb, name string) OmegaMatcher {
			return SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"can't "+verb+" built-in channel \""+name+"\"",
				)),
			)
		}
		var ch channel.Channel
		BeforeEach(func(ctx SpecContext) {
			ch = command()
			Expect(declare(ctx, &ch)).To(Succeed())
		})

		It("Should refuse to delete a built-in channel", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(guardError("delete", ch.Name))
		})

		It("Should refuse to rename a built-in channel", func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, ch.Key(), "renamed", false)
			})).To(guardError("rename", ch.Name))
		})

		It("Should refuse to move a built-in channel", func(ctx SpecContext) {
			g := group.Group{Name: "folder", Scope: resource.TypeChannel}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetGroup(ctx, &g.Key, ch.Key())
			})).To(guardError("move", ch.Name))
		})
	})
})

var _ = Describe("Built-in channels across restarts", func() {
	It("Should keep the key of a built-in channel", func(ctx SpecContext) {
		store := DeferClose(storagetestutil.OpenLayer(ctx))
		declare := func() channel.Key {
			GinkgoHelper()
			node := testutil.OpenNode(
				ctx,
				distribution.LayerConfig{Storage: store.Layer},
			)
			defer func() { Expect(node.Close()).To(Succeed()) }()
			groupSvc := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
				DB: node.DB,
			}))
			defer func() { Expect(groupSvc.Close()).To(Succeed()) }()
			statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
				DB: node.DB,
			}))
			defer func() { Expect(statusSvc.Close()).To(Succeed()) }()
			svc := MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				DB:      node.DB,
				HostKey: node.Identity.HostKey,
				Status:  statusSvc,
			}))
			defer func() { Expect(svc.Close()).To(Succeed()) }()
			ch := channel.Channel{
				Name:     "sy_restart",
				DataType: telem.JSONT,
				Virtual:  true,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Declare(ctx, &ch)
			})).To(Succeed())
			Expect(svc.IsBuiltIn(ch.Key())).To(BeTrue())
			return ch.Key()
		}
		first := declare()
		Expect(declare()).To(Equal(first))
	})
})

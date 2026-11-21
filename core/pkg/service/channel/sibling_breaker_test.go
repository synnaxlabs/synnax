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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Sibling breaker", Ordered, ContinueOnFailure, func() {
	var (
		db       *gorp.DB
		groupSvc *group.Service
		svc      *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, gs := serviceConfig(ctx, node, node.DB)
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
	folder := func(ctx SpecContext, name string, parent *group.Key) group.Group {
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(group.ResourceID(*parent))
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	pathOf := func(err error) []string {
		GinkgoHelper()
		pe, ok := errors.AsType[validate.PathError](err)
		Expect(ok).To(BeTrue(), "not a path error: %v", err)
		return pe.Path
	}

	It(
		"Should name the position of a created channel that a folder's name refuses",
		func(
			ctx SpecContext,
		) {
			taken := UniqueChannelName()
			folder(ctx, taken, nil)
			chs := []channel.Channel{
				virtualIn(UniqueChannelName(), nil),
				virtualIn(UniqueChannelName(), nil),
				virtualIn(taken, nil),
			}
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &chs)
			})
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(pathOf(err)).To(Equal([]string{"2", "name"}))
			one := virtualIn(taken, nil)
			err = db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &one)
			})
			Expect(pathOf(err)).To(Equal([]string{"0", "name"}))
		},
	)

	It(
		"Should name the position of a renamed channel that a folder's name refuses",
		func(
			ctx SpecContext,
		) {
			taken := UniqueChannelName()
			folder(ctx, taken, nil)
			chs := []channel.Channel{
				virtualIn(UniqueChannelName(), nil),
				virtualIn(UniqueChannelName(), nil),
				virtualIn(UniqueChannelName(), nil),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).CreateMany(ctx, &chs)
			})).To(Succeed())
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).RenameMany(
					ctx,
					channel.KeysFromChannels(chs),
					[]string{UniqueChannelName(), UniqueChannelName(), taken},
					false,
				)
			})
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(pathOf(err)).To(Equal([]string{"2", "name"}))
		},
	)

	It(
		"Should rename and move a channel in one dispatch onto a name free in its new folder",
		func(
			ctx SpecContext,
		) {
			name := UniqueChannelName()
			folder(ctx, name, nil)
			dest := folder(ctx, UniqueChannelName(), nil)
			ch := virtualIn(UniqueChannelName(), nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(
					ctx,
					ch.Key(),
					channel.NewRenameAction(channel.RenamePayload{Name: name}),
					channel.NewSetAction(channel.SetPayload{
						Path:  []string{"group"},
						Value: &dest.Key,
					}),
				)
				return err
			})).To(Succeed())
			var stored channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchKeys(ch.Key())).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Name).To(Equal(name))
			Expect(*stored.Group).To(Equal(dest.Key))
		},
	)

	It("Should swap the names of a folder and a channel in one transaction", func(
		ctx SpecContext,
	) {
		parent := folder(ctx, UniqueChannelName(), nil)
		a, b := UniqueChannelName(), UniqueChannelName()
		f := folder(ctx, a, &parent.Key)
		ch := virtualIn(b, &parent.Key)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w, gw := svc.NewWriter(tx), groupSvc.NewWriter(tx)
			if err := gw.Rename(ctx, f.Key, UniqueChannelName()); err != nil {
				return err
			}
			if err := w.Rename(ctx, ch.Key(), a, false); err != nil {
				return err
			}
			return gw.Rename(ctx, f.Key, b)
		})).To(Succeed())
	})

	It(
		"Should take a folder's name for a channel after the folder's delete in one transaction",
		func(
			ctx SpecContext,
		) {
			name := UniqueChannelName()
			f := folder(ctx, name, nil)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := groupSvc.NewWriter(tx).Delete(ctx, f.Key); err != nil {
					return err
				}
				ch := virtualIn(name, nil)
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
		},
	)

	It("Should refuse a rename onto a folder's name sent in another normal form", func(
		ctx SpecContext,
	) {
		suffix := UniqueChannelName()
		folder(ctx, "Stand \u00C5 "+suffix, nil)
		ch := virtualIn(UniqueChannelName(), nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				Rename(ctx, ch.Key(), "Stand A\u030A "+suffix, false)
		})).To(MatchError(MatchRegexp("a (group|channel) named '.*' already exists")))
		decomposed := virtualIn("Stand A\u030A "+suffix, nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &decomposed)
		})).To(MatchError(MatchRegexp("a (group|channel) named '.*' already exists")))
	})

	It(
		"Should refuse a channel moved to the root beside a root folder of its name",
		func(
			ctx SpecContext,
		) {
			name := UniqueChannelName()
			parent := folder(ctx, UniqueChannelName(), nil)
			folder(ctx, name, nil)
			ch := virtualIn(name+"x", &parent.Key)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).
					Rename(ctx, ch.Key(), name, false); err != nil {
					return err
				}
				return svc.NewWriter(tx).SetGroup(ctx, nil, ch.Key())
			})).To(MatchError(MatchRegexp("a (group|channel) named '.*' already exists")))
		},
	)
	It("Should use up no channel key on a create that a folder's name refuses", func(
		ctx SpecContext,
	) {
		taken := UniqueChannelName()
		folder(ctx, taken, nil)
		first := virtualIn(UniqueChannelName(), nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &first)
		})).To(Succeed())
		sameName := virtualIn(first.Name, nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &sameName)
		})).To(MatchError(ContainSubstring("already exists")))
		before := virtualIn(UniqueChannelName(), nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &before)
		})).To(Succeed())
		Expect(before.LocalKey).To(Equal(first.LocalKey + 1))
		refused := virtualIn(taken, nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &refused)
		})).To(MatchError(MatchRegexp("a (group|channel) named '.*' already exists")))
		after := virtualIn(UniqueChannelName(), nil)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &after)
		})).To(Succeed())
		Expect(after.LocalKey).To(Equal(before.LocalKey + 1))
	})
})

var _ = Describe("Sibling breaker over stored duplicates", func() {
	It("Should keep a stored channel and folder of one name writable", func(
		ctx SpecContext,
	) {
		fresh := testutil.NewNode(ctx)
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		f := group.Group{Name: "pt_dup", Scope: resource.TypeChannel}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &f)
		})).To(Succeed())
		stored := channel.Channel{
			Name:        "pt_dup",
			Leaseholder: node.KeyFree,
			LocalKey:    7,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
		createStored(ctx, fresh.DB, stored)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		key := stored.Key()
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Rename(ctx, key, "pt_dup", false)
		})).To(Succeed())
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Rename(ctx, f.Key, "pt_dup")
		})).To(Succeed())
		again := group.Group{Name: "pt_dup", Scope: resource.TypeChannel}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &again)
		})).To(MatchError(MatchRegexp("a (group|channel) named '.*' already exists")))
	})
})

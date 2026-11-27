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
	dtestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Path breaker 3", func() {
	// paused opens a channel service on a store whose next commit stops once its
	// writes are visible, before Gorp applies its index changes.
	type paused struct {
		db     *gorp.DB
		kv     *pausingKV
		groups *group.Service
		svc    *channel.Service
	}
	open := func(ctx SpecContext) paused {
		owned := DeferClose(storagetestutil.OpenLayer(ctx))
		kv := &pausingKV{DB: owned.KV}
		gdb := gorp.Wrap(
			kv,
			gorp.WithCodec(encoding.NewDecodeFallbackCodec(orc.Codec, msgpack.Codec)),
		)
		DeferCleanup(GoRun(ctx, gdb.Run))
		n := dtestutil.NewNode(ctx, distribution.LayerConfig{Storage: &storage.Layer{
			KV:   kv,
			TS:   owned.TS,
			Gorp: gdb,
		}})
		cfg, groups := serviceConfig(ctx, n, gdb)
		return paused{
			db:     gdb,
			kv:     kv,
			groups: groups,
			svc:    MustOpen(channel.OpenService(ctx, cfg)),
		}
	}
	// during runs commit in its own transaction and returns the keys at path that a
	// retrieve against the DB reads while the commit is in flight.
	during := func(
		ctx SpecContext,
		p paused,
		path string,
		commit func(tx gorp.Tx) error,
	) []channel.Key {
		GinkgoHelper()
		applied, release := make(chan struct{}), make(chan struct{})
		hook := func() {
			close(applied)
			<-release
		}
		p.kv.hook.Store(&hook)
		done := make(chan error, 1)
		go func() { done <- p.db.WithTx(ctx, commit) }()
		Eventually(applied).Should(BeClosed())
		var chs []channel.Channel
		// The commit waits on release, so it closes even when the retrieve fails.
		func() {
			defer close(release)
			Expect(p.svc.NewRetrieve().
				Where(channel.MatchPaths(path)).
				Entries(&chs).
				Exec(ctx, p.db)).To(Succeed())
		}()
		Expect(<-done).To(Succeed())
		return channel.KeysFromChannels(chs)
	}
	at := func(ctx SpecContext, p paused, path string) []channel.Key {
		GinkgoHelper()
		var chs []channel.Channel
		Expect(p.svc.NewRetrieve().
			Where(channel.MatchPaths(path)).
			Entries(&chs).
			Exec(ctx, p.db)).To(Succeed())
		return channel.KeysFromChannels(chs)
	}

	It("Should find a channel at a path that it holds before and after a commit, "+
		"while the commit is in flight", func(ctx SpecContext) {
		p := open(ctx)
		dir, name := UniqueChannelName(), UniqueChannelName()
		old := group.Group{Name: dir, Scope: resource.TypeChannel}
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(p.db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := p.groups.NewWriter(tx).Create(ctx, &old); err != nil {
				return err
			}
			ch.Group = &old.Key
			return p.svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		path := dir + "." + name
		Expect(at(ctx, p, path)).To(ConsistOf(ch.Key()))
		// The commit gives the name of the folder to a new folder and moves the
		// channel into it, so the channel sits at path before and after.
		got := during(ctx, p, path, func(tx gorp.Tx) error {
			gw := p.groups.NewWriter(tx)
			if err := gw.Rename(ctx, old.Key, dir+"_old"); err != nil {
				return err
			}
			next := group.Group{Name: dir, Scope: resource.TypeChannel}
			if err := gw.Create(ctx, &next); err != nil {
				return err
			}
			return p.svc.NewWriter(tx).SetGroup(ctx, &next.Key, ch.Key())
		})
		Expect(at(ctx, p, path)).To(ConsistOf(ch.Key()))
		Expect(got).To(ConsistOf(ch.Key()), "path %q during the commit", path)
	})

	It("Should never return a channel for a path that it holds neither before nor "+
		"after a commit, while the commit is in flight", func(ctx SpecContext) {
		p := open(ctx)
		dir, name := UniqueChannelName(), UniqueChannelName()
		f := group.Group{Name: dir, Scope: resource.TypeChannel}
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(p.db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := p.groups.NewWriter(tx).Create(ctx, &f); err != nil {
				return err
			}
			return p.svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		path := dir + "." + name
		Expect(at(ctx, p, path)).To(BeEmpty())
		// Before the commit the channel sits at name, and after it at dir_b.name.
		got := during(ctx, p, path, func(tx gorp.Tx) error {
			if err := p.groups.NewWriter(tx).Rename(ctx, f.Key, dir+"_b"); err != nil {
				return err
			}
			return p.svc.NewWriter(tx).SetGroup(ctx, &f.Key, ch.Key())
		})
		Expect(at(ctx, p, path)).To(BeEmpty())
		Expect(at(ctx, p, dir+"_b."+name)).To(ConsistOf(ch.Key()))
		Expect(got).To(BeEmpty(), "path %q during the commit", path)
	})
})

var _ = Describe("Sibling breaker 3", func() {
	It("Should name the position of a calculated channel whose adopted index a "+
		"folder's name refuses", func(ctx SpecContext) {
		dist := dtestutil.NewNode(ctx)
		cfg, groups := serviceConfig(ctx, dist, dist.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		db := dist.DB
		name := UniqueChannelName()
		orphan := channel.Channel{
			Name:        name + "_time",
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Virtual:     true,
			Leaseholder: node.KeyFree,
		}
		parent := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := svc.NewWriter(tx).Create(ctx, &orphan); err != nil {
				return err
			}
			gw := groups.NewWriter(tx)
			if err := gw.Create(ctx, &parent); err != nil {
				return err
			}
			taken := group.Group{
				Name:   orphan.Name,
				Scope:  resource.TypeChannel,
				Parent: new(group.ResourceID(parent.Key)),
			}
			return gw.Create(ctx, &taken)
		})).To(Succeed())
		chs := []channel.Channel{
			{Name: UniqueChannelName(), DataType: telem.Float32T, Virtual: true},
			{
				Name:       name,
				DataType:   telem.Float64T,
				Expression: "return 1.0",
				Group:      &parent.Key,
			},
		}
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, &chs)
		})
		Expect(err).To(MatchError(validate.ErrValidation))
		pe, ok := errors.AsType[validate.PathError](err)
		Expect(ok).To(BeTrue(), "not a path error: %v", err)
		Expect(pe.Path[0]).To(Equal("1"), err.Error())
	})
})

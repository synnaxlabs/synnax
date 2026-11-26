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
	"slices"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	dtestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Path breaker", func() {
	It("Should never return a channel for a path its record does not hold, during a "+
		"commit", func(ctx SpecContext) {
		owned := DeferClose(storagetestutil.OpenLayer(ctx))
		paused := &pausingKV{DB: owned.KV}
		gdb := gorp.Wrap(
			paused,
			gorp.WithCodec(encoding.NewDecodeFallbackCodec(orc.Codec, msgpack.Codec)),
		)
		DeferCleanup(GoRun(ctx, gdb.Run))
		n := dtestutil.NewNode(ctx, distribution.LayerConfig{Storage: &storage.Layer{
			KV:   paused,
			TS:   owned.TS,
			Gorp: gdb,
		}})
		cfg, _ := serviceConfig(ctx, n, gdb)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		old, next := UniqueChannelName(), UniqueChannelName()
		ch := channel.Channel{Name: old, DataType: telem.Float32T, Virtual: true}
		Expect(gdb.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())

		applied, release := make(chan struct{}), make(chan struct{})
		hook := func() {
			close(applied)
			<-release
		}
		paused.hook.Store(&hook)
		done := make(chan error, 1)
		go func() {
			done <- gdb.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Rename(ctx, ch.Key(), next, false)
			})
		}()
		Eventually(applied).Should(BeClosed())
		var byKey, byName, byPath []channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Entries(&byKey).
			Exec(ctx, gdb)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(channel.MatchNames(old)).
			Entries(&byName).
			Exec(ctx, gdb)).To(Succeed())
		Expect(svc.NewRetrieve().
			Where(channel.MatchPaths(old)).
			Entries(&byPath).
			Exec(ctx, gdb)).To(Succeed())
		close(release)
		Expect(<-done).To(Succeed())
		Expect(byKey).To(ConsistOf(HaveField("Name", next)))
		Expect(byName).To(BeEmpty())
		Expect(byPath).To(BeEmpty(), "path %q returned %v", old, byPath)
	})

	It("Should read no channel record for a path line with an empty in", func(
		ctx SpecContext,
	) {
		fresh := dtestutil.NewNode(ctx)
		cfg, _ := serviceConfig(ctx, fresh, fresh.DB)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		corruptChannel(ctx, fresh.DB, ch.Key())
		retrieve := func(w channel.Where) error {
			var chs []channel.Channel
			return svc.NewRetrieve().
				Where(MustSucceed(channel.MatchWhere(w))).
				Entries(&chs).
				Exec(ctx, fresh.DB)
		}
		Expect(retrieve(channel.Where{Name: &filter.String{In: []string{}}})).
			To(Succeed())
		Expect(retrieve(channel.Where{
			Key: &filter.Set[channel.Key]{In: []channel.Key{}},
		})).To(Succeed())
		Expect(retrieve(channel.Where{Path: &filter.Set[string]{In: []string{}}})).
			To(Succeed())
	})
})

// pausingKV is a kv.DB whose next transaction commit runs hook once its writes are
// visible, before gorp applies the index changes of the commit.
type pausingKV struct {
	kv.DB
	hook atomic.Pointer[func()]
}

func (p *pausingKV) OpenTx() kv.Tx { return &pausingTx{Tx: p.DB.OpenTx(), kv: p} }

type pausingTx struct {
	kv.Tx
	kv *pausingKV
}

func (t *pausingTx) Commit(ctx context.Context, opts ...any) error {
	hook := t.kv.hook.Swap(nil)
	if hook == nil {
		return t.Tx.Commit(ctx, opts...)
	}
	wrapped := slices.Clone(opts)
	for i, o := range wrapped {
		if applied, ok := o.(kv.OnApplied); ok {
			wrapped[i] = kv.OnApplied(func() {
				(*hook)()
				applied()
			})
		}
	}
	return t.Tx.Commit(ctx, wrapped...)
}

func corruptChannel(ctx context.Context, db *gorp.DB, key channel.Key) {
	GinkgoHelper()
	iter := MustSucceed(gorp.WrapReader[channel.Key, channel.Channel](db).
		OpenIterator(gorp.IterOptions{}))
	var raw []byte
	for iter.First(); iter.Valid(); iter.Next() {
		if v := iter.Value(); v != nil && v.GorpKey() == key {
			raw = slices.Clone(iter.Key())
			break
		}
	}
	Expect(iter.Close()).To(Succeed())
	Expect(raw).ToNot(BeNil())
	Expect(db.Set(ctx, raw, []byte("corrupt"))).To(Succeed())
}

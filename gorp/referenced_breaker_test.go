// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"context"
	"slices"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// pausedKV holds the next armed commit after its writes are visible to reads and
// before the DB runs the commit's cleanups, until release closes.
type pausedKV struct {
	kv.DB
	armed   atomic.Bool
	applied chan struct{}
	release chan struct{}
}

func (p *pausedKV) OpenTx() kv.Tx { return &pausedTx{Tx: p.DB.OpenTx(), p: p} }

type pausedTx struct {
	kv.Tx
	p *pausedKV
}

func (t *pausedTx) Commit(ctx context.Context, opts ...any) error {
	if t.p.armed.CompareAndSwap(true, false) {
		opts = slices.Clone(opts)
		for i, o := range opts {
			if applied, ok := o.(kv.OnApplied); ok {
				opts[i] = kv.OnApplied(func() {
					close(t.p.applied)
					<-t.p.release
					applied()
				})
			}
		}
	}
	return t.Tx.Commit(ctx, opts...)
}

var _ = Describe("FilterReferenced breaker", func() {
	var (
		p        *pausedKV
		db       *gorp.DB
		channels *gorp.Table[uint32, usedChannel]
		symbols  *gorp.Table[string, usedSymbol]
		used     = func(referenced bool) gorp.Filter[uint32, usedChannel] {
			return channels.FilterReferenced(referenced, usedReadTag, usedWriteTag)
		}
		retrieve = func(
			ctx SpecContext,
			snap gorp.Snapshot,
			f gorp.Filter[uint32, usedChannel],
		) []uint32 {
			GinkgoHelper()
			var found []usedChannel
			Expect(channels.NewRetrieve().Where(f).Entries(&found).Exec(ctx, snap)).
				To(Succeed())
			keys := make([]uint32, len(found))
			for i, c := range found {
				keys[i] = c.Key
			}
			return keys
		}
	)

	BeforeEach(func(ctx SpecContext) {
		p = &pausedKV{
			DB:      memkv.New(),
			applied: make(chan struct{}),
			release: make(chan struct{}),
		}
		db = DeferClose(gorp.Wrap(p, gorp.WithCodec(msgpack.Codec)))
		DeferCleanup(func() {
			select {
			case <-p.release:
			default:
				close(p.release)
			}
		})
		StartRun(db.Run)
		channels = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[uint32, usedChannel]{
			DB:     db,
			Name:   "channel",
			Issued: func(uint32) bool { return true },
		}))
		symbols = openUsedHolder(
			ctx, db, "symbol", usedReadTag, gorp.Weak,
			func(e *usedSymbol) *[]uint32 { return &e.Channels },
		)
		for _, k := range []uint32{1, 2, 3, 4} {
			Expect(channels.NewCreate().
				Entry(&usedChannel{Key: k}).
				Exec(ctx, db)).To(Succeed())
		}
	})

	It("Should count the holder of a commit in flight, as Dependents does", func(
		ctx SpecContext,
	) {
		p.armed.Store(true)
		done := make(chan error, 1)
		go func() {
			done <- symbols.NewCreate().
				Entry(&usedSymbol{Key: "a", Channels: []uint32{1}}).
				Exec(ctx, db)
		}()
		Eventually(p.applied).Should(BeClosed())
		var holders []usedSymbol
		Expect(symbols.NewRetrieve().Entries(&holders).Exec(ctx, db)).To(Succeed())
		deps := MustSucceed(channels.Dependents(ctx, db, 1))
		unused := retrieve(ctx, db, used(false))
		inUse := retrieve(ctx, db, used(true))
		close(p.release)
		Expect(<-done).To(Succeed())
		Expect(holders).To(HaveLen(1))
		Expect(deps).To(HaveLen(1))
		Expect(map[string][]uint32{"used": inUse, "unused": unused}).To(Equal(
			map[string][]uint32{"used": {1}, "unused": {2, 3, 4}},
		))
	})

	It("Should agree with Dependents on a text key that is not canonical", func(
		ctx SpecContext,
	) {
		notes := MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, usedNote]{
			DB:   db,
			Name: "note",
			References: []gorp.Reference[string, usedNote]{
				gorp.NewTargetReference(gorp.TargetReferenceConfig[string, usedNote]{
					Name:     "target",
					Tables:   []string{"channel"},
					OnDelete: gorp.Detach,
					Kind:     gorp.Use,
					Tag:      usedReadTag,
					Targets: func(n *usedNote) []gorp.Target {
						if n.Target == nil {
							return nil
						}
						return []gorp.Target{*n.Target}
					},
					Detach: func(n *usedNote, _ gorp.Target) { n.Target = nil },
				}),
			},
		}))
		target := gorp.Target{Table: "channel", Key: "03"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return notes.NewUncheckedWriter(tx).Set(
				ctx,
				usedNote{Key: "n", Target: &target},
			)
		})).To(Succeed())
		deps := MustSucceed(channels.Dependents(ctx, db, 3))
		inUse := retrieve(ctx, db, used(true))
		Expect(slices.Contains(inUse, 3)).To(Equal(len(deps) > 0))
	})
})

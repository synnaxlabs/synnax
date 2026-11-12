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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var errCommit = errors.New("commit failed")

// failingKV fails each commit while failed is true.
type failingKV struct {
	kv.DB
	failed bool
}

func (d *failingKV) OpenTx() kv.Tx { return failingTx{Tx: d.DB.OpenTx(), db: d} }

type failingTx struct {
	kv.Tx
	db *failingKV
}

func (t failingTx) Commit(ctx context.Context, opts ...any) error {
	if t.db.failed {
		return errCommit
	}
	return t.Tx.Commit(ctx, opts...)
}

// holdingKV calls hold, when set, after each commit applies and before the commit
// returns.
type holdingKV struct {
	kv.DB
	hold func(context.Context)
}

func (d *holdingKV) OpenTx() kv.Tx { return holdingTx{Tx: d.DB.OpenTx(), db: d} }

type holdingTx struct {
	kv.Tx
	db *holdingKV
}

func (t holdingTx) Commit(ctx context.Context, opts ...any) error {
	if err := t.Tx.Commit(ctx, opts...); err != nil {
		return err
	}
	if t.db.hold != nil {
		t.db.hold(ctx)
	}
	return nil
}

var _ = Describe("Delivery", func() {
	var (
		db      *gorp.DB
		records *gorp.Table[int32, entry]
		create  = func(ctx context.Context, keys ...int32) func(gorp.Tx) error {
			return func(tx gorp.Tx) error {
				entries := make([]entry, len(keys))
				for i, key := range keys {
					entries[i] = entry{ID: key, Data: "data"}
				}
				return records.NewCreate().Entries(&entries).Exec(ctx, tx)
			}
		}
		created = func(key int32) gorp.Change {
			GinkgoHelper()
			return gorp.Change{
				Record: gorp.NewTarget("record", key),
				Kind:   gorp.ChangeCreate,
				Value: MustSucceed(msgpack.Codec.Encode(
					entry{ID: key, Data: "data"},
				)),
			}
		}
		collect = func() *[]gorp.Commit {
			var commits []gorp.Commit
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				commits = append(commits, c)
			}))
			return &commits
		}
		open = func(ctx context.Context, store kv.DB) {
			GinkgoHelper()
			db = DeferClose(gorp.Wrap(store, gorp.WithCodec(msgpack.Codec)))
			records = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB:   db,
				Name: "record",
			}))
		}
		// holdFirstCommit reopens the DB on a store that holds the first commit until
		// release closes. The hold is after the store applies the commit, which is
		// after the release of the write lock and before delivery. secondApplied closes
		// when the store applies the second commit, which has its number by then.
		holdFirstCommit = func(
			ctx context.Context,
			release <-chan struct{},
		) (firstApplied, secondApplied chan struct{}) {
			GinkgoHelper()
			firstApplied, secondApplied = make(chan struct{}), make(chan struct{})
			store := &holdingKV{DB: memkv.New()}
			open(ctx, store)
			var applied atomic.Int32
			store.hold = func(ctx context.Context) {
				switch applied.Add(1) {
				case 1:
					close(firstApplied)
					select {
					case <-release:
					case <-ctx.Done():
					}
				case 2:
					close(secondApplied)
				}
			}
			return firstApplied, secondApplied
		}
	)

	BeforeEach(func(ctx SpecContext) { open(ctx, memkv.New()) })

	It(
		"Should number each commit that changes a record, starting at 1",
		func(ctx SpecContext) {
			commits := collect()
			for key := int32(1); key <= 3; key++ {
				Expect(db.WithTx(ctx, create(ctx, key))).To(Succeed())
			}
			Expect(*commits).To(Equal([]gorp.Commit{
				{Seq: 1, Changes: gorp.Changes{created(1)}},
				{Seq: 2, Changes: gorp.Changes{created(2)}},
				{Seq: 3, Changes: gorp.Changes{created(3)}},
			}))
		},
	)

	DescribeTable(
		"Should give no number to a commit that changes no record",
		func(ctx SpecContext, write func(context.Context, gorp.Tx) error) {
			commits := collect()
			Expect(db.WithTx(ctx, create(ctx, 1))).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error { return write(ctx, tx) })).
				To(Succeed())
			Expect(db.WithTx(ctx, create(ctx, 2))).To(Succeed())
			Expect(*commits).To(Equal([]gorp.Commit{
				{Seq: 1, Changes: gorp.Changes{created(1)}},
				{Seq: 2, Changes: gorp.Changes{created(2)}},
			}))
		},
		Entry("an empty commit", func(context.Context, gorp.Tx) error { return nil }),
		Entry("a raw key-value write", func(ctx context.Context, tx gorp.Tx) error {
			return tx.Set(ctx, []byte("raw"), []byte("value"))
		}),
		Entry("an Add of no values", func(_ context.Context, tx gorp.Tx) error {
			gorp.NewAttachment[string]().Add(tx)
			return nil
		}),
		Entry(
			"a create and a delete of one record",
			func(ctx context.Context, tx gorp.Tx) error {
				if err := create(ctx, 3)(tx); err != nil {
					return err
				}
				return records.NewDelete().
					Where(gorp.MatchKeys[int32, entry](3)).
					Exec(ctx, tx)
			},
		),
	)

	It("Should give no number to a failed commit", func(ctx SpecContext) {
		store := &failingKV{DB: memkv.New()}
		open(ctx, store)
		commits := collect()
		Expect(db.WithTx(ctx, create(ctx, 1))).To(Succeed())
		store.failed = true
		Expect(db.WithTx(ctx, create(ctx, 2))).To(MatchError(errCommit))
		store.failed = false
		// A ticket that the failed commit took would block this commit.
		committed := make(chan error, 1)
		go func() { committed <- db.WithTx(ctx, create(ctx, 3)) }()
		Eventually(committed).Should(Receive(Succeed()))
		Expect(*commits).To(Equal([]gorp.Commit{
			{Seq: 1, Changes: gorp.Changes{created(1)}},
			{Seq: 2, Changes: gorp.Changes{created(3)}},
		}))
	})

	It("Should deliver a write through the DB as one commit", func(ctx SpecContext) {
		commits := collect()
		Expect(create(ctx, 1, 2)(db)).To(Succeed())
		Expect(*commits).To(Equal([]gorp.Commit{
			{Seq: 1, Changes: gorp.Changes{created(1), created(2)}},
		}))
	})

	It("Should deliver before Commit returns", func(ctx SpecContext) {
		var calls []string
		DeferCleanup(db.Observe().OnChange(func(context.Context, gorp.Commit) {
			calls = append(calls, "reader")
		}))
		tx := DeferClose(db.OpenTx())
		Expect(create(ctx, 1)(tx)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		calls = append(calls, "commit")
		Expect(calls).To(Equal([]string{"reader", "commit"}))
	})

	It(
		"Should deliver commits that race in the order they commit",
		func(ctx SpecContext) {
			release := make(chan struct{})
			firstApplied, secondApplied := holdFirstCommit(ctx, release)
			delivered := make(chan uint64, 2)
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				delivered <- c.Seq
			}))
			first := make(chan error, 1)
			go func() { first <- create(ctx, 1)(db) }()
			Eventually(firstApplied).Should(BeClosed())
			second := make(chan error, 1)
			go func() { second <- create(ctx, 2)(db) }()
			Eventually(secondApplied).Should(BeClosed())
			close(release)
			Eventually(first).Should(Receive(Succeed()))
			Eventually(second).Should(Receive(Succeed()))
			Eventually(delivered).Should(Receive(Equal(uint64(1))))
			Eventually(delivered).Should(Receive(Equal(uint64(2))))
		},
	)

	It(
		"Should deliver every number to a reader that connects while a commit waits",
		func(ctx SpecContext) {
			release := make(chan struct{})
			firstApplied, secondApplied := holdFirstCommit(ctx, release)
			disconnect := db.Observe().OnChange(func(context.Context, gorp.Commit) {})
			first := make(chan error, 1)
			go func() { first <- create(ctx, 1)(db) }()
			Eventually(firstApplied).Should(BeClosed())
			disconnect()
			second := make(chan error, 1)
			go func() { second <- create(ctx, 2)(db) }()
			Eventually(secondApplied).Should(BeClosed())
			delivered := make(chan uint64, 3)
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				delivered <- c.Seq
			}))
			close(release)
			Eventually(first).Should(Receive(Succeed()))
			Eventually(second).Should(Receive(Succeed()))
			Expect(create(ctx, 3)(db)).To(Succeed())
			Eventually(delivered).Should(Receive(Equal(uint64(1))))
			Eventually(delivered).Should(Receive(Equal(uint64(2))))
			Eventually(delivered).Should(Receive(Equal(uint64(3))))
		},
	)

	It("Should let a transaction commit while a reader runs", func(ctx SpecContext) {
		var (
			reading   = make(chan struct{})
			release   = make(chan struct{})
			delivered = make(chan uint64, 2)
		)
		DeferCleanup(db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
			if c.Seq == 1 {
				close(reading)
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			delivered <- c.Seq
		}))
		first := make(chan error, 1)
		go func() { first <- create(ctx, 1)(db) }()
		Eventually(reading).Should(BeClosed())
		second := make(chan error, 1)
		go func() { second <- create(ctx, 2)(db) }()
		Eventually(func(ctx SpecContext) bool {
			return MustSucceed(records.NewRetrieve().
				Where(gorp.MatchKeys[int32, entry](2)).
				Exists(ctx, db))
		}).WithContext(ctx).Should(BeTrue())
		close(release)
		Eventually(first).Should(Receive(Succeed()))
		Eventually(second).Should(Receive(Succeed()))
		Eventually(delivered).Should(Receive(Equal(uint64(1))))
		Eventually(delivered).Should(Receive(Equal(uint64(2))))
	})

	It("Should stop delivering to a reader once it disconnects", func(ctx SpecContext) {
		var stopped []uint64
		disconnect := db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
			stopped = append(stopped, c.Seq)
		})
		commits := collect()
		Expect(create(ctx, 1)(db)).To(Succeed())
		disconnect()
		// A second disconnect must leave the other reader connected.
		disconnect()
		Expect(create(ctx, 2)(db)).To(Succeed())
		Expect(stopped).To(Equal([]uint64{1}))
		Expect(*commits).To(HaveLen(2))
	})
})

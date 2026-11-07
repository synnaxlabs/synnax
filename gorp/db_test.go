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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var _ = Describe("DB", func() {
	Describe("WithTx", func() {
		It(
			"Should commit the transaction if the callback returns nil",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).
						Exec(ctx, tx)
				})).To(Succeed())
				var res entry
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](1)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res).To(Equal(entry{ID: 1, Data: "One"}))
			},
		)
		It(
			"Should not commit the transaction if the callback returns an error",
			func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return gorp.NewCreate[int32, entry]().Entry(&entry{ID: 1, Data: "One"}).
						Exec(ctx, tx)
				})).To(Succeed())
				Expect(
					db.WithTx(ctx, func(_ gorp.Tx) error { return query.ErrNotFound }),
				).
					To(MatchError(query.ErrNotFound))
				Expect(
					gorp.NewRetrieve[int32, entry]().Where(gorp.MatchKeys[int32, entry](2)).
						Exec(ctx, db),
				).To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("OnCommit", func() {
		It("Should run the hook after the transaction commits", func(ctx SpecContext) {
			tx := db.OpenTx()
			ran := false
			gorp.OnCommit(tx, func() { ran = true })
			Expect(ran).To(BeFalse())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(ran).To(BeTrue())
			Expect(tx.Close()).To(Succeed())
		})

		It("Should not run the hook when the transaction closes uncommitted", func() {
			tx := db.OpenTx()
			ran := false
			gorp.OnCommit(tx, func() { ran = true })
			Expect(tx.Close()).To(Succeed())
			Expect(ran).To(BeFalse())
		})

		It("Should run the hook once when the transaction closes after a commit",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				runs := 0
				gorp.OnCommit(tx, func() { runs++ })
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())
				Expect(runs).To(Equal(1))
			})

		It("Should run the hook at once for a DB used directly", func() {
			ran := false
			gorp.OnCommit(db, func() { ran = true })
			Expect(ran).To(BeTrue())
		})

		It("Should not run the hook when WithTx fails", func(ctx SpecContext) {
			ran := false
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				gorp.OnCommit(tx, func() { ran = true })
				return query.ErrNotFound
			})).To(MatchError(query.ErrNotFound))
			Expect(ran).To(BeFalse())
		})
	})

	Describe("OverrideTx", func() {
		It("Should return the override transaction if it is not nil", func() {
			tx := db.OpenTx()
			Expect(gorp.OverrideTx(db, tx)).To(Equal(tx))
			Expect(tx.Close()).To(Succeed())
		})
		It(
			"Should return the base transaction if the override transaction is nil",
			func() {
				Expect(gorp.OverrideTx(db, nil)).To(Equal(db))
			},
		)
	})

	Describe("Write lock", func() {
		var (
			lockDB *gorp.DB
			logs   *observer.ObservedLogs
		)
		BeforeEach(func() {
			var core zapcore.Core
			core, logs = observer.New(zap.WarnLevel)
			ins := alamos.New("gorp", alamos.WithLogger(MustSucceed(
				alamos.NewLogger(alamos.LoggerConfig{ZapLogger: zap.New(core)}),
			)))
			lockDB = DeferClose(gorp.Wrap(
				memkv.New(),
				gorp.WithCodec(msgpack.Codec),
				gorp.WithInstrumentation(ins),
			))
		})
		openAsync := func() <-chan gorp.Tx {
			opened := make(chan gorp.Tx, 1)
			go func() { opened <- lockDB.OpenTx() }()
			return opened
		}

		It(
			"Should give a waiting transaction the lock after the first one commits",
			func(ctx SpecContext) {
				first := lockDB.OpenTx()
				Expect(gorp.NewCreate[int32, entry]().
					Entry(&entry{ID: 1, Data: "One"}).
					Exec(ctx, first)).To(Succeed())
				opened := openAsync()
				Consistently(opened, 50*time.Millisecond).ShouldNot(Receive())
				Expect(first.Commit(ctx)).To(Succeed())
				var second gorp.Tx
				Eventually(opened).Should(Receive(&second))
				Expect(first.Close()).To(Succeed())
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](1)).
					Exists(ctx, second)).To(BeTrue())
				Expect(second.Close()).To(Succeed())
			},
		)

		It(
			"Should give a waiting transaction the lock after the first one closes",
			func() {
				first := lockDB.OpenTx()
				opened := openAsync()
				Consistently(opened, 50*time.Millisecond).ShouldNot(Receive())
				Expect(first.Close()).To(Succeed())
				var second gorp.Tx
				Eventually(opened).Should(Receive(&second))
				Expect(second.Close()).To(Succeed())
			},
		)

		It(
			"Should make a write through the DB wait for an open transaction",
			func(ctx SpecContext) {
				first := lockDB.OpenTx()
				written := make(chan error, 1)
				go func() {
					written <- gorp.NewCreate[int32, entry]().
						Entry(&entry{ID: 1, Data: "One"}).
						Exec(ctx, lockDB)
				}()
				Consistently(written, 50*time.Millisecond).ShouldNot(Receive())
				Expect(first.Close()).To(Succeed())
				Eventually(written).Should(Receive(Succeed()))
			},
		)

		It(
			"Should release the lock before observers run",
			func(ctx SpecContext) {
				observed := make(chan error, 1)
				disconnect := lockDB.OnChange(func(ctx context.Context, _ kv.TxReader) {
					tx := lockDB.OpenTx()
					observed <- tx.Close()
				})
				defer disconnect()
				Expect(gorp.NewCreate[int32, entry]().
					Entry(&entry{ID: 1, Data: "One"}).
					Exec(ctx, lockDB)).To(Succeed())
				Eventually(observed).Should(Receive(Succeed()))
			},
		)

		It("Should warn when a transaction waits long for the lock", func() {
			first := lockDB.OpenTx()
			opened := openAsync()
			Consistently(opened, 600*time.Millisecond).ShouldNot(Receive())
			Expect(first.Close()).To(Succeed())
			var second gorp.Tx
			Eventually(opened).Should(Receive(&second))
			Expect(second.Close()).To(Succeed())
			Expect(logs.FilterMessage(
				"gorp transaction waited long for the write lock",
			).Len()).To(Equal(1))
		})

		It("Should not warn when a transaction gets the lock at once", func() {
			Expect(lockDB.OpenTx().Close()).To(Succeed())
			Expect(logs.Len()).To(BeZero())
		})
	})

	Describe("KV", func() {
		It("Should return the underlying key-value store for the DB", func() {
			Expect(db.KV()).To(BeIdenticalTo(kvDB))
		})
	})
})

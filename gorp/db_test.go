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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
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

	Describe("Lock", func() {
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
			"Should release the lock before the readers of Observe run",
			func(ctx SpecContext) {
				observed := make(chan error, 1)
				disconnect := lockDB.Observe().OnChange(
					func(context.Context, gorp.Commit) {
						tx := lockDB.OpenTx()
						observed <- tx.Close()
					},
				)
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

		Describe("WithSnapshot", func() {
			var (
				table    *gorp.Table[int32, indexedEntry]
				nameIdx  *gorp.LookupIndex[int32, indexedEntry, string]
				scoreIdx *gorp.SortedIndex[int32, indexedEntry, int64]
				textIdx  *gorp.SearchIndex[int32, indexedEntry]
			)
			BeforeEach(func(ctx SpecContext) {
				nameIdx = gorp.NewLookupIndex(
					"name", func(e *indexedEntry) string { return e.Name },
				)
				scoreIdx = gorp.NewSortedIndex(
					"score", func(e *indexedEntry) int64 { return e.Score },
				)
				textIdx = newTextIndex()
				table = DeferClose(
					openIndexedTable(ctx, lockDB, nameIdx, scoreIdx, textIdx),
				)
				Expect(table.NewCreate().
					Entries(&[]indexedEntry{
						{ID: 1, Name: "alpha", Score: 10},
						{ID: 2, Name: "beta", Score: 20},
					}).
					Exec(ctx, lockDB)).To(Succeed())
			})
			readAsync := func(ctx context.Context) <-chan []int32 {
				read := make(chan []int32, 1)
				go func() {
					defer GinkgoRecover()
					Expect(lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
						var res []indexedEntry
						if err := table.NewRetrieve().
							Entries(&res).
							Exec(ctx, snap); err != nil {
							return err
						}
						read <- idsOf(res)
						return nil
					})).To(Succeed())
				}()
				return read
			}
			// holdSnapshot holds a snapshot on another goroutine until the returned
			// release runs, or until the spec ends.
			holdSnapshot := func(ctx context.Context) (func(), <-chan error) {
				GinkgoHelper()
				held, released := make(chan struct{}), make(chan struct{})
				done := make(chan error, 1)
				go func() {
					done <- lockDB.WithSnapshot(ctx, func(gorp.Snapshot) error {
						close(held)
						<-released
						return nil
					})
				}()
				release := sync.OnceFunc(func() { close(released) })
				DeferCleanup(release)
				Eventually(held).Should(BeClosed())
				return release, done
			}

			DescribeTable(
				"Should retrieve entries through a snapshot",
				func(
					ctx SpecContext,
					filter func() gorp.Filter[int32, indexedEntry],
				) {
					var res []indexedEntry
					Expect(lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
						return table.NewRetrieve().
							Where(filter()).
							Entries(&res).
							Exec(ctx, snap)
					})).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{2}))
				},
				Entry("by key", func() gorp.Filter[int32, indexedEntry] {
					return gorp.MatchKeys[int32, indexedEntry](2)
				}),
				Entry("by an index filter", func() gorp.Filter[int32, indexedEntry] {
					return nameIdx.Filter("beta")
				}),
				Entry("by a scan", func() gorp.Filter[int32, indexedEntry] {
					return gorp.Match(
						func(_ gorp.Context, e *indexedEntry) (bool, error) {
							return e.Name == "beta", nil
						},
					)
				}),
			)

			DescribeTable(
				"Should find keys through a snapshot",
				func(
					ctx SpecContext,
					find func(context.Context, gorp.Snapshot) ([]int32, error),
				) {
					var keys []int32
					Expect(
						lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
							keys, err = find(ctx, snap)
							return err
						}),
					).To(Succeed())
					Expect(keys).To(Equal([]int32{2}))
				},
				Entry(
					"with a lookup index",
					func(ctx context.Context, snap gorp.Snapshot) ([]int32, error) {
						return nameIdx.Get(ctx, snap, "beta")
					},
				),
				Entry(
					"with a sorted index",
					func(ctx context.Context, snap gorp.Snapshot) ([]int32, error) {
						return scoreIdx.Get(ctx, snap, 20)
					},
				),
				Entry(
					"with a search index",
					func(ctx context.Context, snap gorp.Snapshot) ([]int32, error) {
						res, err := textIdx.Search(
							ctx,
							snap,
							search.Query[int32]{Term: "beta"},
						)
						return res.Keys, err
					},
				),
			)

			It(
				"Should count and check entries through a snapshot",
				func(ctx SpecContext) {
					var (
						count  int
						exists bool
					)
					Expect(
						lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) (err error) {
							if count, err = table.NewRetrieve().
								Count(ctx, snap); err != nil {
								return err
							}
							exists, err = table.NewRetrieve().
								Where(gorp.MatchKeys[int32, indexedEntry](2)).
								Exists(ctx, snap)
							return err
						}),
					).To(Succeed())
					Expect(count).To(Equal(2))
					Expect(exists).To(BeTrue())
				},
			)

			It("Should return the error of the callback", func(ctx SpecContext) {
				Expect(lockDB.WithSnapshot(ctx, func(gorp.Snapshot) error {
					return query.ErrNotFound
				})).To(MatchError(query.ErrNotFound))
			})

			It(
				"Should pass a snapshot that is not a transaction",
				func(ctx SpecContext) {
					Expect(lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
						_, isTx := snap.(gorp.Tx)
						Expect(isTx).To(BeFalse())
						return nil
					})).To(Succeed())
				},
			)

			Context("A filter that retrieves other entries", func() {
				// matchNamed returns a filter whose keys a retrieve by name reads
				// through the Context of the filter.
				matchNamed := func(name string) gorp.Filter[int32, indexedEntry] {
					return gorp.MatchKeysFunc[int32, indexedEntry](
						func(qCtx gorp.Context) ([]int32, error) {
							var res []indexedEntry
							err := table.NewRetrieve().
								Where(nameIdx.Filter(name)).
								Entries(&res).
								Exec(qCtx, qCtx.Snapshot)
							return idsOf(res), err
						},
					)
				}

				It("Should read through the snapshot", func(ctx SpecContext) {
					var res []indexedEntry
					Expect(lockDB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
						return table.NewRetrieve().
							Where(matchNamed("beta")).
							Entries(&res).
							Exec(ctx, snap)
					})).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{2}))
				})

				It("Should see the writes of a transaction", func(ctx SpecContext) {
					tx := DeferClose(lockDB.OpenTx())
					Expect(table.NewCreate().
						Entry(&indexedEntry{ID: 3, Name: "gamma"}).
						Exec(ctx, tx)).To(Succeed())
					var res []indexedEntry
					Expect(table.NewRetrieve().
						Where(matchNamed("gamma")).
						Entries(&res).
						Exec(ctx, tx)).To(Succeed())
					Expect(idsOf(res)).To(Equal([]int32{3}))
				})
			})

			It("Should run two snapshots at the same time", func(ctx SpecContext) {
				release, first := holdSnapshot(ctx)
				Eventually(readAsync(ctx)).Should(Receive(Equal([]int32{1, 2})))
				release()
				Eventually(first).Should(Receive(Succeed()))
			})

			It(
				"Should make a transaction wait for an open snapshot",
				func(ctx SpecContext) {
					release, snapshot := holdSnapshot(ctx)
					opened := openAsync()
					Consistently(opened, 50*time.Millisecond).ShouldNot(Receive())
					release()
					Eventually(snapshot).Should(Receive(Succeed()))
					var tx gorp.Tx
					Eventually(opened).Should(Receive(&tx))
					Expect(tx.Close()).To(Succeed())
				},
			)

			It(
				"Should make a snapshot wait for an open transaction",
				func(ctx SpecContext) {
					tx := lockDB.OpenTx()
					Expect(table.NewCreate().
						Entry(&indexedEntry{ID: 3, Name: "gamma"}).
						Exec(ctx, tx)).To(Succeed())
					read := readAsync(ctx)
					Consistently(read, 50*time.Millisecond).ShouldNot(Receive())
					Expect(tx.Commit(ctx)).To(Succeed())
					Expect(tx.Close()).To(Succeed())
					Eventually(read).Should(Receive(Equal([]int32{1, 2, 3})))
				},
			)

			It(
				"Should warn when a snapshot waits long for the lock",
				func(ctx SpecContext) {
					tx := lockDB.OpenTx()
					done := make(chan error, 1)
					go func() {
						done <- lockDB.WithSnapshot(ctx, func(gorp.Snapshot) error {
							return nil
						})
					}()
					Consistently(done, 600*time.Millisecond).ShouldNot(Receive())
					Expect(tx.Close()).To(Succeed())
					Eventually(done).Should(Receive(Succeed()))
					Expect(logs.FilterMessage(
						"gorp snapshot waited long for the read lock",
					).Len()).To(Equal(1))
				},
			)

			It(
				"Should not warn when a snapshot gets the lock at once",
				func(ctx SpecContext) {
					Expect(lockDB.WithSnapshot(ctx, func(gorp.Snapshot) error {
						return nil
					})).To(Succeed())
					Expect(logs.Len()).To(BeZero())
				},
			)
		})
	})

	Describe("KV", func() {
		It("Should return the underlying key-value store for the DB", func() {
			Expect(db.KV()).To(BeIdenticalTo(kvDB))
		})
	})
})

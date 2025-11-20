// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("PebbleKV", func() {
	Describe("Operations", Ordered, func() {
		var (
			db     kv.DB
			dbPath string
		)

		BeforeAll(func() {
			dbPath = filepath.Join(os.TempDir(), "pebblekv-test")
			pdb, err := pebble.Open(dbPath, &pebble.Options{})
			Expect(err).ToNot(HaveOccurred())
			db = pebblekv.Wrap(pdb)
		})

		AfterAll(func() {
			Expect(db.Close()).To(Succeed())
			Expect(os.RemoveAll(dbPath)).To(Succeed())
		})

		It("Should handle basic CRUD operations", func() {
			key := []byte("key")
			value := []byte("value")
			Expect(db.Set(ctx, key, value)).To(Succeed())
			got, closer, err := db.Get(ctx, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			_, closer, err = db.Get(ctx, []byte("non-existent"))
			Expect(err).To(Equal(kv.NotFound))
			Expect(closer).To(BeNil())

			Expect(db.Delete(ctx, key)).To(Succeed())
			_, closer, err = db.Get(ctx, key)
			Expect(err).To(Equal(kv.NotFound))
			Expect(closer).To(BeNil())
		})

		It("Should handle transactions correctly", func() {
			tx := db.OpenTx()
			key := []byte("tx-key")
			value := []byte("tx-value")
			Expect(tx.Set(ctx, key, value)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			got, closer, err := db.Get(ctx, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			tx = db.OpenTx()
			rollbackKey := []byte("rollback-key")
			rollbackValue := []byte("rollback-value")
			Expect(tx.Set(ctx, rollbackKey, rollbackValue)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			_, closer, err = db.Get(ctx, rollbackKey)
			Expect(err).To(Equal(kv.NotFound))
			Expect(closer).To(BeNil())
		})

		It("Should not return a value if a transaction hasn't been committed", func() {
			tx := db.OpenTx()
			key := []byte("abc-tx-key")
			value := []byte("abc-tx-value")
			Expect(tx.Set(ctx, key, value)).To(Succeed())
			v, closer, err := db.Get(ctx, key)
			Expect(err).To(HaveOccurredAs(kv.NotFound))
			Expect(v).To(BeNil())
			Expect(closer).To(BeNil())
		})

		It("Should iterate over values correctly", func() {
			// Setup test data
			pairs := map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			}
			for k, v := range pairs {
				Expect(db.Set(ctx, []byte(k), []byte(v))).To(Succeed())
			}

			iter, err := db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte("a"),
				UpperBound: []byte("d"),
			})
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				Expect(iter.Close()).To(Succeed())
			}()

			expected := []struct {
				key   string
				value string
			}{
				{"a", "1"},
				{"b", "2"},
				{"c", "3"},
			}

			i := 0
			for iter.First(); iter.Valid(); iter.Next() {
				Expect(string(iter.Key())).To(Equal(expected[i].key))
				Expect(string(iter.Value())).To(Equal(expected[i].value))
				i++
			}
			Expect(i).To(Equal(len(expected)))
		})

		It("Should read transaction changes correctly", func() {
			tx := db.OpenTx()

			Expect(tx.Set(ctx, []byte("k1"), []byte("v1"))).To(Succeed())
			Expect(tx.Set(ctx, []byte("k2"), []byte("v2"))).To(Succeed())
			Expect(tx.Delete(ctx, []byte("k1"))).To(Succeed())

			reader := tx.NewReader()
			Expect(reader.Count()).To(Equal(3))

			changes := make([]kv.Change, 0, 3)
			for change, ok := reader.Next(ctx); ok; change, ok = reader.Next(ctx) {
				changes = append(changes, change)
			}

			Expect(changes).To(HaveLen(3))
			Expect(changes[0].Variant).To(Equal(change.Set))
			Expect(changes[0].Key).To(Equal([]byte("k1")))
			Expect(changes[0].Value).To(Equal([]byte("v1")))

			Expect(changes[1].Variant).To(Equal(change.Set))
			Expect(changes[1].Key).To(Equal([]byte("k2")))
			Expect(changes[1].Value).To(Equal([]byte("v2")))

			Expect(changes[2].Variant).To(Equal(change.Delete))
			Expect(changes[2].Key).To(Equal([]byte("k1")))

			Expect(tx.Close()).To(Succeed())
		})

		It("Should handle iterator bounds correctly", func() {
			for i := byte(0); i < 5; i++ {
				key := []byte{i}
				Expect(db.Set(ctx, key, []byte{i + 10})).To(Succeed())
			}

			iter, err := db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte{1},
				UpperBound: []byte{4},
			})
			Expect(err).ToNot(HaveOccurred())

			values := make([]byte, 0, 3)
			for iter.First(); iter.Valid(); iter.Next() {
				values = append(values, iter.Value()[0])
			}
			Expect(values).To(Equal([]byte{11, 12, 13}))

			values = make([]byte, 0, 3)
			for iter.Last(); iter.Valid(); iter.Prev() {
				values = append(values, iter.Value()[0])
			}
			Expect(values).To(Equal([]byte{13, 12, 11}))
			Expect(iter.Close()).To(Succeed())
		})

		It("Should respect NoSync write options", func() {
			key := []byte("nosync-key")
			value := []byte("nosync-value")

			Expect(db.Set(ctx, key, value, pebble.NoSync)).To(Succeed())

			got, closer, err := db.Get(ctx, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			tx := db.OpenTx()
			txKey := []byte("nosync-tx-key")
			txValue := []byte("nosync-tx-value")

			Expect(tx.Set(ctx, txKey, txValue)).To(Succeed())
			Expect(tx.Commit(ctx, pebble.NoSync)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			got, closer, err = db.Get(ctx, txKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(txValue))
			Expect(closer.Close()).To(Succeed())
		})

		It("Should handle OpenIterator errors", func() {
			// Invalid bounds should still create a valid iterator
			iter, err := db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte("z"),
				UpperBound: []byte("a"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(iter.Valid()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should handle transaction Get operations", func() {
			tx := db.OpenTx()
			key := []byte("tx-get-key")
			value := []byte("tx-get-value")

			_, closer, err := tx.Get(ctx, key)
			Expect(err).To(Equal(kv.NotFound))
			Expect(closer).To(BeNil())

			Expect(tx.Set(ctx, key, value)).To(Succeed())
			got, closer, err := tx.Get(ctx, key)
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			Expect(tx.Close()).To(Succeed())
		})

		It("Should report engine info", func() {
			report := db.(alamos.ReportProvider).Report()
			Expect(report["engine"]).To(Equal("pebble"))
		})

		It("Should handle db.Commit as no-op", func() {
			Expect(db.Commit(ctx)).To(Succeed())
		})

		It("Should immediately return false when opening a reader directly on the DB", func() {
			Expect(db.Set(ctx, []byte("reader-key"), []byte("reader-value"))).To(Succeed())
			reader := db.NewReader()
			Expect(reader).ToNot(BeNil())
			_, ok := reader.Next(ctx)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Observer", func() {
		var db kv.DB

		open := func(disableObserver bool) {
			path := filepath.Join(os.TempDir(), "pebblekv-observer-test")
			pdb := MustSucceed(pebble.Open(path, &pebble.Options{}))
			var opts []pebblekv.OpenOption
			if disableObserver {
				opts = append(opts, pebblekv.DisableObservation())
			}
			db = pebblekv.Wrap(pdb, opts...) // Default: observation enabled
			DeferCleanup(func() {
				Expect(db.Close()).To(Succeed())
				Expect(os.RemoveAll(path)).To(Succeed())
			})
		}

		Context("With observation enabled (default)", Ordered, func() {
			BeforeAll(func() {
				open(false)
			})

			It("Should notify observers when using Set", func() {
				notified := false
				var receivedChanges []kv.Change

				db.OnChange(func(ctx context.Context, reader kv.TxReader) {
					notified = true
					for change, ok := reader.Next(ctx); ok; change, ok = reader.Next(ctx) {
						receivedChanges = append(receivedChanges, change)
					}
				})

				key := []byte("observed-key")
				value := []byte("observed-value")
				Expect(db.Set(ctx, key, value)).To(Succeed())

				Eventually(func() bool { return notified }).Should(BeTrue())
				Expect(receivedChanges).To(HaveLen(1))
				Expect(receivedChanges[0].Variant).To(Equal(change.Set))
				Expect(receivedChanges[0].Key).To(Equal(key))
				Expect(receivedChanges[0].Value).To(Equal(value))
			})

			It("Should notify observers when committing a transaction", func() {
				notified := false
				var receivedChanges []kv.Change

				db.OnChange(func(ctx context.Context, reader kv.TxReader) {
					notified = true
					for change, ok := reader.Next(ctx); ok; change, ok = reader.Next(ctx) {
						receivedChanges = append(receivedChanges, change)
					}
				})

				tx := db.OpenTx()
				key1 := []byte("tx-key-1")
				value1 := []byte("tx-value-1")
				key2 := []byte("tx-key-2")
				value2 := []byte("tx-value-2")

				Expect(tx.Set(ctx, key1, value1)).To(Succeed())
				Expect(tx.Set(ctx, key2, value2)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				Eventually(func() bool { return notified }).Should(BeTrue())
				Expect(receivedChanges).To(HaveLen(2))
				Expect(receivedChanges[0].Key).To(Equal(key1))
				Expect(receivedChanges[0].Value).To(Equal(value1))
				Expect(receivedChanges[1].Key).To(Equal(key2))
				Expect(receivedChanges[1].Value).To(Equal(value2))
			})

			It("Should notify observers on Delete operations", func() {
				notified := false
				var receivedChanges []kv.Change

				key := []byte("delete-key")
				value := []byte("delete-value")
				Expect(db.Set(ctx, key, value)).To(Succeed())

				db.OnChange(func(ctx context.Context, reader kv.TxReader) {
					notified = true
					for change, ok := reader.Next(ctx); ok; change, ok = reader.Next(ctx) {
						receivedChanges = append(receivedChanges, change)
					}
				})

				Expect(db.Delete(ctx, key)).To(Succeed())

				Eventually(func() bool { return notified }).Should(BeTrue())
				Expect(receivedChanges).To(HaveLen(1))
				Expect(receivedChanges[0].Variant).To(Equal(change.Delete))
				Expect(receivedChanges[0].Key).To(Equal(key))
			})

			It("Should provide fresh readers to each subscriber", func() {
				subscriber1Called := false
				subscriber2Called := false

				db.OnChange(func(ctx context.Context, reader kv.TxReader) {
					subscriber1Called = true
					Expect(reader.Count()).To(BeNumerically(">", 0))
				})

				db.OnChange(func(ctx context.Context, reader kv.TxReader) {
					subscriber2Called = true
					Expect(reader.Count()).To(BeNumerically(">", 0))
				})

				key := []byte("multi-subscriber-key")
				value := []byte("multi-subscriber-value")
				Expect(db.Set(ctx, key, value)).To(Succeed())

				Eventually(func() bool { return subscriber1Called && subscriber2Called }).Should(BeTrue())
			})
		})

		Context("With observation disabled", Ordered, func() {
			BeforeAll(func() {
				open(false)
			})

			It("Should not panic when using Set without an observer", func() {
				key := []byte("no-observer-key")
				value := []byte("no-observer-value")
				Expect(db.Set(ctx, key, value)).To(Succeed())

				got, closer, err := db.Get(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(value))
				Expect(closer.Close()).To(Succeed())
			})

			It("Should not panic when committing transactions without an observer", func() {
				tx := db.OpenTx()
				key := []byte("no-observer-tx-key")
				value := []byte("no-observer-tx-value")

				Expect(tx.Set(ctx, key, value)).To(Succeed())
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				got, closer, err := db.Get(ctx, key)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(value))
				Expect(closer.Close()).To(Succeed())
			})

			It("Should handle Delete operations without an observer", func() {
				key := []byte("no-observer-delete-key")
				value := []byte("no-observer-delete-value")

				Expect(db.Set(ctx, key, value)).To(Succeed())
				Expect(db.Delete(ctx, key)).To(Succeed())

				_, closer, err := db.Get(ctx, key)
				Expect(err).To(Equal(kv.NotFound))
				Expect(closer).To(BeNil())
			})

			It("Should perform basic operations correctly without observers", func() {
				// Multiple operations to ensure no observer-related panics
				for i := 0; i < 10; i++ {
					key := []byte{byte(100 + i)}
					value := []byte{byte(200 + i)}
					Expect(db.Set(ctx, key, value)).To(Succeed())
				}

				// Verify all values
				for i := 0; i < 10; i++ {
					key := []byte{byte(100 + i)}
					expectedValue := []byte{byte(200 + i)}
					got, closer, err := db.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(got).To(Equal(expectedValue))
					Expect(closer.Close()).To(Succeed())
				}
			})

			It("Should handle batch operations in transactions without observers", func() {
				tx := db.OpenTx()

				// Batch set
				for i := 0; i < 5; i++ {
					key := []byte{byte(50 + i)}
					value := []byte{byte(150 + i)}
					Expect(tx.Set(ctx, key, value)).To(Succeed())
				}

				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(tx.Close()).To(Succeed())

				// Verify all values
				for i := 0; i < 5; i++ {
					key := []byte{byte(50 + i)}
					expectedValue := []byte{byte(150 + i)}
					got, closer, err := db.Get(ctx, key)
					Expect(err).ToNot(HaveOccurred())
					Expect(got).To(Equal(expectedValue))
					Expect(closer.Close()).To(Succeed())
				}
			})
		})
	})
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/cockroachdb/pebble/v2/vfs/errorfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/pebblekv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("PebbleKV", func() {
	Describe("Operations", Ordered, func() {
		var (
			db     kv.DB
			dbPath string
		)

		BeforeAll(func() {
			ShouldNotLeakGoroutines()
			dbPath = filepath.Join(os.TempDir(), "pebblekv-test")
			DeferCleanup(func() { Expect(os.RemoveAll(dbPath)).To(Succeed()) })
			logger := pebblekv.NewNoopLogger()
			pdb := MustSucceed(pebble.Open(dbPath, &pebble.Options{Logger: logger}))
			db = DeferClose(pebblekv.Wrap(pdb, logger))
		})

		It("Should handle basic CRUD operations", func(ctx SpecContext) {
			key := []byte("key")
			value := []byte("value")
			Expect(db.Set(ctx, key, value)).To(Succeed())
			got, closer := MustSucceed2(db.Get(ctx, key))
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			Expect(db.Get(ctx, []byte("non-existent"))).Error().
				To(MatchError(query.ErrNotFound))

			Expect(db.Delete(ctx, key)).To(Succeed())
			Expect(db.Get(ctx, key)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should handle transactions correctly", func(ctx SpecContext) {
			tx := db.OpenTx()
			key := []byte("tx-key")
			value := []byte("tx-value")
			Expect(tx.Set(ctx, key, value)).To(Succeed())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			got, closer := MustSucceed2(db.Get(ctx, key))
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			tx = db.OpenTx()
			rollbackKey := []byte("rollback-key")
			rollbackValue := []byte("rollback-value")
			Expect(tx.Set(ctx, rollbackKey, rollbackValue)).To(Succeed())
			Expect(tx.Close()).To(Succeed())

			Expect(db.Get(ctx, rollbackKey)).Error().To(MatchError(query.ErrNotFound))
		})

		It(
			"Should not return a value if a transaction hasn't been committed",
			func(ctx SpecContext) {
				tx := db.OpenTx()
				key := []byte("abc-tx-key")
				value := []byte("abc-tx-value")
				Expect(tx.Set(ctx, key, value)).To(Succeed())
				Expect(db.Get(ctx, key)).Error().To(MatchError(query.ErrNotFound))
			},
		)

		It("Should iterate over values correctly", func(ctx SpecContext) {
			// Setup test data
			pairs := map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			}
			for k, v := range pairs {
				Expect(db.Set(ctx, []byte(k), []byte(v))).To(Succeed())
			}

			iter := MustSucceed(db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte("a"),
				UpperBound: []byte("d"),
			}))
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

		It("Should handle iterator bounds correctly", func(ctx SpecContext) {
			for i := range byte(5) {
				key := []byte{i}
				Expect(db.Set(ctx, key, []byte{i + 10})).To(Succeed())
			}

			iter := MustSucceed(db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte{1},
				UpperBound: []byte{4},
			}))

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

		It("Should handle OpenIterator errors", func() {
			// Invalid bounds should still create a valid iterator
			iter := MustSucceed(db.OpenIterator(kv.IteratorOptions{
				LowerBound: []byte("z"),
				UpperBound: []byte("a"),
			}))
			Expect(iter.Valid()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should handle transaction Get operations", func(ctx SpecContext) {
			tx := db.OpenTx()
			key := []byte("tx-get-key")
			value := []byte("tx-get-value")

			Expect(tx.Get(ctx, key)).Error().To(MatchError(query.ErrNotFound))

			Expect(tx.Set(ctx, key, value)).To(Succeed())
			got, closer := MustSucceed2(tx.Get(ctx, key))
			Expect(got).To(Equal(value))
			Expect(closer.Close()).To(Succeed())

			Expect(tx.Close()).To(Succeed())
		})

		It("Should report engine info", func() {
			report := db.(alamos.ReportProvider).Report()
			Expect(report["engine"]).To(Equal("pebble"))
		})

		It("Should handle db.Commit as no-op", func(ctx SpecContext) {
			Expect(db.Commit(ctx)).To(Succeed())
		})
	})

	Describe("OnApplied", func() {
		It("Should call OnApplied after the commit applies", func(ctx SpecContext) {
			logger := pebblekv.NewNoopLogger()
			pdb := MustSucceed(pebble.Open(
				GinkgoT().TempDir(),
				&pebble.Options{Logger: logger},
			))
			db := DeferClose(pebblekv.Wrap(pdb, logger))
			key := []byte("applied-key")
			tx := db.OpenTx()
			Expect(tx.Set(ctx, key, []byte("applied-value"))).To(Succeed())
			applied := false
			Expect(tx.Commit(ctx, kv.OnApplied(func() {
				v, closer := MustSucceed2(db.Get(ctx, key))
				Expect(v).To(Equal([]byte("applied-value")))
				Expect(closer.Close()).To(Succeed())
				applied = true
			}))).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(applied).To(BeTrue())
		})
	})

	Describe("WAL sync", func() {
		It("Should log a fatal error when the WAL sync fails", func(ctx SpecContext) {
			walSync := &errorfs.Toggle{Injector: errorfs.InjectorFunc(failWALSync)}
			logger := &fatalRecorder{}
			pdb := MustSucceed(pebble.Open("", &pebble.Options{
				FS:     errorfs.Wrap(vfs.NewMem(), walSync),
				Logger: logger,
			}))
			db := pebblekv.Wrap(pdb, logger)
			walSync.On()
			Expect(
				db.Set(ctx, []byte("key"), []byte("value")),
			).To(MatchError(errorfs.ErrInjected))
			Expect(
				logger.messages(),
			).To(ConsistOf("failed to sync the WAL: injected error"))
			Expect(db.Close()).To(MatchError(errorfs.ErrInjected))
		})
	})
})

// failWALSync fails every sync of a Pebble WAL file.
func failWALSync(op errorfs.Op) error {
	if op.Kind == errorfs.OpFileSyncData && strings.HasSuffix(op.Path, ".log") {
		return errorfs.ErrInjected
	}
	return nil
}

// fatalRecorder is a pebble.Logger that records the messages of Fatalf instead of
// exiting.
type fatalRecorder struct {
	mu     sync.Mutex
	fatals []string
}

var _ pebble.Logger = (*fatalRecorder)(nil)

func (r *fatalRecorder) Infof(string, ...any) {}

func (r *fatalRecorder) Errorf(string, ...any) {}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *fatalRecorder) messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.fatals
}

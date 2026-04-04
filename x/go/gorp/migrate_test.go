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
	stdbinary "encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

type entryV1 struct {
	ID   int32  `msgpack:"id"`
	Data string `msgpack:"data"`
}

func (e entryV1) GorpKey() int32    { return e.ID }
func (e entryV1) SetOptions() []any { return nil }

const testNamespace = "test"

func migrateWithEntryV1(ctx context.Context, testDB *gorp.DB, migrations []migrate.Migration) error {
	return gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         testDB,
		Namespace:  testNamespace,
		Migrations: migrations,
	})
}

var _ = Describe("Migrate", func() {
	Describe("NewMigration", func() {
		It("Should provide a working gorp.Tx for read/write", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
			migration := gorp.NewMigration(
				"raw_transform",
				func(_ context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = "raw_migrated"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
		})
	})

	Describe("NewEntryMigration", func() {
		It("Should transform entries from one schema to another", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
			Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
			migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"add_suffix",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
			Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
		})

		It("Should apply the transform function to each entry", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
			migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"post_transform",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: "post:" + old.Data}, nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
		})
	})

	Describe("Version tracking", func() {
		It("Should store applied migration names", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			migration := gorp.NewMigration(
				"noop",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).To(Succeed())
			versionKey := []byte("gorp.migration." + testNamespace)
			b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
			Expect(closer.Close()).To(Succeed())
			Expect(string(b)).To(Equal(`["noop"]`))
		})

		It("Should skip already-completed migrations on re-run", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			executionCount := 0
			migration := gorp.NewMigration(
				"counted",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					executionCount++
					return nil
				},
			)
			migrations := []migrate.Migration{migration}
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			Expect(executionCount).To(Equal(1))
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			Expect(executionCount).To(Equal(1))
		})

		It("Should only run new migrations after partial completion", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var executed []string
			m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				executed = append(executed, "first")
				return nil
			})
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1})).To(Succeed())
			Expect(executed).To(Equal([]string{"first"}))
			m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				executed = append(executed, "second")
				return nil
			})
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).To(Succeed())
			Expect(executed).To(Equal([]string{"first", "second"}))
		})
	})

	Describe("Sequential execution", func() {
		It("Should chain two entry migrations sequentially", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
			m1 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"add_suffix",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
				},
			)
			m2 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"add_suffix_2",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
		})

		It("Should chain an entry migration with a raw migration", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
			m1 := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"typed_transform",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
				},
			)
			m2 := gorp.NewMigration("raw_update", func(
				ctx context.Context,
				tx gorp.Tx,
				ins alamos.Instrumentation,
			) error {
				r := gorp.WrapReader[int32, entryV1](tx)
				e := MustSucceed(r.Get(ctx, 1))
				e.Data = e.Data + "_raw"
				w := gorp.WrapWriter[int32, entryV1](tx)
				return w.Set(ctx, e)
			}, "typed_transform")
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
		})
	})

	Describe("Error handling", func() {
		It("Should not commit when a migration fails", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
			migration := gorp.NewMigration(
				"failing",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					return errors.New("migration error")
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).
				To(MatchError(ContainSubstring("failing")))
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
		})
	})

	Describe("Dependencies", func() {
		It("Should order migrations respecting declared dependencies", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var order []string
			m1 := gorp.NewMigration("alpha", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "alpha")
				return nil
			})
			m2 := gorp.NewMigration("beta", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "beta")
				return nil
			}, "alpha")
			m3 := gorp.NewMigration("gamma", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "gamma")
				return nil
			}, "beta")
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m3, m2, m1})).To(Succeed())
			Expect(order).To(Equal([]string{"alpha", "beta", "gamma"}))
		})

		It("Should treat already-applied dependencies as satisfied", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var order []string
			m1 := gorp.NewMigration("first", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "first")
				return nil
			})
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1})).To(Succeed())
			Expect(order).To(Equal([]string{"first"}))
			m2 := gorp.NewMigration("second", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "second")
				return nil
			}, "first")
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).To(Succeed())
			Expect(order).To(Equal([]string{"first", "second"}))
		})

		It("Should detect cyclic dependencies", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			m1 := gorp.NewMigration(
				"a",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				"b",
			)
			m2 := gorp.NewMigration(
				"b",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				"a",
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).
				To(MatchError(graph.ErrCyclicDependency))
		})

		It("Should return error for missing dependency", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			m1 := gorp.NewMigration(
				"orphan",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				"nonexistent",
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1})).
				To(MatchError(query.ErrNotFound))
		})

		It("Should work with migrations that do not declare dependencies", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var order []string
			m1 := gorp.NewMigration("plain_a", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "plain_a")
				return nil
			})
			m2 := gorp.NewMigration("plain_b", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "plain_b")
				return nil
			})
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m1, m2})).To(Succeed())
			Expect(order).To(Equal([]string{"plain_a", "plain_b"}))
		})

		It("Should preserve insertion order for migrations without dependencies "+
			"when mixed with dependency-declaring migrations", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var order []string
			m1 := gorp.NewMigration("no_dep", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "no_dep")
				return nil
			})
			m2 := gorp.NewMigration("has_dep", func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
				order = append(order, "has_dep")
				return nil
			}, "no_dep")
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{m2, m1})).To(Succeed())
			Expect(order).To(Equal([]string{"no_dep", "has_dep"}))
		})
	})

	Describe("Error context", func() {
		It("Should include entry key in transform error", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 42, Data: "bad"})).To(Succeed())
			migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"fail_transform",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{}, errors.New("transform broke")
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).
				To(MatchError(ContainSubstring("transform")))
		})

		It("Should include raw key in decode error", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			prefix := "gorp.entryV1"
			key := make([]byte, len(prefix)+4)
			copy(key, prefix)
			stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
			Expect(testDB.Set(ctx, key, []byte("not valid msgpack"))).To(Succeed())
			migration := gorp.NewEntryMigration[int32, int32, entryV1, entryV1](
				"fail_decode",
				func(ctx context.Context, old entryV1) (entryV1, error) {
					return old, nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []migrate.Migration{migration})).
				To(MatchError(ContainSubstring("decode")))
		})
	})
})

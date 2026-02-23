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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

type entryV1 struct {
	ID   int32  `msgpack:"id"`
	Data string `msgpack:"data"`
}

func (e entryV1) GorpKey() int32    { return e.ID }
func (e entryV1) SetOptions() []any { return nil }

type entryV2 struct {
	ID          int32  `msgpack:"id"`
	Data        string `msgpack:"data"`
	Description string `msgpack:"description"`
}

func (e entryV2) GorpKey() int32    { return e.ID }
func (e entryV2) SetOptions() []any { return nil }

type jsonEntry struct {
	ID   int32  `json:"id"`
	Data string `json:"data"`
}

func (e jsonEntry) GorpKey() int32    { return e.ID }
func (e jsonEntry) SetOptions() []any { return nil }

type jsonEntryCodec struct{}

func (jsonEntryCodec) Marshal(_ context.Context, e jsonEntry) ([]byte, error) {
	return json.Marshal(e)
}

func (jsonEntryCodec) Unmarshal(_ context.Context, data []byte) (jsonEntry, error) {
	var e jsonEntry
	err := json.Unmarshal(data, &e)
	return e, err
}

var _ gorp.Codec[jsonEntry] = jsonEntryCodec{}

var _ = Describe("Gorp", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	Describe("GorpRunner", func() {
		Describe("Basic migration execution", func() {
			It("Should run a single migration successfully", func() {
				executed := false
				runner := gorp.Migrator{
					Key: "test_migration_version",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = true
								return nil
							},
						},
					},
				}
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(BeTrue())
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{1}))
			})
			It("Should run multiple migrations in order", func() {
				var executionOrder []int
				runner := gorp.Migrator{
					Key: "test_migration_version_2",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 1)
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 2)
								return nil
							},
						},
						{
							Name: "third_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionOrder = append(executionOrder, 3)
								return nil
							},
						},
					},
				}
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionOrder).To(Equal([]int{1, 2, 3}))
				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_2")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})

			It("Should not run migrations that are already completed", func() {
				executionCount := 0
				runner := gorp.Migrator{
					Key: "test_migration_version_3",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(1))

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(1))
			})

			It("Should only run new migrations after partial completion", func() {
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_4",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "second")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "second"}))

				runner.Migrations = append(runner.Migrations, gorp.MigrationSpec{
					Name: "third_migration",
					Migrate: func(context.Context, gorp.Tx) error {
						executed = append(executed, "third")
						return nil
					},
				})

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "second", "third"}))

				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_4")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})

			It("Should handle empty migration list", func() {
				runner := gorp.Migrator{
					Key:        "test_migration_version_5",
					Migrations: []gorp.MigrationSpec{},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				Expect(db.Get(ctx, []byte("test_migration_version_5"))).Error().
					To(MatchError(query.ErrNotFound))
			})
		})

		Describe("Error handling", func() {
			It("Should return error when migration fails", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_6",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								return errors.New("test error")
							},
						},
					},
				}

				err := runner.Run(ctx, db)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("migration 1 (failing_migration) failed"))
				Expect(err.Error()).To(ContainSubstring("test error"))

				Expect(db.Get(ctx, []byte("test_migration_version_6"))).Error().
					To(MatchError(query.ErrNotFound))
			})

			It("Should fail when migration count exceeds 255", func() {
				migrations := make([]gorp.MigrationSpec, 256)
				for i := range migrations {
					migrations[i] = gorp.MigrationSpec{
						Name:    "migration",
						Migrate: func(context.Context, gorp.Tx) error { return nil },
					}
				}

				runner := gorp.Migrator{
					Key:        "test_migration_version_7",
					Migrations: migrations,
				}

				Expect(runner.Run(ctx, db)).Error().
					To(MatchError(gorp.ErrMigrationCountExceeded))
			})

			It("Should stop at first failing migration", func() {
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_8",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "second")
								return errors.New("failure")
							},
						},
						{
							Name: "third_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "third")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).Error().
					To(MatchError(ContainSubstring("migration 2 (failing_migration) failed")))
				Expect(executed).To(Equal([]string{"first", "second"}))

				Expect(db.Get(ctx, []byte("test_migration_version_8"))).Error().
					To(MatchError(query.ErrNotFound))
			})
		})

		Describe("Force flag", func() {
			It("Should rerun all migrations when Force is true", func() {
				executionCount := 0
				runner := gorp.Migrator{
					Key: "test_migration_version_9",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executionCount++
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(2))

				runner.Force = true
				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executionCount).To(Equal(4))

				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_9")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{2}))
			})

			It("Should run all migrations with Force even if some completed", func() {
				var executed []string
				runner := gorp.Migrator{
					Key: "test_migration_version_10",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								executed = append(executed, "first")
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first"}))

				runner.Migrations = append(runner.Migrations, gorp.MigrationSpec{
					Name: "second_migration",
					Migrate: func(context.Context, gorp.Tx) error {
						executed = append(executed, "second")
						return nil
					},
				})
				runner.Force = true

				Expect(runner.Run(ctx, db)).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "first", "second"}))
			})
		})

		Describe("Version tracking", func() {
			It("Should increment version after each successful migration", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_12",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								Expect(tx.Get(ctx, []byte("test_migration_version_12"))).
									Error().To(MatchError(query.ErrNotFound))
								return nil
							},
						},
						{
							Name: "second_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								version, closer := MustSucceed2(
									tx.Get(ctx, []byte("test_migration_version_12")),
								)
								Expect(closer.Close()).To(Succeed())
								Expect(version).To(Equal([]byte{1}))
								return nil
							},
						},
						{
							Name: "third_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								version, closer := MustSucceed2(
									tx.Get(ctx, []byte("test_migration_version_12")),
								)
								Expect(closer.Close()).To(Succeed())
								Expect(version).To(Equal([]byte{2}))
								return nil
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				version, closer := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_12")),
				)
				Expect(closer.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{3}))
			})
		})

		Describe("Transaction behavior", func() {
			It("Should rollback all changes when a migration fails", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_13",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								return tx.Set(ctx, []byte("test_key"), []byte("test_value"))
							},
						},
						{
							Name: "failing_migration",
							Migrate: func(context.Context, gorp.Tx) error {
								return errors.New("failure")
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(HaveOccurred())

				Expect(db.Get(ctx, []byte("test_key"))).Error().
					To(MatchError(query.ErrNotFound))

				Expect(db.Get(ctx, []byte("test_migration_version_13"))).Error().
					To(MatchError(query.ErrNotFound))
			})

			It("Should commit all changes when all migrations succeed", func() {
				runner := gorp.Migrator{
					Key: "test_migration_version_14",
					Migrations: []gorp.MigrationSpec{
						{
							Name: "first_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								return tx.Set(ctx, []byte("key1"), []byte("value1"))
							},
						},
						{
							Name: "second_migration",
							Migrate: func(ctx context.Context, tx gorp.Tx) error {
								return tx.Set(ctx, []byte("key2"), []byte("value2"))
							},
						},
					},
				}

				Expect(runner.Run(ctx, db)).To(Succeed())

				value1, closer1 := MustSucceed2(
					db.Get(ctx, []byte("key1")),
				)
				Expect(closer1.Close()).To(Succeed())
				Expect(value1).To(Equal([]byte("value1")))

				value2, closer2 := MustSucceed2(
					db.Get(ctx, []byte("key2")),
				)
				Expect(closer2.Close()).To(Succeed())
				Expect(value2).To(Equal([]byte("value2")))

				version, closer3 := MustSucceed2(
					db.Get(ctx, []byte("test_migration_version_14")),
				)
				Expect(closer3.Close()).To(Succeed())
				Expect(version).To(Equal([]byte{2}))
			})
		})
	})

	Describe("Codec", func() {
		It("Should use Codec.Marshal when a codec is provided to the table", func() {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB:    testDB,
				Codec: jsonEntryCodec{},
			}))
			e := jsonEntry{ID: 1, Data: "hello"}
			Expect(tbl.NewCreate().Entry(&e).Exec(ctx, testDB)).To(Succeed())
			var result jsonEntry
			Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should fall back to DB codec when no table codec is provided", func() {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entry](testDB)
			Expect(w.Set(ctx, entry{ID: 1, Data: "hello"})).To(Succeed())
			r := gorp.WrapReader[int32, entry](testDB)
			result := MustSucceed(r.Get(ctx, 1))
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("hello"))
		})

		It("Should use Codec.Unmarshal in Iterator.Value via table reader", func() {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			tbl := MustSucceed(gorp.OpenTable[int32, jsonEntry](ctx, gorp.TableConfig[jsonEntry]{
				DB:    testDB,
				Codec: jsonEntryCodec{},
			}))
			e := jsonEntry{ID: 1, Data: "iter"}
			Expect(tbl.NewCreate().Entry(&e).Exec(ctx, testDB)).To(Succeed())
			var result jsonEntry
			Expect(tbl.NewRetrieve().WhereKeys(1).Entry(&result).Exec(ctx, testDB)).To(Succeed())
			Expect(result.ID).To(Equal(int32(1)))
			Expect(result.Data).To(Equal("iter"))
		})
	})

	Describe("Migration Interface", func() {
		Describe("TypedMigration", func() {
			It("Should transform entries from one schema to another", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
				migration := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
			})

			It("Should call PostMigrateFunc after auto migration", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				migration := gorp.NewTypedMigration[entryV1, entryV1](
					"post_transform",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return old, nil
					},
					func(_ context.Context, new *entryV1, old entryV1) error {
						new.Data = "post:" + old.Data
						return nil
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
			})

			It("Should handle cross-type migration with manually seeded data", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				codec := &binary.MsgPackCodec{}
				prefix := "__gorp__//entryV2"
				for _, e := range []entryV1{{ID: 1, Data: "one"}, {ID: 2, Data: "two"}} {
					data := MustSucceed(codec.Encode(ctx, e))
					key := make([]byte, len(prefix)+4)
					copy(key, prefix)
					stdbinary.BigEndian.PutUint32(key[len(prefix):], uint32(e.ID))
					Expect(testDB.Set(ctx, key, data)).To(Succeed())
				}
				migration := gorp.NewTypedMigration[entryV1, entryV2](
					"v1_to_v2",
					func(_ context.Context, old entryV1) (entryV2, error) {
						return entryV2{
							ID:          old.ID,
							Data:        old.Data,
							Description: "migrated:" + old.Data,
						}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV2](ctx, gorp.TableConfig[entryV2]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV2](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Description).To(Equal("migrated:one"))
				Expect(MustSucceed(r.Get(ctx, 2)).Description).To(Equal("migrated:two"))
			})
		})

		Describe("RawMigration", func() {
			It("Should provide a working gorp.Tx for read/write", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
				migration := gorp.NewRawMigration(
					"raw_transform",
					func(ctx context.Context, tx gorp.Tx) error {
						r := gorp.WrapReader[int32, entryV1](tx)
						e := MustSucceed(r.Get(ctx, 1))
						e.Data = "raw_migrated"
						w := gorp.WrapWriter[int32, entryV1](tx)
						return w.Set(ctx, e)
					},
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
			})
		})

		Describe("Version tracking", func() {
			It("Should store version as uint16 big-endian", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewRawMigration(
					"noop",
					func(context.Context, gorp.Tx) error { return nil },
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}))
				versionKey := []byte("__gorp_migration__//entryV1")
				b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
				Expect(closer.Close()).To(Succeed())
				Expect(b).To(HaveLen(2))
				Expect(stdbinary.BigEndian.Uint16(b)).To(Equal(uint16(1)))
			})

			It("Should skip already-completed migrations on re-run", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				executionCount := 0
				migration := gorp.NewRawMigration(
					"counted",
					func(context.Context, gorp.Tx) error {
						executionCount++
						return nil
					},
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				Expect(executionCount).To(Equal(1))
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				Expect(executionCount).To(Equal(1))
			})

			It("Should only run new migrations after partial completion", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var executed []string
				m1 := gorp.NewRawMigration("first", func(context.Context, gorp.Tx) error {
					executed = append(executed, "first")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1},
				}))
				Expect(executed).To(Equal([]string{"first"}))
				m2 := gorp.NewRawMigration("second", func(context.Context, gorp.Tx) error {
					executed = append(executed, "second")
					return nil
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				Expect(executed).To(Equal([]string{"first", "second"}))
			})
		})

		Describe("Sequential execution / chaining", func() {
			It("Should chain two TypedMigrations sequentially", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
				m1 := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
					},
					nil,
				)
				m2 := gorp.NewTypedMigration[entryV1, entryV1](
					"add_suffix_2",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
					},
					nil,
				)
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
			})

			It("Should chain a TypedMigration with a RawMigration", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
				m1 := gorp.NewTypedMigration[entryV1, entryV1](
					"typed_transform",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
					},
					nil,
				)
				m2 := gorp.NewRawMigration("raw_update", func(ctx context.Context, tx gorp.Tx) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				})
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{m1, m2},
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
			})
		})

		Describe("Error handling", func() {
			It("Should not commit when a migration fails", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
				migration := gorp.NewRawMigration(
					"failing",
					func(context.Context, gorp.Tx) error {
						return errors.New("migration error")
					},
				)
				_, err := gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failing"))
				versionKey := []byte("__gorp_migration__//entryV1")
				Expect(testDB.Get(ctx, versionKey)).Error().To(MatchError(query.ErrNotFound))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
			})
		})

		Describe("Zero-migration case", func() {
			It("Should run key re-encoding only when no migrations are provided", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "no_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, gorp.TableConfig[entryV1]{
					DB: testDB,
				}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("no_migration"))
			})
		})

		Describe("Idempotent key migration", func() {
			It("Should run key re-encoding even when versioned migrations are at latest", func() {
				testDB := gorp.Wrap(memkv.New())
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := gorp.NewRawMigration(
					"noop",
					func(context.Context, gorp.Tx) error { return nil },
				)
				cfg := gorp.TableConfig[entryV1]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 5, Data: "post_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable[int32, entryV1](ctx, cfg))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
			})
		})
	})
})

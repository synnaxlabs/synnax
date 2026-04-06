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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Table", func() {
	var (
		db  *gorp.DB
		kvs kv.DB
	)
	BeforeEach(func() {
		kvs = memkv.New()
		db = gorp.Wrap(kvs)
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	Describe("OpenTable", func() {
		It("Should open a table on an empty database", func(ctx SpecContext) {
			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())
		})

		It("Should be idempotent when called multiple times", func(ctx SpecContext) {
			e := entry{ID: 1, Data: "data"}
			Expect(gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			table = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1).Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(e))
		})

		It("Should preserve entries after re-encoding", func(ctx SpecContext) {
			entries := []entry{
				{ID: 1, Data: "one"},
				{ID: 2, Data: "two"},
				{ID: 3, Data: "three"},
			}
			Expect(gorp.NewCreate[int32, entry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1, 2, 3).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entries))
		})

		It("Should work with uint64 keys", func(ctx SpecContext) {
			entries := []uint64Entry{
				{ID: 1, Data: "one"},
				{ID: 999999999, Data: "big"},
			}
			Expect(gorp.NewCreate[uint64, uint64Entry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[uint64Entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []uint64Entry
			Expect(gorp.NewRetrieve[uint64, uint64Entry]().
				WhereKeys(1, 999999999).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(2))
		})

		It("Should work with string keys", func(ctx SpecContext) {
			entries := []stringEntry{
				{ID: "alpha", Data: "first"},
				{ID: "beta", Data: "second"},
			}
			Expect(gorp.NewCreate[string, stringEntry]().
				Entries(&entries).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[stringEntry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []stringEntry
			Expect(gorp.NewRetrieve[string, stringEntry]().
				WhereKeys("alpha", "beta").
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entries))
		})
	})

	Describe("Zero-migration case", func() {
		It("Should run key re-encoding only when no migrations are provided", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entry](testDB)
			Expect(w.Set(ctx, entry{ID: 1, Data: "no_migration"})).To(Succeed())
			MustSucceed(gorp.OpenTable[int32, entry](ctx, gorp.TableConfig[entry]{
				DB: testDB,
			}))
			r := gorp.WrapReader[int32, entry](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("no_migration"))
		})
	})

	Describe("Idempotent key migration", func() {
		It("Should run key re-encoding even when versioned migrations are at latest", func(ctx SpecContext) {
			testDB := gorp.Wrap(memkv.New())
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			migration := gorp.NewMigration(
				"noop",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
			)
			cfg := gorp.TableConfig[entry]{
				DB:         testDB,
				Migrations: []migrate.Migration{migration},
			}
			MustSucceed(gorp.OpenTable[int32, entry](ctx, cfg))
			w := gorp.WrapWriter[int32, entry](testDB)
			Expect(w.Set(ctx, entry{ID: 5, Data: "post_migration"})).To(Succeed())
			MustSucceed(gorp.OpenTable[int32, entry](ctx, cfg))
			r := gorp.WrapReader[int32, entry](testDB)
			Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
		})
	})

	Describe("Migration dependency ordering", func() {
		It("Should run user migrations after normalize_keys", func(ctx SpecContext) {
			testKV := memkv.New()
			testDB := gorp.Wrap(testKV)
			defer func() { Expect(testDB.Close()).To(Succeed()) }()

			codec := msgpack.Codec
			typeName := types.Name[entry]()
			oldPrefix := MustSucceed(codec.Encode(ctx, typeName))
			encodedValue := MustSucceed(codec.Encode(ctx, entry{ID: 99, Data: "old"}))
			encodedKey := MustSucceed(codec.Encode(ctx, int32(99)))
			fullKey := make([]byte, len(oldPrefix)+len(encodedKey))
			copy(fullKey, oldPrefix)
			copy(fullKey[len(oldPrefix):], encodedKey)
			Expect(testKV.Set(ctx, fullKey, encodedValue)).To(Succeed())

			var sawEntry bool
			userMigration := gorp.NewMigration(
				"check_entries",
				func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
					var res entry
					err := gorp.NewRetrieve[int32, entry]().
						WhereKeys(99).Entry(&res).Exec(ctx, tx)
					if err == nil && res.Data == "old" {
						sawEntry = true
					}
					return nil
				},
			)

			MustSucceed(gorp.OpenTable[int32, entry](ctx, gorp.TableConfig[entry]{
				DB:         testDB,
				Migrations: []migrate.Migration{userMigration},
			}))
			Expect(sawEntry).To(BeTrue(),
				"user migration should see entries under new prefix, "+
					"meaning normalize_keys ran first")
		})
	})

	Describe("MigrateOldPrefixKeys", func() {
		writeOldFormatEntry := func(ctx context.Context, codec encoding.Codec, e entry) {
			typeName := types.Name[entry]()
			oldPrefix := MustSucceed(codec.Encode(ctx, typeName))
			encodedValue := MustSucceed(codec.Encode(ctx, e))
			// The key suffix doesn't matter for migration because
			// migrateOldPrefixKeys decodes the VALUE to reconstruct the
			// entry. We use an arbitrary suffix to differentiate keys.
			encodedKey := MustSucceed(codec.Encode(ctx, e.ID))
			fullKey := make([]byte, len(oldPrefix)+len(encodedKey))
			copy(fullKey, oldPrefix)
			copy(fullKey[len(oldPrefix):], encodedKey)
			Expect(kvs.Set(ctx, fullKey, encodedValue)).To(Succeed())
		}

		It("Should migrate entries stored under old codec-based prefix", func(ctx SpecContext) {
			codec := msgpack.Codec
			writeOldFormatEntry(ctx, codec, entry{ID: 42, Data: "old format"})

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(42).Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Data).To(Equal("old format"))
		})

		It("Should remove entries from the old prefix after migration", func(ctx SpecContext) {
			codec := msgpack.Codec
			writeOldFormatEntry(ctx, codec, entry{ID: 7, Data: "migrate me"})

			oldPrefix := MustSucceed(codec.Encode(ctx, types.Name[entry]()))
			iter := MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
			iter.First()
			Expect(iter.Valid()).To(BeTrue())
			Expect(iter.Close()).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			iter = MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
			iter.First()
			Expect(iter.Valid()).To(BeFalse())
			Expect(iter.Close()).To(Succeed())
		})

		It("Should handle migration with multiple old-format entries", func(ctx SpecContext) {
			codec := msgpack.Codec
			for i := range 5 {
				writeOldFormatEntry(ctx, codec, entry{ID: int32(i), Data: "old"})
			}

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(5))
		})

		It("Should not duplicate entries already stored under the new prefix", func(ctx SpecContext) {
			e := entry{ID: 10, Data: "already new"}
			Expect(gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(Equal(e))
		})

		It("Should migrate old-format entries while preserving new-format entries", func(ctx SpecContext) {
			codec := msgpack.Codec
			writeOldFormatEntry(ctx, codec, entry{ID: 1, Data: "old"})

			newEntry := entry{ID: 2, Data: "new"}
			Expect(gorp.NewCreate[int32, entry]().
				Entry(&newEntry).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{DB: db}))
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				WhereKeys(1, 2).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(HaveLen(2))
		})
	})

	Describe("Custom Codec", func() {
		var (
			table *gorp.Table[int32, entry]
		)
		BeforeEach(func(ctx SpecContext) {
			table = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[entry]{
				DB: db,
			}))
		})
		AfterEach(func() { Expect(table.Close()).To(Succeed()) })

		Describe("NewCreate + NewRetrieve", func() {
			It("Should create and retrieve an entry using the custom codec", func(ctx SpecContext) {
				e := entry{ID: 1, Data: "json-encoded"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				var res entry
				Expect(table.NewRetrieve().WhereKeys(1).Entry(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(Equal(e))
			})

			It("Should create and retrieve multiple entries", func(ctx SpecContext) {
				entries := []entry{
					{ID: 10, Data: "ten"},
					{ID: 20, Data: "twenty"},
					{ID: 30, Data: "thirty"},
				}
				Expect(table.NewCreate().Entries(&entries).Exec(ctx, db)).To(Succeed())
				var res []entry
				Expect(table.NewRetrieve().WhereKeys(10, 20, 30).Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(Equal(entries))
			})
		})

		Describe("NewUpdate", func() {
			It("Should update an entry using the custom codec", func(ctx SpecContext) {
				e := entry{ID: 50, Data: "before"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				Expect(table.NewUpdate().WhereKeys(50).Change(func(_ gorp.Context, e entry) entry {
					e.Data = "after"
					return e
				}).Exec(ctx, db)).To(Succeed())
				var res entry
				Expect(table.NewRetrieve().WhereKeys(50).Entry(&res).Exec(ctx, db)).To(Succeed())
				Expect(res.Data).To(Equal("after"))
			})
		})

		Describe("NewDelete", func() {
			It("Should delete an entry using the custom codec", func(ctx SpecContext) {
				e := entry{ID: 60, Data: "doomed"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				Expect(table.NewDelete().WhereKeys(60).Exec(ctx, db)).To(Succeed())
				Expect(table.NewRetrieve().WhereKeys(60).Exists(ctx, db)).To(BeFalse())
			})
		})

		Describe("OpenNexter", func() {
			It("Should iterate over entries using the custom codec", func(ctx SpecContext) {
				entries := []entry{
					{ID: 70, Data: "seventy"},
					{ID: 71, Data: "seventy-one"},
				}
				Expect(table.NewCreate().Entries(&entries).Exec(ctx, db)).To(Succeed())
				seq, closer := MustSucceed2(table.OpenNexter(ctx))
				var result []entry
				for e := range seq {
					result = append(result, e)
				}
				Expect(closer.Close()).To(Succeed())
				Expect(result).To(HaveLen(2))
			})
		})

		Describe("Observe", func() {
			It("Should observe changes using the custom codec", func(ctx SpecContext) {
				tx := db.OpenTx()
				e := entry{ID: 80, Data: "observed"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				var changes []change.Change[int32, entry]
				table.Observe().OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
					for ch := range r {
						changes = append(changes, ch)
					}
				})
				Expect(tx.Commit(ctx)).To(Succeed())
				Expect(changes).To(HaveLen(1))
				Expect(changes[0].Value).To(Equal(e))
				Expect(changes[0].Variant).To(Equal(change.VariantSet))
			})
		})
	})
})

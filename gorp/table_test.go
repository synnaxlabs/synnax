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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/types"
)

// normalizeEntryKeys is the key-normalization step an entry table pins in its chain.
func normalizeEntryKeys() gorp.Migration {
	return migrate.NormalizeKeys[int32, entry]("entry")
}

var _ = Describe("Table", func() {
	var (
		db  *gorp.DB
		kvs kv.DB
	)
	BeforeEach(func() {
		kvs = memkv.New()
		db = gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	Describe("OpenTable", func() {
		It("Should open a table on an empty database", func(ctx SpecContext) {
			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())
		})

		It("Should be idempotent when called multiple times", func(ctx SpecContext) {
			e := entry{ID: 1, Data: "data"}
			Expect(gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db)).To(Succeed())

			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())

			table = MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())

			var res entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](1)).
				Entry(&res).Exec(ctx, db)).To(Succeed())
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

			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())

			var res []entry
			Expect(gorp.NewRetrieve[int32, entry]().
				Where(gorp.MatchKeys[int32, entry](1, 2, 3)).
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

			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[uint64, uint64Entry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())

			var res []uint64Entry
			Expect(gorp.NewRetrieve[uint64, uint64Entry]().
				Where(gorp.MatchKeys[uint64, uint64Entry](1, 999999999)).
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

			table := MustSucceed(
				gorp.OpenTable(ctx, gorp.TableConfig[string, stringEntry]{DB: db}),
			)
			Expect(table.Close()).To(Succeed())

			var res []stringEntry
			Expect(gorp.NewRetrieve[string, stringEntry]().
				Where(gorp.MatchKeys[string, stringEntry]("alpha", "beta")).
				Entries(&res).Exec(ctx, db)).To(Succeed())
			Expect(res).To(Equal(entries))
		})
	})

	Describe("NewRetrieve", func() {
		It(
			"Should keep concurrent WherePrefix queries on their own prefixes",
			func(ctx SpecContext) {
				entries := make([]entry, 4)
				for i := range entries {
					entries[i] = entry{ID: int32(i), Data: "data"}
				}
				Expect(gorp.NewCreate[int32, entry]().
					Entries(&entries).Exec(ctx, db)).To(Succeed())
				table := MustOpen(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{DB: db}),
				)
				var wg sync.WaitGroup
				for _, e := range entries {
					wg.Go(func() {
						defer GinkgoRecover()
						for range 100 {
							var res []entry
							Expect(table.NewRetrieve().
								WherePrefix([]byte{0, 0, 0, byte(e.ID)}).
								Entries(&res).Exec(ctx, db)).To(Succeed())
							Expect(res).To(Equal([]entry{e}))
						}
					})
				}
				wg.Wait()
			},
		)
	})

	Describe("Zero-migration case", func() {
		It(
			"Should preserve stored entries when no migrations are provided",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entry](testDB)
				Expect(w.Set(ctx, entry{ID: 1, Data: "no_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
					DB: testDB,
				}))
				r := gorp.WrapReader[int32, entry](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("no_migration"))
			},
		)
	})

	Describe("Idempotent migration", func() {
		It(
			"Should preserve entries written after the chain is fully applied",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				migration := migrate.New(
					"noop",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				)
				cfg := gorp.TableConfig[int32, entry]{
					DB:         testDB,
					Migrations: []gorp.Migration{migration},
				}
				MustSucceed(gorp.OpenTable(ctx, cfg))
				w := gorp.WrapWriter[int32, entry](testDB)
				Expect(w.Set(ctx, entry{ID: 5, Data: "post_migration"})).To(Succeed())
				MustSucceed(gorp.OpenTable(ctx, cfg))
				r := gorp.WrapReader[int32, entry](testDB)
				Expect(MustSucceed(r.Get(ctx, 5)).Data).To(Equal("post_migration"))
			},
		)
	})

	Describe("Migration ordering", func() {
		It("Should run chain migrations in order", func(ctx SpecContext) {
			testKV := memkv.New()
			testDB := gorp.Wrap(testKV, gorp.WithCodec(msgpack.Codec))
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
			userMigration := migrate.New(
				"check_entries",
				func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
					var res entry
					err := gorp.NewRetrieve[int32, entry]().
						Where(gorp.MatchKeys[int32, entry](99)).
						Entry(&res).Exec(ctx, tx)
					if err == nil && res.Data == "old" {
						sawEntry = true
					}
					return nil
				},
			)

			MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB:         testDB,
				Migrations: []gorp.Migration{normalizeEntryKeys(), userMigration},
			}))
			Expect(sawEntry).To(BeTrue(),
				"user migration should see entries under new prefix, "+
					"meaning normalize_keys ran first")
		})
	})

	Describe("Custom Codec", func() {
		var table *gorp.Table[int32, entry]
		BeforeEach(func(ctx SpecContext) {
			table = MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
				DB: db,
			}))
		})

		Describe("NewCreate + NewRetrieve", func() {
			It(
				"Should create and retrieve an entry using the custom codec",
				func(ctx SpecContext) {
					e := entry{ID: 1, Data: "json-encoded"}
					Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
					var res entry
					Expect(
						table.NewRetrieve().
							Where(gorp.MatchKeys[int32, entry](1)).
							Entry(&res).
							Exec(ctx, db),
					).To(Succeed())
					Expect(res).To(Equal(e))
				},
			)

			It("Should create and retrieve multiple entries", func(ctx SpecContext) {
				entries := []entry{
					{ID: 10, Data: "ten"},
					{ID: 20, Data: "twenty"},
					{ID: 30, Data: "thirty"},
				}
				Expect(table.NewCreate().Entries(&entries).Exec(ctx, db)).To(Succeed())
				var res []entry
				Expect(
					table.NewRetrieve().
						Where(gorp.MatchKeys[int32, entry](10, 20, 30)).
						Entries(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res).To(Equal(entries))
			})
		})

		Describe("NewUpdate", func() {
			It("Should update an entry using the custom codec", func(ctx SpecContext) {
				e := entry{ID: 50, Data: "before"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				Expect(
					table.NewUpdate().
						Where(gorp.MatchKeys[int32, entry](50)).
						Change(func(_ gorp.Context, e entry) entry {
							e.Data = "after"
							return e
						}).
						Exec(ctx, db),
				).To(Succeed())
				var res entry
				Expect(
					table.NewRetrieve().
						Where(gorp.MatchKeys[int32, entry](50)).
						Entry(&res).
						Exec(ctx, db),
				).To(Succeed())
				Expect(res.Data).To(Equal("after"))
			})
		})

		Describe("NewDelete", func() {
			It("Should delete an entry using the custom codec", func(ctx SpecContext) {
				e := entry{ID: 60, Data: "doomed"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				Expect(
					table.NewDelete().
						Where(gorp.MatchKeys[int32, entry](60)).
						Exec(ctx, db),
				).To(Succeed())
				Expect(
					table.NewRetrieve().
						Where(gorp.MatchKeys[int32, entry](60)).
						Exists(ctx, db),
				).To(BeFalse())
			})
		})

		Describe("Reader", func() {
			It(
				"Should iterate over the table's entries using the custom codec",
				func(ctx SpecContext) {
					entries := []entry{
						{ID: 70, Data: "seventy"},
						{ID: 71, Data: "seventy-one"},
					}
					Expect(
						table.NewCreate().Entries(&entries).Exec(ctx, db),
					).To(Succeed())
					seq, closer := MustSucceed2(
						gorp.WrapReader[int32, entry](db).OpenNexter(ctx),
					)
					var result []entry
					for e := range seq {
						result = append(result, e)
					}
					Expect(closer.Close()).To(Succeed())
					Expect(result).To(HaveLen(2))
				},
			)
		})

		Describe("Observe", func() {
			It("Should observe changes using the custom codec", func(ctx SpecContext) {
				tx := db.OpenTx()
				e := entry{ID: 80, Data: "observed"}
				Expect(table.NewCreate().Entry(&e).Exec(ctx, tx)).To(Succeed())
				var changes []change.Change[int32, entry]
				table.Observe().
					OnChange(func(ctx context.Context, r gorp.TxReader[int32, entry]) {
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

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
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/gorp/testutil"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

// normalizeEntryKeys is the key-normalization step an entry table pins in its chain.
func normalizeEntryKeys() migrate.Migration {
	return gorp.NormalizeKeysMigration[int32, entry]("entry")
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
				migration := gorp.NewMigration(
					"noop",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
				)
				cfg := gorp.TableConfig[int32, entry]{
					DB:         testDB,
					Migrations: []migrate.Migration{migration},
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
			userMigration := gorp.NewMigration(
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
				Migrations: []migrate.Migration{normalizeEntryKeys(), userMigration},
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

		It(
			"Should migrate entries stored under old codec-based prefix",
			func(ctx SpecContext) {
				codec := msgpack.Codec
				writeOldFormatEntry(ctx, codec, entry{ID: 42, Data: "old format"})

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
						DB:         db,
						Migrations: []migrate.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](42)).
					Entry(&res).Exec(ctx, db)).To(Succeed())
				Expect(res.Data).To(Equal("old format"))
			},
		)

		It(
			"Should remove entries from the old prefix after migration",
			func(ctx SpecContext) {
				codec := msgpack.Codec
				writeOldFormatEntry(ctx, codec, entry{ID: 7, Data: "migrate me"})

				oldPrefix := MustSucceed(codec.Encode(ctx, types.Name[entry]()))
				iter := MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
				iter.First()
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Close()).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
						DB:         db,
						Migrations: []migrate.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				iter = MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
				iter.First()
				Expect(iter.Valid()).To(BeFalse())
				Expect(iter.Close()).To(Succeed())
			},
		)

		It(
			"Should handle migration with multiple old-format entries",
			func(ctx SpecContext) {
				codec := msgpack.Codec
				for i := range 5 {
					writeOldFormatEntry(ctx, codec, entry{ID: int32(i), Data: "old"})
				}

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
						DB:         db,
						Migrations: []migrate.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(5))
			},
		)

		It(
			"Should not duplicate entries already stored under the new prefix",
			func(ctx SpecContext) {
				e := entry{ID: 10, Data: "already new"}
				Expect(
					gorp.NewCreate[int32, entry]().Entry(&e).Exec(ctx, db),
				).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
						DB:         db,
						Migrations: []migrate.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0]).To(Equal(e))
			},
		)

		It(
			"Should migrate old-format entries while preserving new-format entries",
			func(ctx SpecContext) {
				codec := msgpack.Codec
				writeOldFormatEntry(ctx, codec, entry{ID: 1, Data: "old"})

				newEntry := entry{ID: 2, Data: "new"}
				Expect(gorp.NewCreate[int32, entry]().
					Entry(&newEntry).Exec(ctx, db)).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entry]{
						DB:         db,
						Migrations: []migrate.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entry
				Expect(gorp.NewRetrieve[int32, entry]().
					Where(gorp.MatchKeys[int32, entry](1, 2)).
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(2))
			},
		)
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

		Describe("OpenNexter", func() {
			It(
				"Should iterate over entries using the custom codec",
				func(ctx SpecContext) {
					entries := []entry{
						{ID: 70, Data: "seventy"},
						{ID: 71, Data: "seventy-one"},
					}
					Expect(
						table.NewCreate().Entries(&entries).Exec(ctx, db),
					).To(Succeed())
					seq, closer := MustSucceed2(table.OpenNexter(ctx))
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

// reKeyedEntry is the head shape of a table whose key type changed from the int32 its
// legacy rows store to a string.
type reKeyedEntry struct {
	ID   string
	Data string
}

func (r reKeyedEntry) GorpKey() string { return r.ID }

func (reKeyedEntry) SetOptions() []any { return nil }

func (reKeyedEntry) CustomTypeName() string { return "reKeyedEntry" }

// legacyReKeyedEntry is the frozen shape that wrote reKeyedEntry's legacy rows.
type legacyReKeyedEntry struct {
	ID   int32
	Data string
}

func (l legacyReKeyedEntry) GorpKey() int32 { return l.ID }

func (legacyReKeyedEntry) SetOptions() []any { return nil }

func (legacyReKeyedEntry) CustomTypeName() string { return "reKeyedEntry" }

// renamedEntry is the head shape of a table renamed from oldNameEntry.
type renamedEntry struct {
	ID   int32
	Data string
}

func (r renamedEntry) GorpKey() int32 { return r.ID }

func (renamedEntry) SetOptions() []any { return nil }

func (renamedEntry) CustomTypeName() string { return "renamedEntry" }

// oldNameEntry is the frozen shape that wrote renamedEntry's legacy rows under its
// previous type name.
type oldNameEntry struct {
	ID   int32
	Data string
}

func (o oldNameEntry) GorpKey() int32 { return o.ID }

func (oldNameEntry) SetOptions() []any { return nil }

func (oldNameEntry) CustomTypeName() string { return "oldNameEntry" }

// retypedNameEntry keeps its fields and its key, but reports a type name its legacy
// rows were never written under.
type retypedNameEntry struct {
	ID   int32
	Data string
}

func (r retypedNameEntry) GorpKey() int32 { return r.ID }

func (retypedNameEntry) SetOptions() []any { return nil }

func (retypedNameEntry) CustomTypeName() string { return "retypedName" }

var _ = Describe("NormalizeKeysMigration", func() {
	var (
		kvs kv.DB
		db  *gorp.DB
	)
	BeforeEach(func() {
		kvs = memkv.New()
		db = gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec))
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	writeLegacyRow := func(ctx context.Context, typeName string, key, value any) {
		GinkgoHelper()
		oldPrefix := MustSucceed(msgpack.Codec.Encode(ctx, typeName))
		encodedKey := MustSucceed(msgpack.Codec.Encode(ctx, key))
		fullKey := make([]byte, 0, len(oldPrefix)+len(encodedKey))
		fullKey = append(fullKey, oldPrefix...)
		fullKey = append(fullKey, encodedKey...)
		encodedValue := MustSucceed(msgpack.Codec.Encode(ctx, value))
		Expect(kvs.Set(ctx, fullKey, encodedValue)).To(Succeed())
	}

	It(
		"Should fail loudly when the pinned shape cannot decode a legacy row",
		func(ctx SpecContext) {
			writeLegacyRow(
				ctx,
				"reKeyedEntry",
				int32(7),
				legacyReKeyedEntry{ID: 7, Data: "old"},
			)
			Expect(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, reKeyedEntry]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NormalizeKeysMigration[string, reKeyedEntry](
							"reKeyedEntry",
						),
					},
				},
			)).Error().To(MatchError(
				ContainSubstring("normalize_keys: failed to decode entry"),
			))
		},
	)

	It(
		"Should normalize legacy rows with the frozen shape",
		func(ctx SpecContext) {
			writeLegacyRow(
				ctx,
				"reKeyedEntry",
				int32(7),
				legacyReKeyedEntry{ID: 7, Data: "old"},
			)
			lift := gorp.NewMigration(
				"lift_re_keyed",
				func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
					iter, err := gorp.WrapReader[int32, legacyReKeyedEntry](tx).
						OpenIterator(gorp.IterOptions{})
					if err != nil {
						return err
					}
					var lifted []legacyReKeyedEntry
					for iter.First(); iter.Valid(); iter.Next() {
						lifted = append(lifted, *iter.Value(ctx))
					}
					if err := iter.Close(); err != nil {
						return err
					}
					for _, l := range lifted {
						if err := gorp.WrapWriter[int32, legacyReKeyedEntry](tx).
							Delete(ctx, l.ID); err != nil {
							return err
						}
						if err := gorp.WrapWriter[string, reKeyedEntry](tx).Set(
							ctx,
							reKeyedEntry{ID: strconv.Itoa(int(l.ID)), Data: l.Data},
						); err != nil {
							return err
						}
					}
					return nil
				},
			)
			table := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, reKeyedEntry]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NormalizeKeysMigration[int32, legacyReKeyedEntry](
							"reKeyedEntry",
						),
						lift,
					},
				},
			))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			var res reKeyedEntry
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[string, reKeyedEntry]("7")).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Data).To(Equal("old"))
		},
	)

	It(
		"Should normalize rows of a renamed type as a chain step",
		func(ctx SpecContext) {
			writeLegacyRow(
				ctx,
				"oldNameEntry",
				int32(3),
				oldNameEntry{ID: 3, Data: "renamed"},
			)
			table := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, renamedEntry]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NormalizeKeysMigration[int32, oldNameEntry](
							"oldNameEntry",
						),
						gorp.NewEntryMigration(
							"lift_old_name_entries",
							func(_ context.Context, o oldNameEntry) (renamedEntry, error) {
								return renamedEntry(o), nil
							},
						),
					},
				},
			))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			var res renamedEntry
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[int32, renamedEntry](3)).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Data).To(Equal("renamed"))
		},
	)

	It(
		"Should read the legacy prefix from the name, not the frozen shape",
		func(ctx SpecContext) {
			writeLegacyRow(
				ctx,
				"originalName",
				int32(11),
				retypedNameEntry{ID: 11, Data: "renamed in place"},
			)
			table := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, retypedNameEntry]{
					DB: db,
					Migrations: []migrate.Migration{
						gorp.NormalizeKeysMigration[int32, retypedNameEntry](
							"originalName",
						),
					},
				},
			))
			defer func() { Expect(table.Close()).To(Succeed()) }()
			var res retypedNameEntry
			Expect(table.NewRetrieve().
				Where(gorp.MatchKeys[int32, retypedNameEntry](11)).
				Entry(&res).Exec(ctx, db)).To(Succeed())
			Expect(res.Data).To(Equal("renamed in place"))
		},
	)
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	"context"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/kv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

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

var _ = Describe("NormalizeKeys", func() {
	var (
		kvs kv.DB
		db  *gorp.DB
	)
	BeforeEach(func() {
		kvs = memkv.New()
		db = DeferClose(gorp.Wrap(kvs, gorp.WithCodec(msgpack.Codec)))
	})

	Describe("Legacy rows of an unchanged type", func() {
		writeOldFormatEntry := func(ctx context.Context, e entryV1) {
			GinkgoHelper()
			SetPreV54Row(ctx, kvs, "entryV1", e.ID, e)
		}
		normalizeEntryKeys := func() gorp.Migration {
			return migrate.NormalizeKeys[int32, entryV1]("entryV1")
		}

		It(
			"Should migrate entries stored under old codec-based prefix",
			func(ctx SpecContext) {
				writeOldFormatEntry(ctx, entryV1{ID: 42, Data: "old format"})

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entryV1]{
						DB:         db,
						Migrations: []gorp.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res entryV1
				Expect(gorp.NewRetrieve[int32, entryV1]().
					Where(gorp.MatchKeys[int32, entryV1](42)).
					Entry(&res).Exec(ctx, db)).To(Succeed())
				Expect(res.Data).To(Equal("old format"))
			},
		)

		It(
			"Should remove entries from the old prefix after migration",
			func(ctx SpecContext) {
				writeOldFormatEntry(ctx, entryV1{ID: 7, Data: "migrate me"})

				oldPrefix := MustSucceed(msgpack.Codec.Encode(ctx, "entryV1"))
				iter := MustSucceed(kvs.OpenIterator(kv.IterPrefix(oldPrefix)))
				iter.First()
				Expect(iter.Valid()).To(BeTrue())
				Expect(iter.Close()).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entryV1]{
						DB:         db,
						Migrations: []gorp.Migration{normalizeEntryKeys()},
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
				for i := range 5 {
					writeOldFormatEntry(ctx, entryV1{ID: int32(i), Data: "old"})
				}

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entryV1]{
						DB:         db,
						Migrations: []gorp.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entryV1
				Expect(gorp.NewRetrieve[int32, entryV1]().
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(5))
			},
		)

		It(
			"Should not duplicate entries already stored under the new prefix",
			func(ctx SpecContext) {
				e := entryV1{ID: 10, Data: "already new"}
				Expect(
					gorp.NewCreate[int32, entryV1]().Entry(&e).Exec(ctx, db),
				).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entryV1]{
						DB:         db,
						Migrations: []gorp.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entryV1
				Expect(gorp.NewRetrieve[int32, entryV1]().
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(1))
				Expect(res[0]).To(Equal(e))
			},
		)

		It(
			"Should migrate old-format entries while preserving new-format entries",
			func(ctx SpecContext) {
				writeOldFormatEntry(ctx, entryV1{ID: 1, Data: "old"})

				newEntry := entryV1{ID: 2, Data: "new"}
				Expect(gorp.NewCreate[int32, entryV1]().
					Entry(&newEntry).Exec(ctx, db)).To(Succeed())

				table := MustSucceed(
					gorp.OpenTable(ctx, gorp.TableConfig[int32, entryV1]{
						DB:         db,
						Migrations: []gorp.Migration{normalizeEntryKeys()},
					}),
				)
				Expect(table.Close()).To(Succeed())

				var res []entryV1
				Expect(gorp.NewRetrieve[int32, entryV1]().
					Where(gorp.MatchKeys[int32, entryV1](1, 2)).
					Entries(&res).Exec(ctx, db)).To(Succeed())
				Expect(res).To(HaveLen(2))
			},
		)
	})

	It(
		"Should fail loudly when the pinned shape cannot decode a legacy row",
		func(ctx SpecContext) {
			SetPreV54Row(
				ctx,
				kvs,
				"reKeyedEntry",
				int32(7),
				legacyReKeyedEntry{ID: 7, Data: "old"},
			)
			Expect(gorp.OpenTable(
				ctx,
				gorp.TableConfig[string, reKeyedEntry]{
					DB: db,
					Migrations: []gorp.Migration{
						migrate.NormalizeKeys[string, reKeyedEntry](
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
			SetPreV54Row(
				ctx,
				kvs,
				"reKeyedEntry",
				int32(7),
				legacyReKeyedEntry{ID: 7, Data: "old"},
			)
			lift := migrate.New(
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
					Migrations: []gorp.Migration{
						migrate.NormalizeKeys[int32, legacyReKeyedEntry](
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
			SetPreV54Row(
				ctx,
				kvs,
				"oldNameEntry",
				int32(3),
				oldNameEntry{ID: 3, Data: "renamed"},
			)
			table := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, renamedEntry]{
					DB: db,
					Migrations: []gorp.Migration{
						migrate.NormalizeKeys[int32, oldNameEntry](
							"oldNameEntry",
						),
						migrate.NewEntry(
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
			SetPreV54Row(
				ctx,
				kvs,
				"originalName",
				int32(11),
				retypedNameEntry{ID: 11, Data: "renamed in place"},
			)
			table := MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[int32, retypedNameEntry]{
					DB: db,
					Migrations: []gorp.Migration{
						migrate.NormalizeKeys[int32, retypedNameEntry](
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

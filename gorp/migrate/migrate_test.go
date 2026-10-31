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
	stdbinary "encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap/zapcore"
)

type entryV1 struct {
	ID   int32  `msgpack:"id"   json:"id"`
	Data string `msgpack:"data" json:"data"`
}

func (e entryV1) GorpKey() int32    { return e.ID }
func (e entryV1) SetOptions() []any { return nil }

const testNamespace = "test"

func migrateWithEntryV1(
	ctx context.Context,
	testDB *gorp.DB,
	migrations []gorp.Migration,
) error {
	return gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         testDB,
		Namespace:  testNamespace,
		Migrations: migrations,
	})
}

var _ = Describe("Migrate", func() {
	Describe("New", func() {
		It("Should provide a working gorp.Tx for read/write", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "raw"})).To(Succeed())
			migration := migrate.New(
				"raw_transform",
				func(_ context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = "raw_migrated"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				},
			)
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
			).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("raw_migrated"))
		})
	})

	Describe("NewEntry", func() {
		It(
			"Should transform entries from one schema to another",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
				migration := migrate.NewEntry(
					"add_suffix",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_migrated"}, nil
					},
				)
				Expect(
					migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
				).To(Succeed())
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one_migrated"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two_migrated"))
			},
		)

		It("Should apply the transform function to each entry", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
			migration := migrate.NewEntry(
				"post_transform",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: "post:" + old.Data}, nil
				},
			)
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
			).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("post:one"))
		})
	})

	Describe("NewInstrumentedEntry", func() {
		It("Should hand the transform the runner's instrumentation", func(
			ctx SpecContext,
		) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
			var received alamos.Instrumentation
			migration := migrate.NewInstrumentedEntry(
				"observe",
				func(
					_ context.Context,
					old entryV1,
					ins alamos.Instrumentation,
				) (entryV1, error) {
					received = ins
					return old, nil
				},
			)
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: alamos.New("runner"),
				DB:              testDB,
				Namespace:       testNamespace,
				Migrations:      []gorp.Migration{migration},
			})).To(Succeed())
			Expect(received.IsZero()).To(BeFalse())
		})
	})

	Describe("Sequential execution", func() {
		It("Should chain two entry migrations sequentially", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "chain"})).To(Succeed())
			m1 := migrate.NewEntry(
				"add_suffix",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_v2"}, nil
				},
			)
			m2 := migrate.NewEntry(
				"add_suffix_2",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{ID: old.ID, Data: old.Data + "_v3"}, nil
				},
			)
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{m1, m2}),
			).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("chain_v2_v3"))
		})

		It(
			"Should chain an entry migration with a raw migration",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "mixed"})).To(Succeed())
				m1 := migrate.NewEntry(
					"typed_transform",
					func(_ context.Context, old entryV1) (entryV1, error) {
						return entryV1{ID: old.ID, Data: old.Data + "_typed"}, nil
					},
				)
				m2 := migrate.New("raw_update", func(
					ctx context.Context,
					tx gorp.Tx,
					ins alamos.Instrumentation,
				) error {
					r := gorp.WrapReader[int32, entryV1](tx)
					e := MustSucceed(r.Get(ctx, 1))
					e.Data = e.Data + "_raw"
					w := gorp.WrapWriter[int32, entryV1](tx)
					return w.Set(ctx, e)
				})
				Expect(
					migrateWithEntryV1(ctx, testDB, []gorp.Migration{m1, m2}),
				).To(Succeed())
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("mixed_typed_raw"))
			},
		)
	})

	Describe("Codec", func() {
		It("Should re-encode all entries without changing data", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
			Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())
			Expect(w.Set(ctx, entryV1{ID: 3, Data: "three"})).To(Succeed())
			migration := migrate.Codec[int32, entryV1]("codec_v1")
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
			).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one"))
			Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two"))
			Expect(MustSucceed(r.Get(ctx, 3)).Data).To(Equal("three"))
		})

		It("Should work on an empty table", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			migration := migrate.Codec[int32, entryV1]("codec_empty")
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
			).To(Succeed())
		})

		It(
			"Should re-encode entries from an old codec to a new one",
			func(ctx SpecContext) {
				kvDB := memkv.New()
				defer func() { Expect(kvDB.Close()).To(Succeed()) }()

				// Write entries using JSON codec.
				jsonDB := gorp.Wrap(kvDB, gorp.WithCodec(json.Codec))
				w := gorp.WrapWriter[int32, entryV1](jsonDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "one"})).To(Succeed())
				Expect(w.Set(ctx, entryV1{ID: 2, Data: "two"})).To(Succeed())

				// Run CodecMigration with MessagePack primary + JSON fallback. This
				// reads entries via JSON fallback and writes them back as MessagePack.
				fallbackDB := gorp.Wrap(
					kvDB,
					gorp.WithCodec(
						encoding.NewDecodeFallbackCodec(msgpack.Codec, json.Codec),
					),
				)
				migration := migrate.Codec[int32, entryV1]("json_to_msgpack")
				Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
					DB:         fallbackDB,
					Namespace:  testNamespace,
					Migrations: []gorp.Migration{migration},
				})).To(Succeed())

				// Verify entries are now readable with plain MessagePack (no fallback
				// needed).
				msgpackDB := gorp.Wrap(kvDB, gorp.WithCodec(msgpack.Codec))
				r := gorp.WrapReader[int32, entryV1](msgpackDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("one"))
				Expect(MustSucceed(r.Get(ctx, 2)).Data).To(Equal("two"))
			},
		)

		It("Should be skipped on re-run", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "stable"})).To(Succeed())
			migration := migrate.Codec[int32, entryV1]("codec_idempotent")
			migrations := []gorp.Migration{migration}
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("stable"))
		})
	})

	DescribeTable("Progress logging",
		func(ctx SpecContext, migration gorp.Migration) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			for i := range 2000 {
				Expect(w.Set(ctx, entryV1{ID: int32(i), Data: "data"})).To(Succeed())
			}
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              testDB,
				Namespace:       testNamespace,
				Migrations:      []gorp.Migration{migration},
			})).To(Succeed())
			var counts []int64
			for _, e := range logs.FilterMessage("migration progress").All() {
				counts = append(counts, e.ContextMap()["entries"].(int64))
			}
			Expect(counts).To(Equal([]int64{1, 10, 100, 1000, 2000}))
		},
		Entry("NewEntry", migrate.NewEntry(
			"entry_progress",
			func(_ context.Context, old entryV1) (entryV1, error) { return old, nil },
		)),
		Entry("Codec", migrate.Codec[int32, entryV1]("codec_progress")),
	)

	Describe("Error context", func() {
		It("Should include entry key in transform error", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 42, Data: "bad"})).To(Succeed())
			migration := migrate.NewEntry(
				"fail_transform",
				func(_ context.Context, old entryV1) (entryV1, error) {
					return entryV1{}, errors.New("transform broke")
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration})).
				To(MatchError(ContainSubstring("transform")))
		})

		It("Should include raw key in decode error", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			prefix := "gorp.entryV1"
			key := make([]byte, len(prefix)+4)
			copy(key, prefix)
			stdbinary.BigEndian.PutUint32(key[len(prefix):], 1)
			Expect(testDB.Set(ctx, key, []byte("not valid msgpack"))).To(Succeed())
			migration := migrate.NewEntry(
				"fail_decode",
				func(ctx context.Context, old entryV1) (entryV1, error) {
					return old, nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration})).
				To(MatchError(ContainSubstring("decode")))
		})
	})
})

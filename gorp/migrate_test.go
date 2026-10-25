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
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	. "github.com/synnaxlabs/gorp/testutil"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
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
	Describe("Version tracking", func() {
		It("Should store applied migration names", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			migration := migrate.New(
				"noop",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error { return nil },
			)
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration}),
			).To(Succeed())
			versionKey := []byte("gorp.migration." + testNamespace)
			b, closer := MustSucceed2(testDB.Get(ctx, versionKey))
			Expect(closer.Close()).To(Succeed())
			Expect(string(b)).To(Equal(`["noop"]`))
		})

		It("Should skip already-completed migrations on re-run", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			executionCount := 0
			migration := migrate.New(
				"counted",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					executionCount++
					return nil
				},
			)
			migrations := []gorp.Migration{migration}
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			Expect(executionCount).To(Equal(1))
			Expect(migrateWithEntryV1(ctx, testDB, migrations)).To(Succeed())
			Expect(executionCount).To(Equal(1))
		})

		It(
			"Should only run new migrations after partial completion",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				var executed []string
				m1 := migrate.New(
					"first",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						executed = append(executed, "first")
						return nil
					},
				)
				Expect(
					migrateWithEntryV1(ctx, testDB, []gorp.Migration{m1}),
				).To(Succeed())
				Expect(executed).To(Equal([]string{"first"}))
				m2 := migrate.New(
					"second",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						executed = append(executed, "second")
						return nil
					},
				)
				Expect(
					migrateWithEntryV1(ctx, testDB, []gorp.Migration{m1, m2}),
				).To(Succeed())
				Expect(executed).To(Equal([]string{"first", "second"}))
			},
		)
	})

	Describe("Error handling", func() {
		It("Should not commit when a migration fails", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			w := gorp.WrapWriter[int32, entryV1](testDB)
			Expect(w.Set(ctx, entryV1{ID: 1, Data: "original"})).To(Succeed())
			migration := migrate.New(
				"failing",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					return errors.New("migration error")
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []gorp.Migration{migration})).
				To(MatchError(ContainSubstring("failing")))
			r := gorp.WrapReader[int32, entryV1](testDB)
			Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("original"))
		})

		It("Should not run the migrations after one that fails", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var ran []string
			record := func(key string, err error) gorp.Migration {
				return migrate.New(
					key,
					func(context.Context, gorp.Tx, alamos.Instrumentation) error {
						ran = append(ran, key)
						return err
					},
				)
			}
			Expect(migrateWithEntryV1(ctx, testDB, []gorp.Migration{
				record("a", nil),
				record("b", errors.New("boom")),
				record("c", nil),
			})).To(MatchError(ContainSubstring("migration b failed")))
			Expect(ran).To(Equal([]string{"a", "b"}))
		})
	})

	Describe("Ordering", func() {
		It("Should run migrations in list order", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			var order []string
			m1 := migrate.New(
				"alpha",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "alpha")
					return nil
				},
			)
			m2 := migrate.New(
				"beta",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "beta")
					return nil
				},
			)
			m3 := migrate.New(
				"gamma",
				func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
					order = append(order, "gamma")
					return nil
				},
			)
			Expect(
				migrateWithEntryV1(ctx, testDB, []gorp.Migration{m1, m2, m3}),
			).To(Succeed())
			Expect(order).To(Equal([]string{"alpha", "beta", "gamma"}))
		})

		It(
			"Should run an entry migration after an earlier raw migration",
			func(ctx SpecContext) {
				testDB := OpenGorpMsgpackDB()
				defer func() { Expect(testDB.Close()).To(Succeed()) }()
				w := gorp.WrapWriter[int32, entryV1](testDB)
				Expect(w.Set(ctx, entryV1{ID: 1, Data: "x"})).To(Succeed())
				var order []string
				base := migrate.New(
					"base",
					func(_ context.Context, _ gorp.Tx, _ alamos.Instrumentation) error {
						order = append(order, "base")
						return nil
					},
				)
				entry := migrate.NewEntry(
					"entry_after",
					func(_ context.Context, old entryV1) (entryV1, error) {
						order = append(order, "entry_after")
						return entryV1{ID: old.ID, Data: old.Data + "_done"}, nil
					},
				)
				Expect(
					migrateWithEntryV1(ctx, testDB, []gorp.Migration{base, entry}),
				).To(Succeed())
				Expect(order).To(Equal([]string{"base", "entry_after"}))
				r := gorp.WrapReader[int32, entryV1](testDB)
				Expect(MustSucceed(r.Get(ctx, 1)).Data).To(Equal("x_done"))
			},
		)
	})

	Describe("Validation", func() {
		It("Should reject duplicate keys before running any migration", func(
			ctx SpecContext,
		) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			ran := false
			m := migrate.New(
				"twice",
				func(context.Context, gorp.Tx, alamos.Instrumentation) error {
					ran = true
					return nil
				},
			)
			Expect(migrateWithEntryV1(ctx, testDB, []gorp.Migration{m, m})).
				To(MatchError(ContainSubstring(`duplicate migration name "twice"`)))
			Expect(ran).To(BeFalse())
		})

		It("Should succeed with no migrations", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			Expect(migrateWithEntryV1(ctx, testDB, nil)).To(Succeed())
		})
	})

	Describe("Logging", func() {
		noop := func(key string) gorp.Migration {
			return migrate.New(
				key,
				func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
			)
		}
		run := func(
			ctx context.Context,
			testDB *gorp.DB,
			ins alamos.Instrumentation,
			migrations ...gorp.Migration,
		) error {
			return gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              testDB,
				Namespace:       testNamespace,
				Migrations:      migrations,
			})
		}

		It("Should log already applied and pending migrations", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			Expect(run(ctx, testDB, alamos.Instrumentation{}, noop("a"))).To(Succeed())
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			Expect(run(ctx, testDB, ins, noop("a"), noop("b"))).To(Succeed())
			running := logs.FilterMessage("running migrations")
			Expect(running.Len()).To(Equal(1))
			fields := running.All()[0].ContextMap()
			Expect(fields["already_applied"]).To(ConsistOf("a"))
			Expect(fields["pending"]).To(ConsistOf("b"))
		})

		It("Should log each migration start and completion", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			Expect(run(ctx, testDB, ins, noop("a"), noop("b"))).To(Succeed())
			starts := logs.FilterMessage("running migration")
			Expect(starts.Len()).To(Equal(2))
			Expect(starts.All()[0].ContextMap()["migration"]).To(Equal("a"))
			Expect(starts.All()[1].ContextMap()["migration"]).To(Equal("b"))
			Expect(logs.FilterMessage("migration completed").Len()).To(Equal(2))
		})

		It("Should log when all migrations are already applied", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			Expect(run(ctx, testDB, alamos.Instrumentation{}, noop("a"))).To(Succeed())
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			Expect(run(ctx, testDB, ins, noop("a"))).To(Succeed())
			Expect(logs.FilterMessage("all migrations already applied").Len()).
				To(Equal(1))
		})

		It("Should log a migration failure", func(ctx SpecContext) {
			testDB := OpenGorpMsgpackDB()
			defer func() { Expect(testDB.Close()).To(Succeed()) }()
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			bad := migrate.New(
				"bad",
				func(context.Context, gorp.Tx, alamos.Instrumentation) error {
					return errors.New("boom")
				},
			)
			Expect(run(ctx, testDB, ins, bad)).
				To(MatchError(ContainSubstring("migration bad failed: boom")))
			failures := logs.FilterMessage("migration failed")
			Expect(failures.Len()).To(Equal(1))
			Expect(failures.All()[0].ContextMap()["migration"]).To(Equal("bad"))
		})
	})
})

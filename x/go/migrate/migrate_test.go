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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

type mockMigration struct {
	key  string
	deps set.Set[string]
	fn   func(ctx context.Context, ins alamos.Instrumentation) error
}

func (m *mockMigration) Key() string                  { return m.key }
func (m *mockMigration) Dependencies() set.Set[string] { return m.deps }
func (m *mockMigration) Run(ctx context.Context, ins alamos.Instrumentation) error {
	return m.fn(ctx, ins)
}

func newMock(key string, order *[]string, deps ...string) *mockMigration {
	return &mockMigration{
		key:  key,
		deps: set.New(deps...),
		fn: func(_ context.Context, _ alamos.Instrumentation) error {
			*order = append(*order, key)
			return nil
		},
	}
}

func noop(key string, deps ...string) *mockMigration {
	return &mockMigration{
		key:  key,
		deps: set.New(deps...),
		fn:   func(_ context.Context, _ alamos.Instrumentation) error { return nil },
	}
}

func failing(key string, deps ...string) *mockMigration {
	return &mockMigration{
		key:  key,
		deps: set.New(deps...),
		fn: func(_ context.Context, _ alamos.Instrumentation) error {
			return errors.New("boom")
		},
	}
}

func cfg(migrations ...migrate.Migration) migrate.Config {
	return migrate.Config{
		Migrations: migrations,
		Applied:    make(set.Set[string]),
	}
}

func cfgWithApplied(applied set.Set[string], migrations ...migrate.Migration) migrate.Config {
	return migrate.Config{
		Migrations: migrations,
		Applied:    applied,
	}
}

var _ = Describe("Migrate", func() {
	var ctx context.Context
	BeforeEach(func() { ctx = context.Background() })

	Describe("Migrate", func() {
		It("Should run a single migration", func() {
			var order []string
			m := newMock("a", &order)
			applied := MustSucceed(migrate.Migrate(ctx, cfg(m)))
			Expect(order).To(Equal([]string{"a"}))
			Expect(applied.Contains("a")).To(BeTrue())
		})

		It("Should run multiple migrations in order", func() {
			var order []string
			m1 := newMock("a", &order)
			m2 := newMock("b", &order)
			MustSucceed(migrate.Migrate(ctx, cfg(m1, m2)))
			Expect(order).To(Equal([]string{"a", "b"}))
		})

		It("Should skip already-applied migrations", func() {
			var order []string
			m := newMock("a", &order)
			applied := set.New("a")
			MustSucceed(migrate.Migrate(ctx, cfgWithApplied(applied, m)))
			Expect(order).To(BeEmpty())
		})

		It("Should only run new migrations when some are already applied", func() {
			var order []string
			m1 := newMock("a", &order)
			m2 := newMock("b", &order)
			applied := set.New("a")
			MustSucceed(migrate.Migrate(ctx, cfgWithApplied(applied, m1, m2)))
			Expect(order).To(Equal([]string{"b"}))
		})

		It("Should return the updated applied set including newly run migrations", func() {
			applied := MustSucceed(migrate.Migrate(ctx, cfg(noop("a"), noop("b"))))
			Expect(applied).To(HaveLen(2))
			Expect(applied.Contains("a")).To(BeTrue())
			Expect(applied.Contains("b")).To(BeTrue())
		})

		It("Should return the applied set unchanged when all migrations are applied", func() {
			applied := set.New("a", "b")
			result := MustSucceed(migrate.Migrate(ctx, cfgWithApplied(applied, noop("a"), noop("b"))))
			Expect(result).To(HaveLen(2))
		})

		It("Should handle an empty migrations list", func() {
			applied := MustSucceed(migrate.Migrate(ctx, cfg()))
			Expect(applied).To(HaveLen(0))
		})

		It("Should handle a nil applied set", func() {
			c := migrate.Config{Migrations: []migrate.Migration{noop("a")}}
			Expect(migrate.Migrate(ctx, c)).Error().ToNot(HaveOccurred())
		})

		It("Should return error on duplicate migration keys", func() {
			Expect(migrate.Migrate(ctx, cfg(noop("a"), noop("a")))).
				Error().To(MatchError(ContainSubstring("duplicate")))
		})

		It("Should not run any migrations after one fails", func() {
			var order []string
			m1 := newMock("a", &order)
			m2 := failing("b")
			m3 := newMock("c", &order)
			Expect(migrate.Migrate(ctx, cfg(m1, m2, m3))).
				Error().To(HaveOccurred())
			Expect(order).To(Equal([]string{"a"}))
		})

		It("Should wrap the failed migration key in the error message", func() {
			Expect(migrate.Migrate(ctx, cfg(failing("my_migration")))).
				Error().To(MatchError(ContainSubstring("my_migration")))
		})
	})

	Describe("Dependencies", func() {
		It("Should respect declared dependency ordering", func() {
			var order []string
			m1 := newMock("a", &order)
			m2 := newMock("b", &order, "a")
			m3 := newMock("c", &order, "b")
			MustSucceed(migrate.Migrate(ctx, cfg(m3, m2, m1)))
			Expect(order).To(Equal([]string{"a", "b", "c"}))
		})

		It("Should treat already-applied dependencies as satisfied", func() {
			var order []string
			m1 := newMock("a", &order)
			m2 := newMock("b", &order, "a")
			applied := set.New("a")
			MustSucceed(migrate.Migrate(ctx, cfgWithApplied(applied, m1, m2)))
			Expect(order).To(Equal([]string{"b"}))
		})

		It("Should detect cyclic dependencies", func() {
			m1 := noop("a", "b")
			m2 := noop("b", "a")
			Expect(migrate.Migrate(ctx, cfg(m1, m2))).
				Error().To(MatchError(graph.ErrCyclicDependency))
		})

		It("Should return error for missing dependency", func() {
			m := noop("a", "nonexistent")
			Expect(migrate.Migrate(ctx, cfg(m))).
				Error().To(MatchError(query.ErrNotFound))
		})

		It("Should handle diamond dependency graphs", func() {
			var order []string
			root := newMock("root", &order)
			left := newMock("left", &order, "root")
			right := newMock("right", &order, "root")
			tip := newMock("tip", &order, "left", "right")
			applied := MustSucceed(migrate.Migrate(ctx, cfg(tip, left, right, root)))
			Expect(applied).To(HaveLen(4))
			Expect(order[0]).To(Equal("root"))
			Expect(order[3]).To(Equal("tip"))
		})
	})

	Describe("WithAddedDeps", func() {
		It("Should add dependencies to a migration without mutating the original", func() {
			original := noop("a")
			Expect(original.Dependencies()).To(HaveLen(0))
			wrapped := migrate.WithAddedDeps(original, "x", "y")
			Expect(wrapped.Key()).To(Equal("a"))
			Expect(wrapped.Dependencies()).To(HaveLen(2))
			Expect(wrapped.Dependencies().Contains("x")).To(BeTrue())
			Expect(wrapped.Dependencies().Contains("y")).To(BeTrue())
			Expect(original.Dependencies()).To(HaveLen(0))
		})

		It("Should merge added deps with existing deps", func() {
			m := noop("a", "existing")
			wrapped := migrate.WithAddedDeps(m, "added")
			deps := wrapped.Dependencies()
			Expect(deps).To(HaveLen(2))
			Expect(deps.Contains("existing")).To(BeTrue())
			Expect(deps.Contains("added")).To(BeTrue())
		})
	})

	Describe("AllWithAddedDeps", func() {
		It("Should add the same deps to all migrations in the slice", func() {
			migrations := []migrate.Migration{noop("a"), noop("b"), noop("c")}
			wrapped := migrate.AllWithAddedDeps(migrations, "root")
			for _, m := range wrapped {
				Expect(m.Dependencies().Contains("root")).To(BeTrue())
			}
		})

		It("Should not mutate the original migrations", func() {
			m1 := noop("a")
			m2 := noop("b")
			originals := []migrate.Migration{m1, m2}
			migrate.AllWithAddedDeps(originals, "root")
			Expect(m1.Dependencies()).To(HaveLen(0))
			Expect(m2.Dependencies()).To(HaveLen(0))
		})
	})

	Describe("Logging", func() {
		It("Should log already applied and pending migrations", func() {
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			c := migrate.Config{
				Instrumentation: ins,
				Migrations:      []migrate.Migration{noop("a"), noop("b")},
				Applied:         set.New("a"),
			}
			MustSucceed(migrate.Migrate(ctx, c))
			running := logs.FilterMessage("running migrations")
			Expect(running.Len()).To(Equal(1))
			fields := running.All()[0].ContextMap()
			Expect(fields["already_applied"]).To(ConsistOf("a"))
			Expect(fields["pending"]).To(ConsistOf("b"))
		})

		It("Should log each migration start and completion", func() {
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			c := migrate.Config{
				Instrumentation: ins,
				Migrations:      []migrate.Migration{noop("a"), noop("b")},
				Applied:         make(set.Set[string]),
			}
			MustSucceed(migrate.Migrate(ctx, c))
			starts := logs.FilterMessage("running migration")
			Expect(starts.Len()).To(Equal(2))
			Expect(starts.All()[0].ContextMap()["migration"]).To(Equal("a"))
			Expect(starts.All()[1].ContextMap()["migration"]).To(Equal("b"))
			completions := logs.FilterMessage("migration completed")
			Expect(completions.Len()).To(Equal(2))
		})

		It("Should log when all migrations are already applied", func() {
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			c := migrate.Config{
				Instrumentation: ins,
				Migrations:      []migrate.Migration{noop("a")},
				Applied:         set.New("a"),
			}
			MustSucceed(migrate.Migrate(ctx, c))
			applied := logs.FilterMessage("all migrations already applied")
			Expect(applied.Len()).To(Equal(1))
		})

		It("Should log on migration failure", func() {
			ins, logs := ObservedInstrumentation(zapcore.InfoLevel)
			c := migrate.Config{
				Instrumentation: ins,
				Migrations:      []migrate.Migration{failing("bad")},
				Applied:         make(set.Set[string]),
			}
			Expect(migrate.Migrate(ctx, c)).Error().To(HaveOccurred())
			failures := logs.FilterMessage("migration failed")
			Expect(failures.Len()).To(Equal(1))
			Expect(failures.All()[0].ContextMap()["migration"]).To(Equal("bad"))
		})
	})
})

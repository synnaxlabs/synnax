// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/graph"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// Migration is a high level interface that represents an abstract data migration.
// A migration only runs once. It is identified by its key, which must be unique
// across all migrations that are run together.
type Migration interface {
	// Key returns a unique identifier for the migration.
	Key() string
	// Dependencies returns a list of migration keys that this migration depends on.
	Dependencies() set.Set[string]
	// Run executes the migration.
	Run(ctx context.Context, ins alamos.Instrumentation) error
}

// Config is the configuration for Migrate.
type Config struct {
	alamos.Instrumentation
	// Migrations is the full list of migrations to consider.
	Migrations []Migration
	// Applied is the set of migration keys that have already been run. Migrate
	// will skip these and only execute pending ones. A nil set is treated as empty.
	Applied set.Set[string]
}

// Migrate topologically sorts the provided migrations, skips any that are already
// applied, and runs the remaining ones in dependency order. It returns the updated
// applied set. All migration keys must be unique.
func Migrate(ctx context.Context, cfg Config) (set.Set[string], error) {
	if cfg.Applied == nil {
		cfg.Applied = make(set.Set[string])
	}
	byKey := make(map[string]Migration, len(cfg.Migrations))
	adj := make(map[string][]string)
	for _, m := range cfg.Migrations {
		k := m.Key()
		if _, dup := byKey[k]; dup {
			return nil, errors.Newf("duplicate migration name %q", k)
		}
		byKey[k] = m
		if cfg.Applied.Contains(k) {
			continue
		}
		adj[k] = nil
		for dep := range m.Dependencies() {
			if !cfg.Applied.Contains(dep) {
				adj[k] = append(adj[k], dep)
			}
		}
	}
	if len(adj) == 0 {
		cfg.L.Info("all migrations already applied", zap.Int("applied", len(cfg.Applied)))
		return cfg.Applied, nil
	}
	order, err := graph.TopoSort(adj)
	if err != nil {
		return nil, err
	}
	cfg.L.Info(
		"running migrations",
		zap.Strings("already_applied", cfg.Applied.Slice()),
		zap.Strings("pending", order),
	)
	for _, key := range order {
		cfg.L.Info("running migration", zap.String("migration", key))
		if err = byKey[key].Run(ctx, cfg.Instrumentation); err != nil {
			cfg.L.Error("migration failed", zap.String("migration", key), zap.Error(err))
			return nil, errors.Wrapf(err, "migration %s failed", key)
		}
		cfg.L.Info("migration completed", zap.String("migration", key))
		cfg.Applied.Add(key)
	}
	return cfg.Applied, nil
}

type addedDeps struct {
	addedDeps set.Set[string]
	Migration
}

func (a *addedDeps) Dependencies() set.Set[string] {
	deps := a.Migration.Dependencies().Copy()
	deps.Add(a.addedDeps.Slice()...)
	return deps
}

// Unwrap returns the innermost Migration if m wraps another (e.g. via
// WithAddedDeps). If m does not wrap anything, it returns m unchanged.
func Unwrap(m Migration) Migration {
	type wrapper interface{ Unwrap() Migration }
	for {
		w, ok := m.(wrapper)
		if !ok {
			return m
		}
		m = w.Unwrap()
	}
}

// Unwrap implements wrapper.
func (a *addedDeps) Unwrap() Migration { return a.Migration }

// WithAddedDeps wraps a Migration to declare additional dependencies beyond what
// it already declares. The original migration is not mutated.
func WithAddedDeps(base Migration, deps ...string) Migration {
	return &addedDeps{addedDeps: set.New(deps...), Migration: base}
}

// AllWithAddedDeps applies WithAddedDeps to every migration in the slice,
// returning a new slice. The original migrations are not mutated.
func AllWithAddedDeps(migrations []Migration, deps ...string) []Migration {
	result := make([]Migration, len(migrations))
	for i, m := range migrations {
		result[i] = WithAddedDeps(m, deps...)
	}
	return result
}

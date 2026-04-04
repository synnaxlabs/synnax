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

type Config struct {
	alamos.Instrumentation
	Migrations []Migration
	Applied    set.Set[string]
}

func Migrate(ctx context.Context, cfg Config) (set.Set[string], error) {
	migrationKeys := make(set.Set[string])
	for _, m := range cfg.Migrations {
		if migrationKeys.Contains(m.Key()) {
			return nil, errors.Newf("duplicate migration name %q", m.Key())
		}
		migrationKeys.Add(m.Key())
	}
	var notApplied = set.Difference(migrationKeys, cfg.Applied)
	cfg.L.Info(
		"running migrations",
		zap.Strings("not_applied", notApplied.ToSlice()),
		zap.Strings("applied", cfg.Applied.ToSlice()),
	)
	sorted, err := topoSort(cfg.Migrations, cfg.Applied)
	if err != nil {
		return nil, err
	}
	for _, m := range sorted {
		cfg.L.Info(
			"running migration",
			zap.String("migration", m.Key()),
		)
		if err = m.Run(ctx, cfg.Instrumentation); err != nil {
			cfg.L.Error(
				"migration failed",
				zap.String("migration", m.Key()),
				zap.Error(err),
			)
			return nil, errors.Wrapf(err, "migration %s failed", m.Key())
		}
		cfg.L.Info(
			"migration completed",
			zap.String("migration", m.Key()),
		)
		cfg.Applied.Add(m.Key())
	}
	return cfg.Applied, nil
}

// topoSort filters out already-applied migrations, then produces a valid
// execution order. Dependencies that are already applied are considered
// satisfied and do not need to appear in the pending set.
func topoSort(migrations []Migration, applied set.Set[string]) ([]Migration, error) {
	byName := make(map[string]Migration, len(migrations))
	for _, m := range migrations {
		if _, dup := byName[m.Key()]; dup {
			return nil, errors.Newf("duplicate migration name %q", m.Key())
		}
		byName[m.Key()] = m
	}

	var pending []Migration
	for _, m := range migrations {
		if !applied.Contains(m.Key()) {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}

	adj := make(map[string][]string, len(pending))
	for _, m := range pending {
		name := m.Key()
		adj[name] = nil
		for dep := range m.Dependencies() {
			if applied.Contains(dep) {
				continue
			}
			adj[name] = append(adj[name], dep)
		}
	}

	order, err := graph.TopoSort(adj)
	if err != nil {
		return nil, err
	}

	sorted := make([]Migration, len(order))
	for i, name := range order {
		sorted[i] = byName[name]
	}
	return sorted, nil
}

type addedDeps struct {
	addedDeps set.Set[string]
	Migration
}

func (a *addedDeps) Dependencies() set.Set[string] {
	deps := a.Migration.Dependencies().Copy()
	deps.Add(a.addedDeps.ToSlice()...)
	return deps
}

func WithAddedDeps(base Migration, deps set.Set[string]) Migration {
	return &addedDeps{addedDeps: deps, Migration: base}
}

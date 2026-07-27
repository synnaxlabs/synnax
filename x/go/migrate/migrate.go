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
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// Migration is a high level interface that represents an abstract data migration. A
// migration only runs once. It is identified by its key, which must be unique across
// all migrations that are run together.
type Migration interface {
	// Key returns a unique identifier for the migration.
	Key() string
	// Run executes the migration.
	Run(context.Context, alamos.Instrumentation) error
}

// Config is the configuration for Migrate.
type Config struct {
	alamos.Instrumentation
	// Migrations is the ordered list of migrations to consider.
	Migrations []Migration
	// Applied is the set of migration keys that have already been run. Migrate will
	// skip these and only execute pending ones. A nil set is treated as empty.
	Applied set.Set[string]
}

// Migrate runs the provided migrations in order, skipping any that are already applied.
// It returns the updated applied set. All migration keys must be unique.
func Migrate(ctx context.Context, cfg Config) (set.Set[string], error) {
	if cfg.Applied == nil {
		cfg.Applied = make(set.Set[string])
	}
	var (
		seen    = make(set.Set[string], len(cfg.Migrations))
		pending = make([]Migration, 0, len(cfg.Migrations))
	)
	for _, m := range cfg.Migrations {
		k := m.Key()
		if seen.Contains(k) {
			return nil, errors.Newf("duplicate migration name %q", k)
		}
		seen.Add(k)
		if !cfg.Applied.Contains(k) {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		cfg.L.Info("all migrations already applied", zap.Int("applied", len(cfg.Applied)))
		return cfg.Applied, nil
	}
	pendingKeys := make([]string, len(pending))
	for i, m := range pending {
		pendingKeys[i] = m.Key()
	}
	cfg.L.Info(
		"running migrations",
		zap.Strings("already_applied", cfg.Applied.Slice()),
		zap.Strings("pending", pendingKeys),
	)
	for _, m := range pending {
		key := m.Key()
		cfg.L.Info("running migration", zap.String("migration", key))
		if err := m.Run(ctx, cfg.Instrumentation); err != nil {
			cfg.L.Error("migration failed", zap.String("migration", key), zap.Error(err))
			return nil, errors.Wrapf(err, "migration %s failed", key)
		}
		cfg.L.Info("migration completed", zap.String("migration", key))
		cfg.Applied.Add(key)
	}
	return cfg.Applied, nil
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"encoding/json/v2"
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// Migration is a versioned change to the entries in a DB. A migration runs once:
// Migrate records its key after it succeeds and skips it from then on. The
// gorp/migrate package builds migrations.
type Migration interface {
	// Key returns the migration's identifier, unique among the migrations that run
	// together.
	Key() string
	// Run applies the migration within tx.
	Run(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
}

// MigrateConfig configures a call to Migrate.
type MigrateConfig struct {
	// Instrumentation is used for migration logging.
	alamos.Instrumentation
	// DB is the gorp DB to migrate.
	DB *DB
	// Namespace scopes the applied-migrations record key. Must be stable across runs so
	// previously applied migrations are not re-run.
	Namespace string
	// Migrations is the ordered set of migrations to consider. Migrations whose key is
	// already in the applied set are skipped.
	Migrations []Migration
}

// Migrate runs, in order, each migration in cfg that has not yet run in cfg.Namespace.
// All of them run in one transaction, which commits only if every one succeeds. Migrate
// returns an error without running any migration if two migrations share a key.
func Migrate(ctx context.Context, cfg MigrateConfig) (err error) {
	tx := cfg.DB.OpenTx()
	defer func() { err = errors.Combine(err, tx.Close()) }()
	applied, err := readAppliedMigrations(ctx, tx, cfg.Namespace)
	if err != nil {
		return err
	}
	pending, err := pendingMigrations(cfg.Migrations, applied)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		cfg.L.Info("all migrations already applied", zap.Int("applied", len(applied)))
	} else {
		cfg.L.Info(
			"running migrations",
			zap.Strings("already_applied", applied.Slice()),
			zap.Strings("pending", migrationKeys(pending)),
		)
	}
	for _, mig := range pending {
		k := mig.Key()
		cfg.L.Info("running migration", zap.String("migration", k))
		if err = mig.Run(ctx, tx, cfg.Instrumentation); err != nil {
			cfg.L.Error("migration failed", zap.String("migration", k), zap.Error(err))
			return errors.Wrapf(err, "migration %s failed", k)
		}
		cfg.L.Info("migration completed", zap.String("migration", k))
		applied.Add(k)
	}
	if err = writeAppliedMigrations(ctx, tx, cfg.Namespace, applied); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// pendingMigrations returns the migrations whose keys are not in applied, in order.
func pendingMigrations(
	migrations []Migration,
	applied set.Set[string],
) ([]Migration, error) {
	var (
		seen    = make(set.Set[string], len(migrations))
		pending = make([]Migration, 0, len(migrations))
	)
	for _, mig := range migrations {
		k := mig.Key()
		if seen.Contains(k) {
			return nil, errors.Newf("duplicate migration name %q", k)
		}
		seen.Add(k)
		if !applied.Contains(k) {
			pending = append(pending, mig)
		}
	}
	return pending, nil
}

func migrationKeys(migrations []Migration) []string {
	keys := make([]string, len(migrations))
	for i, mig := range migrations {
		keys[i] = mig.Key()
	}
	return keys
}

// readAppliedMigrations reads the set of applied migration names from the KV store.
// Names are stored as a JSON string array.
func readAppliedMigrations(
	ctx context.Context,
	tx Tx,
	namespace string,
) (applied set.Set[string], err error) {
	b, closer, err := tx.Get(ctx, key.Migrations(namespace))
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return make(set.Set[string]), nil
		}
		return nil, err
	}
	defer func() { err = errors.Combine(err, closer.Close()) }()
	var names []string
	if err := json.Unmarshal(b, &names); err != nil {
		return nil, err
	}
	return set.New(names...), nil
}

// writeAppliedMigrations persists the set of applied migration names as a sorted JSON
// string array.
func writeAppliedMigrations(
	ctx context.Context,
	tx Tx,
	namespace string,
	applied set.Set[string],
) error {
	names := applied.Slice()
	slices.Sort(names)
	b, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Set(ctx, key.Migrations(namespace), b)
}

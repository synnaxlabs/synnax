// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides utilities for handling data migrations between different
// versions of a schema or data structure.
package migrate

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/version"
	"go.uber.org/zap"
)

// Migratable represents a type that can be migrated between versions
type Migratable interface {
	GetVersion() version.Counter
}

// Context is provided for use by each migration.
type Context struct {
	// Context is the underlying std. context.
	context.Context
	// Instrumentation can be used for logging, tracing, etc.
	alamos.Instrumentation
}

// Migration represents a function that can migrate data from one version to another
type Migration[I, O Migratable] func(Context, I) (O, error)

// MigrationConfig holds the configuration for creating a migration
type MigrationConfig[I, O Migratable] struct {
	// Name is the name of the migration
	Name string
	// Migrate is the migration function.
	Migrate Migration[I, O]
}

// CreateMigration creates a new migration with logging
func CreateMigration[I, O Migratable](cfg MigrationConfig[I, O]) Migration[Migratable, Migratable] {
	return func(ctx Context, input Migratable) (Migratable, error) {
		var zero O
		out, err := cfg.Migrate(ctx, input.(I))
		if err != nil {
			ctx.L.Error("migration failed",
				zap.String("module", cfg.Name),
				zap.Int("input_version", int(input.GetVersion())),
				zap.Int("output_version", int(out.GetVersion())), zap.Error(err))
			return zero, errors.Wrapf(err, "migration %s failed for version %s to %s", cfg.Name, input.GetVersion(), out.GetVersion())
		}
		ctx.L.Info("migration completed",
			zap.String("module", cfg.Name),
			zap.Int("input_version", int(input.GetVersion())),
			zap.Int("output_version", int(out.GetVersion())),
		)
		return out, nil
	}
}

// Migrations is a map of version strings to migration functions
type Migrations map[version.Counter]Migration[Migratable, Migratable]

// LatestVersion returns the most recent (highest) version in the migrations.
func (m Migrations) LatestVersion() version.Counter {
	var latestV version.Counter
	for v := range m {
		if v.NewerThan(latestV) {
			latestV = v
		}
	}
	return latestV
}

// MigratorConfig holds the configuration for creating a migrator
type MigratorConfig[I, O Migratable] struct {
	alamos.Instrumentation
	Name       string
	Migrations Migrations
	Default    O
}

// NewMigrator creates a function that can migrate data from one version to another
func NewMigrator[I, O Migratable](cfg MigratorConfig[I, O]) func(I) O {
	if len(cfg.Migrations) == 0 {
		return func(v I) O {
			if v.GetVersion() != cfg.Default.GetVersion() {
				cfg.L.Warn("no migrations available, using default",
					zap.String("module", cfg.Name),
					zap.Int("version", int(v.GetVersion())),
					zap.Int("default_version", int(cfg.Default.GetVersion())),
				)
			}
			return cfg.Default
		}
	}

	var (
		applied bool
		migrate func(Migratable) (O, error)
		latestV = cfg.Migrations.LatestVersion()
	)
	migrate = func(old Migratable) (O, error) {
		v := old.GetVersion()
		if old.GetVersion().NewerThan(latestV) {
			if applied {
				cfg.L.Info("migration complete",
					zap.String("module", cfg.Name),
					zap.Int("version", int(v)),
				)
			} else {
				cfg.L.Info("version up to date",
					zap.String("module", cfg.Name),
					zap.Int("version", int(v)),
					zap.Int("target_version", int(cfg.Default.GetVersion())),
				)
			}
			return old.(O), nil
		}

		migration, ok := cfg.Migrations[v]
		if !ok {
			return cfg.Default, errors.Newf("no migration found for v %v", int(v))
		}

		next, err := migration(Context{Context: context.Background(), Instrumentation: cfg.Instrumentation}, old)
		if err != nil {
			return cfg.Default, err
		}

		applied = true
		return migrate(next)
	}

	return func(v I) O {
		result, err := migrate(v)
		if err != nil {
			cfg.L.Error("migration failed",
				zap.String("module", cfg.Name),
				zap.Error(err),
			)
			return cfg.Default
		}
		return result
	}
}

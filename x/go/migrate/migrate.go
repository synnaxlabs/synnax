// Package migrate provides utilities for handling data migrations between different versions
// of a schema or data structure.
package migrate

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/version"
	"go.uber.org/zap"
)

// Migratable represents a type that can be migrated between versions
type Migratable interface {
	GetVersion() version.Semantic
}

type Context struct {
	context.Context
	alamos.Instrumentation
}

// Migration represents a function that can migrate data from one version to another
type Migration[I, O Migratable] func(Context, I) (O, error)

// MigrationConfig holds the configuration for creating a migration
type MigrationConfig[I, O Migratable] struct {
	Name    string
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
				zap.String("input_version", string(input.GetVersion())),
				zap.String("output_version", string(out.GetVersion())), zap.Error(err))
			return zero, errors.Wrapf(err, "migration %s failed for version %s to %s", cfg.Name, input.GetVersion(), out.GetVersion())
		}
		ctx.L.Info("migration completed",
			zap.String("module", cfg.Name),
			zap.String("input_version", string(input.GetVersion())),
			zap.String("output_version", string(out.GetVersion())),
		)
		return out, nil
	}
}

// Migrations is a map of version strings to migration functions
type Migrations map[version.Semantic]Migration[Migratable, Migratable]

// MigratorConfig holds the configuration for creating a migrator
type MigratorConfig[I, O Migratable] struct {
	alamos.Instrumentation
	Name           string
	Migrations     Migrations
	Default        O
	DefaultVersion version.Semantic
}

// Migrator creates a function that can migrate data from one version to another
func Migrator[I, O Migratable](cfg MigratorConfig[I, O]) func(I) O {
	var latestV version.Semantic = "0.0.0"
	for v := range cfg.Migrations {
		if comp, err := version.CompareSemantic(v, latestV); err == nil && comp > 0 {
			latestV = v
		}
	}

	if len(cfg.Migrations) == 0 {
		return func(v I) O {
			if v.GetVersion() != cfg.Default.GetVersion() {
				cfg.L.Info("no migrations available, using default",
					zap.String("module", cfg.Name),
					zap.String("v", string(v.GetVersion())),
					zap.String("default_version", string(cfg.Default.GetVersion())),
				)
			}
			return cfg.Default
		}
	}

	var applied bool

	var migrate func(Migratable) (O, error)
	migrate = func(old Migratable) (O, error) {
		v := old.GetVersion()
		if v == "" {
			v = cfg.DefaultVersion
		}

		comp, err := version.CompareSemantic(v, latestV)
		if err != nil {
			return cfg.Default, fmt.Errorf("comparing versions: %w", err)
		}

		if comp > 0 {
			if applied {
				cfg.L.Info("migration complete",
					zap.String("module", cfg.Name),
					zap.String("v", string(v)),
				)
			} else {
				cfg.L.Info("v up to date",
					zap.String("module", cfg.Name),
					zap.String("v", string(v)),
					zap.String("target_version", string(cfg.Default.GetVersion())),
				)
			}
			return old.(O), nil
		}

		migration, ok := cfg.Migrations[v]
		if !ok {
			return cfg.Default, fmt.Errorf("no migration found for v %s", v)
		}

		new_, err := migration(Context{Context: context.Background(), Instrumentation: cfg.Instrumentation}, old)
		if err != nil {
			return cfg.Default, err
		}

		applied = true
		return migrate(new_)
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

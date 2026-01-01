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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

var ErrMigrationCountExceeded = errors.Newf(
	"migration count is greater than the maximum of 255",
)

// MigrationSpec defines a single migration that should be run with a transaction.
type MigrationSpec struct {
	// Name is a unique identifier for this migration (e.g., "name_validation")
	Name string
	// Migrate is the migration function to execute
	Migrate func(context.Context, Tx) error
}

// Migrator executes a series of migrations in order, tracking progress with
// incrementing versions. Migrations are run sequentially from current_version + 1 up to
// the latest version. The version starts at 0 (no migrations) and increments to
// len(Migrations).
type Migrator struct {
	// Key is the KV store key used to track migration version (e.g.,
	// "sy_channel_migration_version").
	Key string
	// Migrations is the ordered list of migrations to run.
	Migrations []MigrationSpec
	// Force, when true, will run all migrations, regardless of whether they have been
	// completed or not. This is useful for testing or debugging.
	Force bool
}

// Run executes all migrations that haven't been completed yet. Migrations run
// sequentially and the version is incremented after each successful migration.
//
// Example:
//
//	func (s *Service) migrate(ctx context.Context) error {
//	    return gorp.Migrator{
//	        Key: "sy_channel_migration_version",
//	        Migrations: []gorp.MigrationSpec{
//	            {Name: "name_validation", Migrate: s.migrateChannelNames},
//	            {Name: "future_migration", Migrate: s.migrateSomethingElse},
//	        },
//	    }.Run(ctx, s.DB)
//	}
func (r Migrator) Run(ctx context.Context, db *DB) error {
	return db.WithTx(ctx, func(tx Tx) error {
		migrationCount := len(r.Migrations)
		if migrationCount > 255 {
			return errors.Wrapf(
				ErrMigrationCountExceeded,
				"migration count is greater than the maximum of 255: %d",
				migrationCount,
			)
		}
		var currentVersion uint8
		if !r.Force {
			versionBytes, closer, err := tx.Get(ctx, []byte(r.Key))
			if err := errors.Skip(err, kv.NotFound); err != nil {
				return err
			}
			if closer != nil {
				if err := closer.Close(); err != nil {
					return err
				}
			}
			if len(versionBytes) > 0 {
				currentVersion = versionBytes[0]
			}
		}
		for i := currentVersion; i < uint8(migrationCount); i++ {
			migration := r.Migrations[i]
			newVersion := i + 1
			if err := migration.Migrate(ctx, tx); err != nil {
				return errors.Wrapf(
					err,
					"migration %d (%s) failed",
					newVersion,
					migration.Name,
				)
			}
			if err := tx.Set(ctx, []byte(r.Key), []byte{newVersion}); err != nil {
				return errors.Wrapf(
					err,
					"failed to migrate to version %d",
					newVersion,
				)
			}
		}
		return nil
	})
}

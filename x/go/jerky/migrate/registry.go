// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides runtime migration execution for jerky-managed types.
package migrate

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/x/gorp"
)

// TypedMigrator is implemented by generated migration code.
// It handles version-aware reading and migration using gorp.
type TypedMigrator interface {
	// TypeName returns the name of the type being migrated.
	TypeName() string
	// CurrentVersion returns the current (latest) schema version.
	CurrentVersion() int
	// MigrateAll migrates all records from the given version to the current version.
	// It uses gorp's iterator for proper prefix handling and writes back using gorp.
	MigrateAll(ctx context.Context, db *gorp.DB, fromVersion int) error
}

// Migration represents a single version migration.
type Migration struct {
	// FromVersion is the source version.
	FromVersion int
	// ToVersion is the target version.
	ToVersion int
	// Migrate transforms data from FromVersion to ToVersion.
	Migrate func(data []byte) ([]byte, error)
}

// Registry holds migration information for a single type.
type Registry struct {
	// TypeName is the name of the type being migrated.
	TypeName string
	// CurrentVersion is the latest schema version.
	CurrentVersion int
	// Migrations is a list of migrations in order.
	Migrations []Migration
}

// GetMigration returns the migration for a specific version transition.
func (r *Registry) GetMigration(fromVersion, toVersion int) *Migration {
	for i := range r.Migrations {
		if r.Migrations[i].FromVersion == fromVersion && r.Migrations[i].ToVersion == toVersion {
			return &r.Migrations[i]
		}
	}
	return nil
}

// NeedsMigration returns true if the data at the given version needs migration.
func (r *Registry) NeedsMigration(version int) bool {
	return version < r.CurrentVersion
}

// MigrateToLatest migrates data from fromVersion to the current version.
// It runs migrations sequentially (v0→v1, v1→v2, etc.).
func (r *Registry) MigrateToLatest(data []byte, fromVersion int) ([]byte, error) {
	for version := fromVersion; version < r.CurrentVersion; version++ {
		migration := r.GetMigration(version, version+1)
		if migration == nil {
			return nil, fmt.Errorf("no migration found for %s v%d -> v%d", r.TypeName, version, version+1)
		}

		var err error
		data, err = migration.Migrate(data)
		if err != nil {
			return nil, fmt.Errorf("migration %d -> %d failed: %w", version, version+1, err)
		}
	}
	return data, nil
}

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

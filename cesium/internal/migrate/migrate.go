// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"fmt"

	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/version"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/migrate"
	xversion "github.com/synnaxlabs/x/version"
)

// DBState is meta-data about a single-channel database that can be migrated. This data
// structure is passed into migration functions on bootup.
type DBState struct {
	// Channel is the channel specification for the DB.
	Channel core.Channel
	// FS is the file-system for that channel in the DB. This is not the
	// top level cesium directory, but the channel-specific directory itself.
	FS xfs.FS
	// ShouldIgnoreChannel can be set to true by the migration function if the channel
	// should be ignored on database startup.
	ShouldIgnoreChannel bool
}

// GetVersion implements migrate.Migratable.
func (d DBState) GetVersion() xversion.Semantic {
	return xversion.Semantic(fmt.Sprintf("%d.%d.%d", d.Channel.Version, 0, 0))
}

var _ migrate.Migratable = DBState{}

var (
	migrateV0toV1 = migrate.CreateMigration(migrate.MigrationConfig[DBState, DBState]{
		Name: "cesium.migrate",
		Migrate: func(context migrate.Context, state DBState) (DBState, error) {
			state.Channel.Version = version.V1
			return state, nil
		},
	})
	migrateV1toV2 = migrate.CreateMigration(migrate.MigrationConfig[DBState, DBState]{
		Name: "cesium.migrate",
		Migrate: func(context migrate.Context, state DBState) (DBState, error) {
			state.Channel.Version = version.V2
			if state.Channel.Virtual || state.Channel.IsIndex {
				return state, nil
			}
			// Any persisted channel with an index of 0 is rate based, so it should
			// be removed.
			state.ShouldIgnoreChannel = state.Channel.Index == 0
			return state, nil
		},
	})
	migrations = migrate.Migrations{
		"0.0.0": migrateV0toV1,
		"1.0.0": migrateV1toV2,
	}
	Migrate = migrate.Migrator(migrate.MigratorConfig[DBState, DBState]{
		Migrations: migrations,
	})
)

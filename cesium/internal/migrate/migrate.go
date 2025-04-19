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

type DBState struct {
	Channel core.Channel
	FS      xfs.FS
	Purge   bool
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
			state.Purge = state.Channel.Index == 0
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v4"
)

// MigrationsConfig configures the stored-task migration chain.
type MigrationsConfig struct {
	// Status writes the statuses of tasks that the migrations change.
	Status *status.Service
	// Configs routes each task type to the store that owns its config records.
	Configs config.Registry
}

// NewMigrations returns the ordered migration chain for stored tasks.
func NewMigrations(cfg MigrationsConfig) []gorp.Migration {
	return slices.Concat(
		[]gorp.Migration{v0.NormalizeKeys},
		v0.NewMigrations(v0.MigrationConfig{Status: cfg.Status}),
		[]gorp.Migration{
			v1.Migration,
			v2.NewMigration(v2.MigrationConfig{Status: cfg.Status}),
			v3.NewMigration(v3.MigrationConfig{
				Status:  cfg.Status,
				Configs: cfg.Configs,
			}),
			v2.NameMigration,
			v3.Migration,
			v3.RangeMigration,
			v4.NewSnapshotMigration(v3.MigrationConfig{
				Status:  cfg.Status,
				Configs: cfg.Configs,
			}),
			v4.Migration,
		},
	)
}

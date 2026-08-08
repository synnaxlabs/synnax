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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationsConfig configures the stored-task migration chain.
type MigrationsConfig struct {
	// Status backfills unknown statuses for tasks missing them.
	Status *status.Service
	// Ontology defines the config record relationships the config-record migration
	// creates.
	Ontology *ontology.Ontology
	// Configs routes each task type to the store that owns its config records.
	Configs common.ConfigRegistry
}

// NewMigrations returns the ordered migration chain for stored tasks.
func NewMigrations(cfg MigrationsConfig) []migrate.Migration {
	return append(
		v0.NewMigrations(v0.MigrationConfig{Status: cfg.Status}),
		v1.Migration,
		v2.Migration,
		v3.NewMigration(v3.MigrationConfig{
			Ontology: cfg.Ontology,
			Configs:  cfg.Configs,
		}),
	)
}

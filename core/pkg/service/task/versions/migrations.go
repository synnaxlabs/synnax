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
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v0 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// NormalizeKeys re-keys task rows stored under the pre-v0.54 key format. The head Task
// key is a UUID, which cannot decode the uint64 keys those rows carry, so
// normalization runs with the frozen v0 shape.
func NormalizeKeys() migrate.Migration {
	return gorp.NormalizeKeysMigration[v0.Key, v0.Task](gorp.NormalizeKeysMigrationKey)
}

// MigrationsConfig configures the stored-task migration chain.
type MigrationsConfig struct {
	// Status backfills unknown statuses for tasks missing them.
	Status *status.Service
	// Ontology defines the task resources the config-record migration creates and
	// removes.
	Ontology *ontology.Ontology
	// Configs routes each task type to the store that owns its config records.
	Configs config.Registry
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

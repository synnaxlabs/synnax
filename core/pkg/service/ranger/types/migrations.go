// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v1"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationConfig is the configuration for Migrations.
type MigrationConfig = v0.MigrationConfig

// Migrations returns the ordered migration chain for stored ranges.
func Migrations(cfg MigrationConfig) []migrate.Migration {
	return []migrate.Migration{
		v0.Migration(cfg),
		v1.CodecMigration,
		v1.ColorNullableMigration(),
	}
}

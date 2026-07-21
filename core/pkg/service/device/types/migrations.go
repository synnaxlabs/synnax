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
	v0 "github.com/synnaxlabs/synnax/pkg/service/device/types/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/device/types/v2"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationConfig is the configuration for Migrations.
type MigrationConfig = v0.MigrationConfig

// Migrations returns the ordered migration chain for stored devices.
func Migrations(cfg MigrationConfig) []migrate.Migration {
	return []migrate.Migration{
		v0.Migration(cfg),
		v2.CodecMigration,
		v2.Migration,
	}
}

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
	v0 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v2"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationsConfig configures the stored-rack migration chain.
type MigrationsConfig = v0.MigrationConfig

// NewMigrations returns the ordered migration chain for stored racks.
func NewMigrations(cfg MigrationsConfig) []migrate.Migration {
	return append(v0.NewMigrations(cfg), v1.Migration, v2.Migration)
}

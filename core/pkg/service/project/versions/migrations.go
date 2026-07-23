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
	v1 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationsConfig is the configuration for NewMigrations.
type MigrationsConfig = v1.MigrationsConfig

// NewMigrations returns the ordered migration chain for stored projects.
func NewMigrations(cfg MigrationsConfig) []migrate.Migration {
	return v1.NewMigrations(cfg)
}

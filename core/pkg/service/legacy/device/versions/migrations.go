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
	v0 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v1"
)

// MigrationsConfig configures the stored-device migration chain.
type MigrationsConfig = v0.MigrationConfig

// NewMigrations returns the ordered migration chain for stored devices.
func NewMigrations(cfg MigrationsConfig) []gorp.Migration {
	return slices.Concat(
		[]gorp.Migration{v0.NormalizeKeys},
		v0.NewMigrations(cfg),
		[]gorp.Migration{
			v1.Migration,
			v1.HTTPHealthCheckMigration,
			v1.EtherCATMakeMigration,
			v1.EtherCATChannelMapMigration,
		},
	)
}

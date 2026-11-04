// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v1"
)

// MigrateReadConfig lifts a v1 read config to v2.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// ReadMigration lifts stored read configs to v2.
var ReadMigration = migrate.NewEntry("v61_nan_fill_enabled", MigrateReadConfig)

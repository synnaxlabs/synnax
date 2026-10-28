// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
)

// MigrateScanConfig lifts a v2 scan config to v3.
func MigrateScanConfig(ctx context.Context, old v2.ScanConfig) (ScanConfig, error) {
	return autoMigrateScanConfig(ctx, old)
}

// ScanMigration lifts stored scan configs to v3.
var ScanMigration = migrate.NewEntry("v61_drop_ignored_models", MigrateScanConfig)

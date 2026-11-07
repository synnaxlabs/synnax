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
	v2 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v2"
)

// MigrateWriteChannel lifts a v2 write channel to v3. An entry channel gets a stop
// value of 0.
func MigrateWriteChannel(
	ctx context.Context,
	old v2.WriteChannel,
) (WriteChannel, error) {
	return autoMigrateWriteChannel(ctx, old)
}

// MigrateWriteConfig lifts a v2 write config to v3.
func MigrateWriteConfig(ctx context.Context, old v2.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// WriteMigration lifts stored write configs to v3.
var WriteMigration = migrate.NewEntry(
	"v61_ethercat_write_stop_value",
	MigrateWriteConfig,
)

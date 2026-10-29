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
	v2 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v2"
)

// MigrateReadChannel lifts a v2 read channel to v3, where it reads the whole value.
func MigrateReadChannel(ctx context.Context, old v2.ReadChannel) (ReadChannel, error) {
	return autoMigrateReadChannel(ctx, old)
}

// MigrateReadConfig lifts a v2 read config to v3.
func MigrateReadConfig(ctx context.Context, old v2.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// ReadMigration lifts stored read configs to v3.
var ReadMigration = migrate.NewEntry("v61_structure_fields", MigrateReadConfig)

// MigrateWriteChannel lifts a v2 write channel to v3, where it writes the value of its
// node.
func MigrateWriteChannel(
	ctx context.Context,
	old v2.WriteChannel,
) (WriteChannel, error) {
	ch, err := autoMigrateWriteChannel(ctx, old)
	ch.Type = WriteTypeNode
	return ch, err
}

// MigrateWriteConfig lifts a v2 write config to v3.
func MigrateWriteConfig(ctx context.Context, old v2.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// WriteMigration lifts stored write configs to v3.
var WriteMigration = migrate.NewEntry("v61_method_channels", MigrateWriteConfig)

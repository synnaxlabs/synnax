// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v3"
)

// MigrateReadConfig lifts a v3 HTTP read task config to v4.
func MigrateReadConfig(ctx context.Context, old v3.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateReadEndpoint lifts a v3 read endpoint to v4.
func MigrateReadEndpoint(
	ctx context.Context,
	old v3.ReadEndpoint,
) (ReadEndpoint, error) {
	return autoMigrateReadEndpoint(ctx, old)
}

// MigrateReadField lifts a v3 read field to v4.
func MigrateReadField(ctx context.Context, old v3.ReadField) (ReadField, error) {
	return autoMigrateReadField(ctx, old)
}

// MigrateWriteConfig lifts a v3 HTTP write task config to v4.
func MigrateWriteConfig(ctx context.Context, old v3.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// MigrateWriteEndpoint lifts a v3 write endpoint to v4.
func MigrateWriteEndpoint(
	ctx context.Context,
	old v3.WriteEndpoint,
) (WriteEndpoint, error) {
	return autoMigrateWriteEndpoint(ctx, old)
}

// MigrateChannelField lifts a v3 channel field to v4.
func MigrateChannelField(
	ctx context.Context,
	old v3.ChannelField,
) (ChannelField, error) {
	return autoMigrateChannelField(ctx, old)
}

// MigrateWriteField lifts a v3 write field to v4.
func MigrateWriteField(ctx context.Context, old v3.WriteField) (WriteField, error) {
	return autoMigrateWriteField(ctx, old)
}

// Migration lifts stored HTTP read task configs from v3 to v4.
var Migration = migrate.NewEntry("v61_optional_read_field", MigrateReadConfig)

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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v1"
)

// MigrateReadChannel lifts a v1 read channel to v2, where it reads the whole value.
func MigrateReadChannel(ctx context.Context, old v1.ReadChannel) (ReadChannel, error) {
	return autoMigrateReadChannel(ctx, old)
}

// MigrateReadConfig lifts a v1 read config to v2, where it samples its nodes.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	cfg.Mode = ReadModeSampled
	return cfg, err
}

// ReadMigration lifts stored read configs to v2.
var ReadMigration = migrate.NewEntry("v61_nan_fill_enabled", MigrateReadConfig)

// MigrateWriteChannel lifts a v1 write channel to v2, where it writes the value of its
// node.
func MigrateWriteChannel(_ context.Context, old v1.WriteChannel) (WriteChannel, error) {
	return WriteChannel{Variant: NodeWriteChannel{BaseWriteChannel: BaseWriteChannel{
		BaseChannel: old.BaseChannel,
		CmdChannel:  old.CmdChannel,
	}}}, nil
}

// MigrateWriteConfig lifts a v1 write config to v2.
func MigrateWriteConfig(ctx context.Context, old v1.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil {
		return WriteConfig{}, err
	}
	cfg.Channels, err = lo.MapErr(
		old.Channels,
		func(ch v1.WriteChannel, _ int) (WriteChannel, error) {
			return MigrateWriteChannel(ctx, ch)
		},
	)
	return cfg, err
}

// WriteMigration lifts stored write configs to v2.
var WriteMigration = migrate.NewEntry("v61_method_channels", MigrateWriteConfig)

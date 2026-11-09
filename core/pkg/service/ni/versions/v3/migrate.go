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

// MigrateAnalogWriteConfig lifts a v2 analog write config to v3.
func MigrateAnalogWriteConfig(
	ctx context.Context,
	old v2.AnalogWriteConfig,
) (AnalogWriteConfig, error) {
	return autoMigrateAnalogWriteConfig(ctx, old)
}

// MigrateBaseAOChannel lifts the shared fields of a v2 analog output channel to v3.
func MigrateBaseAOChannel(
	ctx context.Context,
	old v2.BaseAOChannel,
) (BaseAOChannel, error) {
	return autoMigrateBaseAOChannel(ctx, old)
}

// MigrateAOChannel lifts a v2 analog output channel to v3. A function generator drops
// its command and state channels. The other types keep them.
func MigrateAOChannel(ctx context.Context, old v2.AOChannel) (AOChannel, error) {
	ch, err := autoMigrateAOChannel(ctx, old)
	if err != nil {
		return AOChannel{}, err
	}
	base := old.Base()
	commanded := CommandedAOChannel{
		CmdChannel:       base.CmdChannel,
		StateChannel:     base.StateChannel,
		CmdChannelName:   base.CmdChannelName,
		StateChannelName: base.StateChannelName,
	}
	switch v := ch.Variant.(type) {
	case AOCurrentChannel:
		v.CommandedAOChannel = commanded
		ch.Variant = v
	case AOVoltageChannel:
		v.CommandedAOChannel = commanded
		ch.Variant = v
	}
	return ch, nil
}

// AnalogWriteMigration lifts stored analog write configs to v3.
var AnalogWriteMigration = migrate.NewEntry(
	"v61_drop_func_gen_channels",
	MigrateAnalogWriteConfig,
)

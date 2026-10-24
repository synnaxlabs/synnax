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

	v1 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v1"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateRegisterValue keeps a stored true as an explicit override and drops a stored
// false, so the channel follows the device. No released Console could set the v1
// fields, so a stored false was never a choice.
func MigrateRegisterValue(
	ctx context.Context,
	old v1.RegisterValue,
) (RegisterValue, error) {
	v, err := autoMigrateRegisterValue(ctx, old)
	if err != nil {
		return RegisterValue{}, err
	}
	if old.BytesSwapped {
		v.BytesSwapped = new(true)
	}
	if old.WordsSwapped {
		v.WordsSwapped = new(true)
	}
	return v, nil
}

// MigrateReadConfig lifts a v1 read config to v2.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateWriteConfig lifts a v1 write config to v2.
func MigrateWriteConfig(ctx context.Context, old v1.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

var (
	// ReadMigration lifts stored read configs to v2.
	ReadMigration = gorp.NewEntryMigration("v59_swap_inherit", MigrateReadConfig)
	// WriteMigration lifts stored write configs to v2.
	WriteMigration = gorp.NewEntryMigration("v59_swap_inherit", MigrateWriteConfig)
)

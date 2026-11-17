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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/opcua/versions/v1"
	"go.uber.org/zap"
)

// MigrateReadChannel lifts a v1 read channel to v2, where it reads the whole value.
func MigrateReadChannel(ctx context.Context, old v1.ReadChannel) (ReadChannel, error) {
	return autoMigrateReadChannel(ctx, old)
}

// MigrateReadConfig lifts a v1 read config to v2. A config in array mode becomes array
// sampling, and an array size below 1 clamps to 1.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	cfg.Sampling = Sampling{Variant: UnarySampling{}}
	if old.ArrayMode {
		cfg.Sampling.Variant = ArraySampling{Size: uint32(max(old.ArraySize, 1))}
	}
	return cfg, nil
}

func migrateStoredReadConfig(
	ctx context.Context,
	old v1.ReadConfig,
	ins alamos.Instrumentation,
) (ReadConfig, error) {
	cfg, err := MigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	if arr, ok := cfg.Sampling.Variant.(ArraySampling); ok &&
		int64(arr.Size) != int64(old.ArraySize) {
		ins.L.Warn(
			"clamped an OPC UA array size",
			zap.Stringer("config", old.Key),
			zap.String("field", "sampling.array.size"),
			zap.Int32("old", old.ArraySize),
			zap.Uint32("new", arr.Size),
		)
	}
	return cfg, nil
}

// ReadMigration lifts stored read configs to v2.
var ReadMigration = migrate.NewInstrumentedEntry(
	"v61_nan_fill_enabled",
	migrateStoredReadConfig,
)

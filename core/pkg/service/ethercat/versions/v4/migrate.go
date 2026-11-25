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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v3"
)

// MigrateReadChannel lifts a v3 read channel to v4. An entry channel with an empty data
// type gets none.
func MigrateReadChannel(_ context.Context, old v3.ReadChannel) (ReadChannel, error) {
	switch v := old.Variant.(type) {
	case v3.EntryReadChannel:
		return ReadChannel{Variant: EntryReadChannel{
			BaseReadChannel: v.BaseReadChannel,
			Index:           v.Index,
			SubIndex:        v.SubIndex,
			DataType:        lo.EmptyableToPtr(v.DataType),
			Calibrated:      v.Calibrated,
			Scale:           v.Scale,
		}}, nil
	case v3.UnresolvedReadChannel:
		return ReadChannel{Variant: UnresolvedReadChannel(v)}, nil
	}
	return ReadChannel{}, nil
}

// MigrateReadConfig lifts a v3 read config to v4.
func MigrateReadConfig(ctx context.Context, old v3.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// ReadMigration lifts stored read configs to v4.
var ReadMigration = migrate.NewEntry("v63_ethercat_read_data_type", MigrateReadConfig)

// MigrateWriteChannel lifts a v3 write channel to v4. An entry channel with an empty
// data type gets none.
func MigrateWriteChannel(_ context.Context, old v3.WriteChannel) (WriteChannel, error) {
	switch v := old.Variant.(type) {
	case v3.EntryWriteChannel:
		return WriteChannel{Variant: EntryWriteChannel{
			BaseWriteChannel: v.BaseWriteChannel,
			Index:            v.Index,
			SubIndex:         v.SubIndex,
			DataType:         lo.EmptyableToPtr(v.DataType),
			StopValue:        v.StopValue,
		}}, nil
	case v3.UnresolvedWriteChannel:
		return WriteChannel{Variant: UnresolvedWriteChannel(v)}, nil
	}
	return WriteChannel{}, nil
}

// MigrateWriteConfig lifts a v3 write config to v4.
func MigrateWriteConfig(ctx context.Context, old v3.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// WriteMigration lifts stored write configs to v4.
var WriteMigration = migrate.NewEntry("v63_ethercat_write_data_type", MigrateWriteConfig)

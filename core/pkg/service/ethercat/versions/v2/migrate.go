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

	v1 "github.com/synnaxlabs/synnax/pkg/service/ethercat/versions/v1"
)

// MigrateReadChannel lifts a v1 read channel: a manual channel keeps its address, and
// an automatic channel becomes unresolved until ResolveReadConfig matches its name.
func MigrateReadChannel(ctx context.Context, old v1.ReadChannel) (ReadChannel, error) {
	switch v := old.Variant.(type) {
	case v1.ManualReadChannel:
		return ReadChannel{Variant: EntryReadChannel{
			BaseReadChannel: v.BaseReadChannel,
			Index:           v.Index,
			SubIndex:        v.SubIndex,
			DataType:        v.DataType,
		}}, nil
	case v1.AutomaticReadChannel:
		return ReadChannel{Variant: UnresolvedReadChannel{
			BaseReadChannel: v.BaseReadChannel,
			Pdo:             v.Pdo,
		}}, nil
	}
	return autoMigrateReadChannel(ctx, old)
}

// MigrateWriteChannel lifts a v1 write channel: a manual channel keeps its address, and
// an automatic channel becomes unresolved until ResolveWriteConfig matches its name.
func MigrateWriteChannel(
	ctx context.Context,
	old v1.WriteChannel,
) (WriteChannel, error) {
	switch v := old.Variant.(type) {
	case v1.ManualWriteChannel:
		return WriteChannel{Variant: EntryWriteChannel{
			BaseWriteChannel: v.BaseWriteChannel,
			Index:            v.Index,
			SubIndex:         v.SubIndex,
			DataType:         v.DataType,
		}}, nil
	case v1.AutomaticWriteChannel:
		return WriteChannel{Variant: UnresolvedWriteChannel{
			BaseWriteChannel: v.BaseWriteChannel,
			Pdo:              v.Pdo,
		}}, nil
	}
	return autoMigrateWriteChannel(ctx, old)
}

// MigrateReadConfig lifts a v1 read config to v2.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateWriteConfig lifts a v1 write config to v2.
func MigrateWriteConfig(ctx context.Context, old v1.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

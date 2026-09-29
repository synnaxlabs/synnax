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

	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateLog lifts a v2 log into v3. v2 validated its timestamp precision to 0-3, so
// the narrowing to uint8 is exact.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	out.TimestampPrecision = uint8(old.TimestampPrecision)
	return out, nil
}

// MigrateChannelEntry replaces the v2 precision of -1, which showed the value exactly,
// with an absent precision. v2 validated every other precision to 0-17.
func MigrateChannelEntry(
	ctx context.Context,
	old v2.ChannelEntry,
) (ChannelEntry, error) {
	out, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Precision >= 0 {
		out.Precision = new(uint8(old.Precision))
	}
	return out, nil
}

// Migration lifts stored logs from v2 to v3.
var Migration = gorp.NewEntryMigration("v59_absent_log_precision", MigrateLog)

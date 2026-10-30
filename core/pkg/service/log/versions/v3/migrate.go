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
	"strings"

	"github.com/synnaxlabs/gorp/migrate"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
)

// MigrateLog lifts a v2 log to v3, clamping the timestamp precision to 0-3.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	out, err := autoMigrateLog(ctx, old)
	if err != nil {
		return Log{}, err
	}
	out.TimestampPrecision = uint8(min(max(old.TimestampPrecision, 0), 3))
	out.TimestampsHidden = old.ReceiptTimestampHidden
	return out, nil
}

// MigrateChannelEntry lifts a v2 channel entry to v3. The v2 sentinel precision of -1
// becomes an absent precision, a larger one clamps to 17, and an alias is cut to 64
// bytes.
func MigrateChannelEntry(
	ctx context.Context,
	old v2.ChannelEntry,
) (ChannelEntry, error) {
	out, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Precision >= 0 {
		precision := uint8(min(old.Precision, 17))
		out.Precision = &precision
	}
	out.Alias = strings.ToValidUTF8(old.Alias[:min(len(old.Alias), 64)], "")
	return out, nil
}

// Migration lifts stored logs from v2 to v3.
var Migration = migrate.NewEntry("v60_optional_channel_precision", MigrateLog)

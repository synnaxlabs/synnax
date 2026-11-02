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
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
)

// MigrateLog lifts a v2 log to v3.
func MigrateLog(ctx context.Context, old v2.Log) (Log, error) {
	return autoMigrateLog(ctx, old)
}

// MigrateChannelEntry lifts a v2 channel entry to v3. The v2 sentinel precision of -1
// becomes an absent precision.
func MigrateChannelEntry(
	ctx context.Context,
	old v2.ChannelEntry,
) (ChannelEntry, error) {
	out, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Precision >= 0 {
		precision := old.Precision
		out.Precision = &precision
	}
	return out, nil
}

// Migration lifts stored logs from v2 to v3.
var Migration = migrate.NewEntry("v60_optional_channel_precision", MigrateLog)

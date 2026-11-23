// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v5"
)

// MigrateLog lifts a v5 log to v6, which stores each channel color as a reference.
func MigrateLog(ctx context.Context, old v5.Log) (Log, error) {
	return autoMigrateLog(ctx, old)
}

// Migration lifts stored logs from v5 to v6.
var Migration = migrate.NewEntry("v63_color_ref", MigrateLog)

// MigrateChannelEntry lifts a v5 channel entry to v6. A color becomes a fixed color,
// and an absent color stays absent.
func MigrateChannelEntry(
	ctx context.Context,
	old v5.ChannelEntry,
) (ChannelEntry, error) {
	entry, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Color != nil {
		entry.Color = &color.Ref{Variant: color.FixedRef{Value: *old.Color}}
	}
	return entry, nil
}

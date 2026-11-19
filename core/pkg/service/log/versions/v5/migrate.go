// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v4"
	colorv0 "github.com/synnaxlabs/x/color/versions/v0"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateLog lifts a v4 log to v5, which stores each channel color as a fixed color
// with a 0-255 alpha byte.
func MigrateLog(ctx context.Context, old v4.Log) (Log, error) {
	return autoMigrateLog(ctx, old)
}

// Migration lifts stored logs from v4 to v5.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateLog)

// MigrateChannelEntry lifts a v4 channel entry to v5. A zero color becomes absent, as
// v4 stored the zero color to mean no color. Any other color becomes a fixed color.
func MigrateChannelEntry(
	ctx context.Context,
	old v4.ChannelEntry,
) (ChannelEntry, error) {
	entry, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Color == (colorv0.Color{}) {
		return entry, nil
	}
	c, err := xcolor.MigrateColor(ctx, old.Color)
	if err != nil {
		return ChannelEntry{}, err
	}
	entry.Color = &color.Ref{Variant: color.FixedRef{Value: c}}
	return entry, nil
}

// NameMigration names each stored log that has an empty name "Log <key>", where <key>
// is the first eight characters of its key.
var NameMigration = migrate.NewEntry(
	"v62_require_name",
	func(_ context.Context, l Log) (Log, error) {
		if l.Name == "" {
			l.Name = "Log " + l.Key.String()[:8]
		}
		return l, nil
	},
)

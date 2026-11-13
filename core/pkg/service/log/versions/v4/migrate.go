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

	"github.com/synnaxlabs/gorp/migrate"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	colorv0 "github.com/synnaxlabs/x/color/versions/v0"
	color "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateLog lifts a v3 log to v4, which stores every color alpha as a 0-255 byte.
func MigrateLog(ctx context.Context, old v3.Log) (Log, error) {
	return autoMigrateLog(ctx, old)
}

// Migration lifts stored logs from v3 to v4.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateLog)

// MigrateChannelEntry lifts a v3 channel entry to v4. A zero color becomes absent, as
// v3 stored the zero color to mean no color.
func MigrateChannelEntry(
	ctx context.Context,
	old v3.ChannelEntry,
) (ChannelEntry, error) {
	entry, err := autoMigrateChannelEntry(ctx, old)
	if err != nil {
		return ChannelEntry{}, err
	}
	if old.Color == (colorv0.Color{}) {
		return entry, nil
	}
	c, err := color.MigrateColor(ctx, old.Color)
	if err != nil {
		return ChannelEntry{}, err
	}
	entry.Color = &c
	return entry, nil
}

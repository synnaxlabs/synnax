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
	"math"

	v1 "github.com/synnaxlabs/x/color/versions/v1"
)

// MigrateColor converts a v1 color with a 0-1 alpha to a 0-255 alpha. A non-zero alpha
// stays non-zero, so a faint color never turns into the zero color.
func MigrateColor(ctx context.Context, old v1.Color) (Color, error) {
	c, err := autoMigrateColor(ctx, old)
	if err != nil {
		return Color{}, err
	}
	if old.A > 0 {
		c.A = uint8(max(1, math.Round(min(old.A, 1)*255)))
	}
	return c, nil
}

// MigrateStop converts a v1 stop to v2.
func MigrateStop(ctx context.Context, old v1.Stop) (Stop, error) {
	return autoMigrateStop(ctx, old)
}

// MigrateBand converts a v1 band to v2.
func MigrateBand(ctx context.Context, old v1.Band) (Band, error) {
	return autoMigrateBand(ctx, old)
}

// MigrateScale converts a v1 scale to v2.
func MigrateScale(ctx context.Context, old v1.Scale) (Scale, error) {
	return autoMigrateScale(ctx, old)
}

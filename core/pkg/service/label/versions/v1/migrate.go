// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v0"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateLabel lifts a v0 label to v1. Its color becomes a fixed color with a 0-255
// alpha.
func MigrateLabel(ctx context.Context, old v0.Label) (Label, error) {
	l, err := autoMigrateLabel(ctx, old)
	if err != nil {
		return Label{}, err
	}
	c, err := xcolor.MigrateColor(ctx, old.Color)
	if err != nil {
		return Label{}, err
	}
	l.Color = color.Ref{Variant: color.FixedRef{Value: c}}
	return l, nil
}

// Migration lifts stored labels from v0 to v1.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateLabel)

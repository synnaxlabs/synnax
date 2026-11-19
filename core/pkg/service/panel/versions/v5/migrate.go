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
	v4 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v4"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// MigratePanel lifts a v4 panel to v5, which stores the icon color as a fixed color
// with a 0-255 alpha byte.
func MigratePanel(ctx context.Context, old v4.Panel) (Panel, error) {
	p, err := autoMigratePanel(ctx, old)
	if err != nil || old.Color == nil {
		return p, err
	}
	c, err := xcolor.MigrateColor(ctx, *old.Color)
	if err != nil {
		return Panel{}, err
	}
	p.Color = &color.Ref{Variant: color.FixedRef{Value: c}}
	return p, nil
}

// Migration lifts stored panels from v4 to v5.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigratePanel)

// NameMigration names each stored panel that has an empty name "Panel <key>", where
// <key> is the first eight characters of its key.
var NameMigration = migrate.NewEntry(
	"v62_require_name",
	func(_ context.Context, p Panel) (Panel, error) {
		if p.Name == "" {
			p.Name = "Panel " + p.Key.String()[:8]
		}
		return p, nil
	},
)

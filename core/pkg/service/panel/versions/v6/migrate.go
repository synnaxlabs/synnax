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
	v5 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v5"
)

// MigratePanel lifts a v5 panel to v6, which stores the icon color as a reference. A
// color becomes a fixed color, and an absent color stays absent.
func MigratePanel(ctx context.Context, old v5.Panel) (Panel, error) {
	p, err := autoMigratePanel(ctx, old)
	if err != nil {
		return Panel{}, err
	}
	if old.Color != nil {
		p.Color = &color.Ref{Variant: color.FixedRef{Value: *old.Color}}
	}
	return p, nil
}

// Migration lifts stored panels from v5 to v6.
var Migration = migrate.NewEntry("v63_color_ref", MigratePanel)

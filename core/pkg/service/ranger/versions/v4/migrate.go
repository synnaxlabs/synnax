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
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v3"
)

// MigrateRange lifts a v3 range to v4, which stores its color as a reference. A color
// becomes a fixed color, and an absent color stays absent.
func MigrateRange(ctx context.Context, old v3.Range) (Range, error) {
	r, err := autoMigrateRange(ctx, old)
	if err != nil {
		return Range{}, err
	}
	if old.Color != nil {
		r.Color = &color.Ref{Variant: color.FixedRef{Value: *old.Color}}
	}
	return r, nil
}

// Migration lifts stored ranges from v3 to v4.
var Migration = migrate.NewEntry("v63_color_ref", MigrateRange)

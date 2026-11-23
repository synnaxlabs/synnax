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

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/label/versions/v1"
)

// MigrateLabel lifts a v1 label to v2, which stores its color as a reference. The
// color becomes a fixed color.
func MigrateLabel(ctx context.Context, old v1.Label) (Label, error) {
	l, err := autoMigrateLabel(ctx, old)
	if err != nil {
		return Label{}, err
	}
	l.Color = color.Ref{Variant: color.FixedRef{Value: old.Color}}
	return l, nil
}

// Migration lifts stored labels from v1 to v2.
var Migration = migrate.NewEntry("v63_color_ref", MigrateLabel)

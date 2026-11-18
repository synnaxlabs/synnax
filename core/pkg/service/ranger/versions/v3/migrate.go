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
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateRange lifts a v2 range to v3. Its color becomes a fixed color with a 0-255
// alpha, and no color stays absent.
func MigrateRange(ctx context.Context, old v2.Range) (Range, error) {
	r, err := autoMigrateRange(ctx, old)
	if err != nil {
		return Range{}, err
	}
	if old.Color == nil {
		return r, nil
	}
	c, err := xcolor.MigrateColor(ctx, *old.Color)
	if err != nil {
		return Range{}, err
	}
	r.Color = &color.Ref{Variant: color.FixedRef{Value: c}}
	return r, nil
}

// Migration lifts stored ranges from v2 to v3.
var Migration = migrate.NewEntry("v62_color_alpha_byte", MigrateRange)

// MigrateWhere lifts a v2 range filter to v3. A v2 filter sets none of the lines that
// v3 adds.
func MigrateWhere(ctx context.Context, old v2.Where) (Where, error) {
	and, err := migrateWheres(ctx, old.And)
	if err != nil {
		return Where{}, err
	}
	or, err := migrateWheres(ctx, old.Or)
	if err != nil {
		return Where{}, err
	}
	var not *Where
	if old.Not != nil {
		w, err := MigrateWhere(ctx, *old.Not)
		if err != nil {
			return Where{}, err
		}
		not = &w
	}
	return Where{
		And:      and,
		Or:       or,
		Not:      not,
		Key:      old.Key,
		Name:     old.Name,
		Overlaps: old.Overlaps,
		Labels:   old.Labels,
		Parents:  old.Parents,
	}, nil
}

func migrateWheres(ctx context.Context, old []v2.Where) ([]Where, error) {
	if old == nil {
		return nil, nil
	}
	ws := make([]Where, len(old))
	for i, w := range old {
		var err error
		if ws[i], err = MigrateWhere(ctx, w); err != nil {
			return nil, err
		}
	}
	return ws, nil
}

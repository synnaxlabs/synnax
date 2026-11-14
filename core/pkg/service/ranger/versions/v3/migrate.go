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
	v2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
)

// MigrateRange lifts a v2 range to v3, which stores every color alpha as a 0-255
// byte.
func MigrateRange(ctx context.Context, old v2.Range) (Range, error) {
	return autoMigrateRange(ctx, old)
}

// Migration lifts stored ranges from v2 to v3.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateRange)

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

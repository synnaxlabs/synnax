// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/color"
	labelv54 "github.com/synnaxlabs/x/label/migrations/v54"
	statusv54 "github.com/synnaxlabs/x/status/migrations/v54"
	"github.com/synnaxlabs/x/telem"
)

// MigrateStatus migrates a persisted v54 status into the current server-side Status
// entity. The v54 status stored its descriptive fields and identity in a single flat
// struct; the current entity splits the descriptive payload into the embedded
// Status while keeping key, name, and labels on the entity itself.
func MigrateStatus[Details any](
	ctx context.Context,
	old statusv54.Status[Details],
) (Status[Details], error) {
	labels := make([]label.Label, len(old.Labels))
	for i, v := range old.Labels {
		var err error
		if labels[i], err = migrateLabel(ctx, v); err != nil {
			return Status[Details]{}, err
		}
	}
	return Status[Details]{
		Status: Status[Details]{
			Variant:     Variant(old.Variant),
			Message:     old.Message,
			Description: old.Description,
			Time:        telem.TimeStamp(old.Time),
			Details:     old.Details,
		},
		Key:    old.Key,
		Name:   old.Name,
		Labels: labels,
	}, nil
}

func migrateLabel(_ context.Context, old labelv54.Label) (label.Label, error) {
	return label.Label{
		Key:   label.Key(old.Key),
		Name:  old.Name,
		Color: color.Color(old.Color),
	}, nil
}

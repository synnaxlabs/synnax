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

	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// colorNullableMigration converts every range from the Orc value-color layout (Color
// stored inline) to the current nullable layout (Color a presence-flagged pointer). A
// zero stored color denoted "no color" under the value layout, so it maps to nil. It
// depends on the codec migration so it always reads the deterministic value-color
// encoding that migration leaves behind.
var colorNullableMigration = gorp.NewEntryMigration(
	"range_color_nullable",
	func(_ context.Context, old v0.Range) (Range, error) {
		rng := Range{Key: old.Key, Name: old.Name, TimeRange: old.TimeRange}
		if !old.Color.IsZero() {
			c := old.Color
			rng.Color = &c
		}
		return rng, nil
	},
	v0.Migration.Key(),
)

// Migrations is the ordered set of migrations introduced at this version.
var Migrations = []migrate.Migration{colorNullableMigration}

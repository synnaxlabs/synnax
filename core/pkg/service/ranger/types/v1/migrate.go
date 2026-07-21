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

	v0 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v0"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// CodecMigrationKey is the key of the migration that re-encodes ranges from the
// legacy msgpack layout to the orc value-color layout. It is pinned to v0.Range so
// it always produces the value-color encoding regardless of how Range later evolves.
const CodecMigrationKey = "msgpack_to_orc"

// colorNullableMigrationKey is the key of the migration that re-encodes Range.Color
// from a stored value to a nullable pointer.
const colorNullableMigrationKey = "range_color_nullable"

// ColorNullableMigration converts every range from the orc value-color layout (Color
// stored inline) to the current nullable layout (Color a presence-flagged pointer).
// A zero stored color denoted "no color" under the value layout, so it maps to nil.
// It depends on CodecMigrationKey so it always reads the deterministic value-color
// encoding that migration leaves behind.
func ColorNullableMigration() migrate.Migration {
	return gorp.NewEntryMigration(
		colorNullableMigrationKey,
		func(_ context.Context, old v0.Range) (Range, error) {
			rng := Range{Key: old.Key, Name: old.Name, TimeRange: old.TimeRange}
			if !old.Color.IsZero() {
				c := old.Color
				rng.Color = &c
			}
			return rng, nil
		},
		CodecMigrationKey,
	)
}

// CodecMigration re-encodes ranges from the legacy msgpack layout to the orc
// value-color layout. It is pinned to v0.Range so it always produces the
// value-color encoding regardless of how Range later evolves.
var CodecMigration = gorp.CodecMigration[Key, v0.Range](
	CodecMigrationKey, v0.MigrationKey,
)

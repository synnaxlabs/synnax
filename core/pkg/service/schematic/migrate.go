// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	v56 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v56"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// MigrateSchematic transforms the previous schematic snapshot (v56) into the
// current Schematic. v56 stored configs as opaque records written verbatim by
// the Console (camelCase field keys, never validated); each entry is
// normalized to the snake_case wire form and decoded into the typed element
// config union. Entries that match no known variant are dropped rather than
// failing the migration.
func MigrateSchematic(ctx context.Context, old v56.Schematic) (Schematic, error) {
	out, err := AutoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	out.Configs = make(map[string]ElementConfig, len(old.Configs))
	for k, raw := range old.Configs {
		if cfg, ok := migrateConfigEntry(raw); ok {
			out.Configs[k] = cfg
		}
	}
	return out, nil
}

// migrateConfigEntry converts a v56 stored config (camelCase keys, written
// verbatim by the Console) into the typed element config. ok is false when
// the entry does not conform to any known variant.
func migrateConfigEntry(raw msgpack.EncodedJSON) (ElementConfig, bool) {
	normalized := normalizeConfigKeys(raw)
	extractTelemArgs(normalized)
	cfg, err := decodeElementConfig(normalized)
	if err != nil {
		return ElementConfig{}, false
	}
	return cfg, true
}

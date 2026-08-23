// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8

import (
	"context"

	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateSchematic lifts a v7 schematic into the v8 shape, dropping each node's
// measured dimensions and decoding its opaque configs into the element config union.
func MigrateSchematic(ctx context.Context, old v7.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	out.Configs = typeConfigs(old.Configs)
	return out, nil
}

// typeConfigs decodes v7's opaque config entries into the element config union. The
// entries reach here in the camelCase form the Console wrote verbatim and never
// validated, so each is normalized to the snake_case wire form and has its stored
// telem pipelines rewritten into semantic arguments first. An entry matching no known
// variant is dropped rather than failing the migration.
func typeConfigs(raw map[string]msgpack.EncodedJSON) map[string]ElementConfig {
	out := make(map[string]ElementConfig, len(raw))
	for k, entry := range raw {
		normalized := NormalizeConfigKeys(entry)
		if normalized == nil {
			continue
		}
		extractTelemArgs(normalized)
		cfg, err := DecodeElementConfig(normalized)
		if err != nil {
			continue
		}
		out[k] = cfg
	}
	return out
}

// MigrateNode lifts a v7 node into the v8 shape, dropping its measured dimensions.
func MigrateNode(ctx context.Context, old v7.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// Migration lifts stored schematics from v7 to v8.
var Migration = gorp.NewEntryMigration("v57_drop_node_measured", MigrateSchematic)

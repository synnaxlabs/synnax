// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9

import (
	"context"
	"slices"

	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateSchematic lifts a v8 schematic into the v9 shape, decoding its opaque configs
// into the element config union. A config the union rejects is dropped: the lift runs
// unattended over every stored schematic, so one bad entry must not fail it. Prefer
// ImportSchematic wherever the caller can report the loss.
func MigrateSchematic(ctx context.Context, old v8.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	out.Configs, _ = typeConfigs(old.Configs)
	return out, nil
}

// ImportSchematic lifts a v8 schematic into the v9 shape for an import. Unlike
// MigrateSchematic it keeps no partial result: it wraps validate.ErrValidation naming
// every node whose config the union rejects.
func ImportSchematic(ctx context.Context, old v8.Schematic) (Schematic, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	configs, rejected := typeConfigs(old.Configs)
	if len(rejected) > 0 {
		return Schematic{}, rejectedConfigsError(rejected)
	}
	out.Configs = configs
	return out, nil
}

// typeConfigs decodes v8's opaque config entries into the element config union. The
// entries reach here in the camelCase form the Console wrote verbatim and never
// validated, so each is normalized to the snake_case wire form and has its stored
// telem pipelines rewritten into semantic arguments first. It returns the entries the
// union rejected, keyed by node, alongside the ones it accepted.
func typeConfigs(
	raw map[string]msgpack.EncodedJSON,
) (map[string]ElementConfig, map[string]error) {
	var (
		out      = make(map[string]ElementConfig, len(raw))
		rejected map[string]error
	)
	for k, entry := range raw {
		normalized := NormalizeConfigKeys(entry)
		if normalized != nil {
			extractTelemArgs(normalized)
		}
		cfg, err := DecodeElementConfig(normalized)
		if err != nil {
			if rejected == nil {
				rejected = make(map[string]error)
			}
			rejected[k] = err
			continue
		}
		out[k] = cfg
	}
	return out, rejected
}

// rejectedConfigsError joins per-node decode failures into one validation error, in
// node order so the message is stable.
func rejectedConfigsError(rejected map[string]error) error {
	keys := make([]string, 0, len(rejected))
	for k := range rejected {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	joined := make([]error, len(keys))
	for i, k := range keys {
		joined[i] = errors.Wrapf(rejected[k], "node %s", k)
	}
	return errors.Join(joined...)
}

// Migration types stored schematic element configs, lifting them from v8 to v9.
var Migration = gorp.NewEntryMigration("v58_type_element_configs", MigrateSchematic)

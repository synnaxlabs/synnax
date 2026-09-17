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

	"github.com/synnaxlabs/alamos"
	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

// MigrateSchematic lifts a v8 schematic into the v9 shape, decoding its opaque configs
// into the element config union and filling each one's schema defaults. A config the
// union rejects is reset to its variant's defaults, or dropped when the variant is
// unknown: the lift runs unattended over every stored schematic, so one bad entry must
// not fail it. Prefer ImportSchematic wherever the caller can report the loss.
func MigrateSchematic(ctx context.Context, old v8.Schematic) (Schematic, error) {
	out, _, err := lift(ctx, old)
	return out, err
}

// ImportSchematic lifts a v8 schematic into the v9 shape for an import, filling each
// config's schema defaults. Unlike MigrateSchematic it keeps no partial result: it
// wraps validate.ErrValidation naming every node whose config the union rejects.
func ImportSchematic(ctx context.Context, old v8.Schematic) (Schematic, error) {
	out, losses, err := lift(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	if len(losses) > 0 {
		return Schematic{}, rejectedConfigsError(losses)
	}
	return out, nil
}

// migrateStored is MigrateSchematic for the stored-data migration, which logs every
// config the lift reset or dropped so a vanished symbol can be traced to its cause.
func migrateStored(
	ctx context.Context,
	old v8.Schematic,
	ins alamos.Instrumentation,
) (Schematic, error) {
	out, losses, err := lift(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	for key, loss := range losses {
		fields := []zap.Field{
			zap.Stringer("schematic", old.Key),
			zap.String("node", key),
			zap.Error(loss.err),
		}
		if loss.reset {
			ins.L.Warn("reset a rejected schematic config to its variant's defaults", fields...)
		} else {
			ins.L.Warn("dropped a schematic config naming no known variant", fields...)
		}
	}
	return out, nil
}

// lift runs the v8 to v9 migration and reports, keyed by node, every config the union
// rejected and how the lift carried it.
func lift(
	ctx context.Context,
	old v8.Schematic,
) (Schematic, map[string]configLoss, error) {
	out, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, nil, err
	}
	var losses map[string]configLoss
	out.Configs, losses = typeConfigs(old.Configs)
	out.ApplyDefaults()
	return out, losses, nil
}

// configLoss describes how the lift carried a stored config the union rejected.
type configLoss struct {
	// err is why the union rejected the stored config.
	err error
	// reset is true when the config was replaced by its variant's defaults, false when
	// it was dropped because the variant is unknown.
	reset bool
}

// typeConfigs decodes v8's opaque config entries into the element config union. The
// entries reach here in the camelCase form the Console wrote verbatim and never
// validated, so each is normalized to the snake_case wire form and has its stored
// telem pipelines, legacy page keys, and zero colors rewritten into the typed shape
// first. An entry the union rejects is replaced by its variant's zero config when the
// variant is known and left out otherwise; both are reported in the returned losses.
func typeConfigs(
	raw map[string]msgpack.EncodedJSON,
) (map[string]ElementConfig, map[string]configLoss) {
	var (
		out    = make(map[string]ElementConfig, len(raw))
		losses map[string]configLoss
	)
	for k, entry := range raw {
		normalized := NormalizeConfigKeys(entry)
		if normalized != nil {
			extractTelemArgs(normalized)
			normalizePage(normalized)
			stripZeroColors(map[string]any(normalized))
		}
		cfg, err := DecodeElementConfig(normalized)
		if err == nil {
			out[k] = cfg
			continue
		}
		if losses == nil {
			losses = make(map[string]configLoss)
		}
		loss := configLoss{err: err}
		if normalized != nil {
			cfg, fallbackErr := DecodeElementConfig(
				msgpack.EncodedJSON{"variant": normalized["variant"]},
			)
			if fallbackErr == nil {
				out[k] = cfg
				loss.reset = true
			}
		}
		losses[k] = loss
	}
	return out, losses
}

// rejectedConfigsError joins per-node decode failures into one validation error, in
// node order so the message is stable.
func rejectedConfigsError(losses map[string]configLoss) error {
	keys := make([]string, 0, len(losses))
	for k := range losses {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	joined := make([]error, len(keys))
	for i, k := range keys {
		joined[i] = errors.Wrapf(losses[k].err, "node %s", k)
	}
	return errors.Join(joined...)
}

// Migration types stored schematic element configs, lifting them from v8 to v9.
var Migration = gorp.NewInstrumentedEntryMigration(
	"v58_type_element_configs", migrateStored,
)

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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/vmihailenco/msgpack/v5"
)

// Migration lifts stored symbols out of the untyped v0 shape into the typed v2 shape.
var Migration = gorp.NewEntryMigration("v2_typed_symbol", migrateSymbol)

func migrateSymbol(ctx context.Context, old v0.Symbol) (Symbol, error) {
	out, err := autoMigrateSymbol(ctx, old)
	if err != nil {
		return Symbol{}, err
	}
	if len(old.Data) > 0 {
		b, err := msgpack.Marshal(old.Data)
		if err != nil {
			return Symbol{}, errors.Wrap(err, "encode v0 symbol data")
		}
		if err = msgpack.Unmarshal(b, &out.Data); err != nil {
			return Symbol{}, errors.Wrap(err, "decode v0 symbol data")
		}
	}
	out.ApplyDefaults()
	return out, nil
}

// SpecFromConsole lifts the frozen Console export's specification into the current
// Spec.
func SpecFromConsole(spec legacy.Spec) Spec {
	return Spec{
		SVG:             spec.SVG,
		Variant:         spec.Variant,
		Scale:           spec.Scale,
		ScaleStroke:     spec.ScaleStroke,
		PreviewViewport: spec.PreviewViewport,
		States: lo.Map(spec.States, func(s legacy.State, _ int) State {
			return State{
				Key:  s.Key,
				Name: s.Name,
				Regions: lo.Map(s.Regions, func(r legacy.Region, _ int) Region {
					return Region{
						Key:         r.Key,
						Name:        r.Name,
						Selectors:   r.Selectors,
						StrokeColor: r.StrokeColor,
						FillColor:   r.FillColor,
					}
				}),
			}
		}),
		Handles: lo.Map(spec.Handles, func(h legacy.Handle, _ int) Handle {
			return Handle{
				Key: h.Key, Position: h.Position, Orientation: h.Orientation,
			}
		}),
	}
}

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
	"math"

	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateSchematic lifts a v7 schematic into the v8 shape, dropping each node's
// measured dimensions.
func MigrateSchematic(ctx context.Context, old v7.Schematic) (Schematic, error) {
	return autoMigrateSchematic(ctx, old)
}

// MigrateNode lifts a v7 node into the v8 shape, dropping its measured dimensions.
func MigrateNode(ctx context.Context, old v7.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// Migration lifts stored schematics from v7 to v8.
var Migration = gorp.NewEntryMigration("v57_drop_node_measured", MigrateSchematic)

// A scale written before it could rotate stores "left" and sizes its bar together with
// the tick gutter beside it.
const (
	scaleVariant     = "scale"
	verticalScale    = "top"
	horizontalScale  = "right"
	scaleGutter      = 26
	variantField     = "variant"
	orientationField = "orientation"
	dimensionsField  = "dimensions"
	indicatorField   = "indicator"
	showScaleField   = "showScale"
	widthField       = "width"
)

// NormalizeScales restates every stored scale in s as a vertical bar sized without its
// gutter, leaving the rest of the configs untouched.
func NormalizeScales(s Schematic) {
	for _, cfg := range s.Configs {
		variant, ok := cfg[variantField].(string)
		if !ok || variant != scaleVariant {
			continue
		}
		o, ok := cfg[orientationField].(string)
		if ok && (o == verticalScale || o == horizontalScale) {
			continue
		}
		cfg[orientationField] = verticalScale
		narrowScaleBar(cfg)
	}
}

// narrowScaleBar takes the gutter off cfg's width. Hidden ticks reserved none.
func narrowScaleBar(cfg msgpack.EncodedJSON) {
	if indicator, ok := cfg[indicatorField].(map[string]any); ok {
		if shown, ok := indicator[showScaleField].(bool); ok && !shown {
			return
		}
	}
	dims, ok := cfg[dimensionsField].(map[string]any)
	if !ok {
		return
	}
	width, ok := dims[widthField].(float64)
	if !ok {
		return
	}
	dims[widthField] = math.Max(0, width-scaleGutter)
}

// ScaleMigration restates the axis and bar width of stored scales.
var ScaleMigration = gorp.NewEntryMigration(
	"v58_scale_geometry",
	func(_ context.Context, s Schematic) (Schematic, error) {
		NormalizeScales(s)
		return s, nil
	},
)

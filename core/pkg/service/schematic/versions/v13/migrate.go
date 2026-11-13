// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v13

import (
	"context"
	"reflect"

	"github.com/synnaxlabs/gorp/migrate"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	color "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateSchematic lifts a v12 schematic to v13, which stores every color alpha as a
// 0-255 byte. An optional color that holds the zero color becomes absent, as v12 stored
// the zero color to mean that the theme picks the color. Symbol state overrides keep
// their zero colors, which paint a region transparent.
func MigrateSchematic(ctx context.Context, old v12.Schematic) (Schematic, error) {
	s, err := autoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	for k, cfg := range s.Configs {
		v := reflect.ValueOf(&cfg).Elem()
		clearZeroColors(v)
		s.Configs[k] = cfg
	}
	return s, nil
}

var (
	colorPtrType = reflect.TypeFor[*color.Color]()
	statesType   = reflect.TypeFor[[]symbol.State]()
)

// clearZeroColors sets every optional color under v that holds the zero color to nil,
// except inside symbol states. v must be settable.
func clearZeroColors(v reflect.Value) {
	if v.Type() == statesType {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return
		}
		if v.Type() == colorPtrType {
			if *v.Interface().(*color.Color) == (color.Color{}) {
				v.SetZero()
			}
			return
		}
		clearZeroColors(v.Elem())
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		c := reflect.New(v.Elem().Type()).Elem()
		c.Set(v.Elem())
		clearZeroColors(c)
		v.Set(c)
	case reflect.Struct:
		for i := range v.NumField() {
			if f := v.Field(i); f.CanSet() {
				clearZeroColors(f)
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			clearZeroColors(v.Index(i))
		}
	default:
	}
}

// Migration lifts stored schematics from v12 to v13.
var Migration = migrate.NewEntry("v61_color_alpha_byte", MigrateSchematic)

// MigrateElementConfig lifts a v12 element config to v13.
func MigrateElementConfig(
	ctx context.Context,
	old v12.ElementConfig,
) (ElementConfig, error) {
	return autoMigrateElementConfig(ctx, old)
}

// MigrateStaticSymbolConfig lifts a v12 static symbol config to v13.
func MigrateStaticSymbolConfig(
	ctx context.Context,
	old v12.StaticSymbolConfig,
) (StaticSymbolConfig, error) {
	return autoMigrateStaticSymbolConfig(ctx, old)
}

// MigrateToggleSymbolConfig lifts a v12 toggle symbol config to v13.
func MigrateToggleSymbolConfig(
	ctx context.Context,
	old v12.ToggleSymbolConfig,
) (ToggleSymbolConfig, error) {
	return autoMigrateToggleSymbolConfig(ctx, old)
}

// MigrateStalenessConfig lifts a v12 staleness config to v13.
func MigrateStalenessConfig(
	ctx context.Context,
	old v12.StalenessConfig,
) (StalenessConfig, error) {
	return autoMigrateStalenessConfig(ctx, old)
}

// MigrateStateMapping lifts a v12 state mapping to v13.
func MigrateStateMapping(
	ctx context.Context,
	old v12.StateMapping,
) (StateMapping, error) {
	return autoMigrateStateMapping(ctx, old)
}

// MigrateScaleIndicatorConfig lifts a v12 scale indicator config to v13.
func MigrateScaleIndicatorConfig(
	ctx context.Context,
	old v12.ScaleIndicatorConfig,
) (ScaleIndicatorConfig, error) {
	return autoMigrateScaleIndicatorConfig(ctx, old)
}

// MigrateDummyToggleSymbolConfig lifts a v12 dummy toggle symbol config to v13.
func MigrateDummyToggleSymbolConfig(
	ctx context.Context,
	old v12.DummyToggleSymbolConfig,
) (DummyToggleSymbolConfig, error) {
	return autoMigrateDummyToggleSymbolConfig(ctx, old)
}

// MigrateToggleConfig lifts a v12 toggle config to v13.
func MigrateToggleConfig(
	ctx context.Context,
	old v12.ToggleConfig,
) (ToggleConfig, error) {
	return autoMigrateToggleConfig(ctx, old)
}

// MigrateSegmentedEdgeConfig lifts a v12 segmented edge config to v13.
func MigrateSegmentedEdgeConfig(
	ctx context.Context,
	old v12.SegmentedEdgeConfig,
) (SegmentedEdgeConfig, error) {
	return autoMigrateSegmentedEdgeConfig(ctx, old)
}

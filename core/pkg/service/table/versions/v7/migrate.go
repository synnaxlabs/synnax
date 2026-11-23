// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v6 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v6"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
)

// MigrateTable lifts a v6 table to v7, which stores each cell color as a reference.
func MigrateTable(ctx context.Context, old v6.Table) (Table, error) {
	return autoMigrateTable(ctx, old)
}

// Migration lifts stored tables from v6 to v7.
var Migration = migrate.NewEntry("v63_color_ref", MigrateTable)

// MigrateRow lifts a v6 row to v7.
func MigrateRow(ctx context.Context, old v6.Row) (Row, error) {
	return autoMigrateRow(ctx, old)
}

// MigrateCell lifts a v6 cell to v7.
func MigrateCell(ctx context.Context, old v6.Cell) (Cell, error) {
	return autoMigrateCell(ctx, old)
}

// MigrateCellConfig lifts a v6 cell config to v7. Each color becomes a fixed color, and
// an absent color stays absent.
func MigrateCellConfig(ctx context.Context, old v6.CellConfig) (CellConfig, error) {
	c, err := autoMigrateCellConfig(ctx, old)
	if err != nil {
		return CellConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v6.TextCellConfig:
		v := as[TextCellConfig](c.Variant)
		v.FillColor = fixed(o.FillColor)
		v.TextColor = fixed(o.TextColor)
		c.Variant = v
	case v6.ValueCellConfig:
		v := as[ValueCellConfig](c.Variant)
		v.FillColor = fixed(o.FillColor)
		v.TextColor = fixed(o.TextColor)
		v.StalenessColor = fixed(o.StalenessColor)
		c.Variant = v
	}
	return c, nil
}

// as returns a migrated variant as a T. It panics when v is not a T, since a generated
// migration keeps each variant's kind.
func as[T any](v any) T {
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("migrated variant is a %T, not a %T", v, t))
	}
	return t
}

func fixed(c *xcolor.Color) *color.Ref {
	if c == nil {
		return nil
	}
	return &color.Ref{Variant: color.FixedRef{Value: *c}}
}

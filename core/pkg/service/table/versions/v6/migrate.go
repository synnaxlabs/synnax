// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/gorp/migrate"
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
	xcolorv1 "github.com/synnaxlabs/x/color/versions/v1"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/set"
)

// MigrateTable lifts a v5 table to v6. Every color alpha becomes a 0-255 byte, each
// cell color becomes a fixed color, and each entry of the cells map moves into the row
// cell that references it. A row cell with no config gets an empty text cell, so it
// keeps its column. A cell that no row references is dropped. A key that more than one
// row cell references keeps its first cell, and each later cell gets a new key with a
// copy of the config. Each row then holds one cell per column: cells past the last
// column are dropped, and a short row gets empty text cells with new keys.
func MigrateTable(ctx context.Context, old v5.Table) (Table, error) {
	t, err := autoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	used := make(set.Set[string])
	for _, row := range old.Rows {
		used.Add(row.Cells...)
	}
	seen := make(set.Set[string])
	for i, row := range old.Rows {
		cells := make([]Cell, 0, len(old.Columns))
		for _, key := range row.Cells[:min(len(row.Cells), len(old.Columns))] {
			c := Cell{Key: key}
			if seen.Contains(key) {
				c.Key = newKey(key, used)
			}
			seen.Add(c.Key)
			if cfg, ok := old.Cells[key]; ok {
				if c.Config, err = migrateCellConfig(ctx, cfg); err != nil {
					return Table{}, err
				}
			} else {
				c.Config = emptyConfig()
			}
			cells = append(cells, c)
		}
		for len(cells) < len(old.Columns) {
			key := newKey(t.Rows[i].Key, used)
			cells = append(cells, Cell{Key: key, Config: emptyConfig()})
		}
		t.Rows[i].Cells = cells
	}
	return t, nil
}

// migrateCellConfig lifts a v5 cell config to v6. Each color becomes a fixed color, and
// an absent color stays absent.
func migrateCellConfig(ctx context.Context, old v5.CellConfig) (CellConfig, error) {
	c, err := autoMigrateCellConfig(ctx, old)
	if err != nil {
		return CellConfig{}, err
	}
	switch o := old.Variant.(type) {
	case v5.TextCellConfig:
		v := as[TextCellConfig](c.Variant)
		if v.FillColor, err = fixedColor(ctx, o.FillColor); err != nil {
			return CellConfig{}, err
		}
		if v.TextColor, err = fixedColor(ctx, o.TextColor); err != nil {
			return CellConfig{}, err
		}
		c.Variant = v
	case v5.ValueCellConfig:
		v := as[ValueCellConfig](c.Variant)
		if v.FillColor, err = fixedColor(ctx, o.FillColor); err != nil {
			return CellConfig{}, err
		}
		if v.TextColor, err = fixedColor(ctx, o.TextColor); err != nil {
			return CellConfig{}, err
		}
		if v.StalenessColor, err = fixedColor(ctx, o.StalenessColor); err != nil {
			return CellConfig{}, err
		}
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

func fixedColor(ctx context.Context, old *xcolorv1.Color) (*color.Ref, error) {
	if old == nil {
		return nil, nil
	}
	c, err := xcolor.MigrateColor(ctx, *old)
	if err != nil {
		return nil, err
	}
	return &color.Ref{Variant: color.FixedRef{Value: c}}, nil
}

func emptyConfig() CellConfig {
	return CellConfig{Variant: defaults.Fill(TextCellConfig{})}
}

// newKey returns a key derived from base that used does not hold, and adds it to used.
func newKey(base string, used set.Set[string]) string {
	for n := 1; ; n++ {
		if k := fmt.Sprintf("%s-%d", base, n); !used.Contains(k) {
			used.Add(k)
			return k
		}
	}
}

// MigrateRow copies the row. MigrateTable sets its cells from the cells map.
func MigrateRow(ctx context.Context, old v5.Row) (Row, error) {
	return autoMigrateRow(ctx, old)
}

// Migration lifts stored tables from v5 to v6.
var Migration = migrate.NewEntry("v62_color_bytes_cells_in_rows", MigrateTable)

// NameMigration names each stored table that has an empty name "Table <key>", where
// <key> is the first eight characters of its key.
var NameMigration = migrate.NewEntry(
	"v62_require_name",
	func(_ context.Context, t Table) (Table, error) {
		if t.Name == "" {
			t.Name = "Table " + t.Key.String()[:8]
		}
		return t, nil
	},
)

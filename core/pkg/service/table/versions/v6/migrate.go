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
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/set"
)

// MigrateTable lifts a v5 table to v6. Every color alpha becomes a 0-255 byte, and each
// entry of the cells map moves into the row cell that references it. A row cell with no
// config gets an empty text cell, so it keeps its column. A cell that no row references
// is dropped. A key that more than one row cell references keeps its first cell, and
// each later cell gets a new key with a copy of the config. Each row then holds one
// cell per column: cells past the last column are dropped, and a short row gets empty
// text cells with new keys.
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
				if c.Config, err = autoMigrateCellConfig(ctx, cfg); err != nil {
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

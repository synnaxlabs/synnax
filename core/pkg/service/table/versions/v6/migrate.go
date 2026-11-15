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

	"github.com/synnaxlabs/gorp/migrate"
	v5 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v5"
)

// MigrateTable lifts a v5 table to v6. Every color alpha becomes a 0-255 byte, and each
// entry of the cells map moves into the row cell that references it. Pluto draws
// nothing for a row cell with no config, so it is dropped. A cell that no row
// references is dropped too.
func MigrateTable(ctx context.Context, old v5.Table) (Table, error) {
	t, err := autoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	for i, row := range old.Rows {
		cells := make([]Cell, 0, len(row.Cells))
		for _, key := range row.Cells {
			cfg, ok := old.Cells[key]
			if !ok {
				continue
			}
			c := Cell{Key: key}
			if c.Config, err = autoMigrateCellConfig(ctx, cfg); err != nil {
				return Table{}, err
			}
			cells = append(cells, c)
		}
		t.Rows[i].Cells = cells
	}
	return t, nil
}

// MigrateRow copies the row. MigrateTable sets its cells from the cells map.
func MigrateRow(ctx context.Context, old v5.Row) (Row, error) {
	return autoMigrateRow(ctx, old)
}

// Migration lifts stored tables from v5 to v6.
var Migration = migrate.NewEntry("v62_color_bytes_cells_in_rows", MigrateTable)

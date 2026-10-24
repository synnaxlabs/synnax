// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"strconv"

	v2 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v2"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateTable transforms the previous Table snapshot (v2) into the v3 Table, which
// gives every row and column a key. The keys derive from position, so every migration
// of the same table produces the same keys.
func MigrateTable(ctx context.Context, old v2.Table) (Table, error) {
	out, err := autoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	for i := range out.Rows {
		out.Rows[i].Key = "row-" + strconv.Itoa(i)
	}
	for i := range out.Columns {
		out.Columns[i].Key = "col-" + strconv.Itoa(i)
	}
	return out, nil
}

// MigrateRow lifts a v2 row to v3. MigrateTable assigns its key.
func MigrateRow(ctx context.Context, old v2.Row) (Row, error) {
	return autoMigrateRow(ctx, old)
}

// MigrateColumn lifts a v2 column to v3. MigrateTable assigns its key.
func MigrateColumn(ctx context.Context, old v2.Column) (Column, error) {
	return autoMigrateColumn(ctx, old)
}

// Migration gives every row and column of a stored table a key.
var Migration = gorp.NewEntryMigration("v3_row_and_column_keys", MigrateTable)

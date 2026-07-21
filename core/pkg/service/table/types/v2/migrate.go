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

	"github.com/synnaxlabs/synnax/pkg/service/table/types/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/types/legacy/v0"
	tablev0 "github.com/synnaxlabs/synnax/pkg/service/table/types/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateTable transforms the previous Table snapshot (v0) into the current
// strongly-typed Table. AutoMigrateTable handles the trivially-copyable
// gorp-entry fields (Key, Name); the structural fields (Rows, Columns, Cells)
// are sourced from the opaque blob the console used to persist alongside those
// fields, after legacy.MigrateData decodes it as v0.Data. v0 is the last
// snapshot in which Table.Data is untyped; future migrations transform one
// typed snapshot into another and never need this blob handling.
func MigrateTable(ctx context.Context, old tablev0.Table) (Table, error) {
	out, err := AutoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return Table{}, err
	}
	out.Rows = migrateRows(d.Layout.Rows)
	out.Columns = migrateColumns(d.Layout.Columns)
	out.Cells = migrateCells(d.Cells)
	return out, nil
}

func migrateRows(in []v0.Row) []Row {
	out := make([]Row, len(in))
	for i, r := range in {
		cells := make([]string, len(r.Cells))
		for j, c := range r.Cells {
			cells[j] = c.Key
		}
		out[i] = Row{Size: r.Size, Cells: cells}
	}
	return out
}

func migrateColumns(in []v0.Column) []Column {
	out := make([]Column, len(in))
	for i, c := range in {
		out[i] = Column{Size: c.Size}
	}
	return out
}

func migrateCells(in map[string]v0.Cell) map[string]Cell {
	out := make(map[string]Cell, len(in))
	for k, c := range in {
		out[k] = Cell{
			Key:     c.Key,
			Variant: c.Variant,
			Props:   msgpack.EncodedJSON(c.Props),
		}
	}
	return out
}

// codecMigrationKey names the codec migration the lift migration depends on.
const codecMigrationKey = "msgpack_to_orc"

// CodecMigration re-encodes stored tables from msgpack to orc. It is pinned to
// the v1 shape so its output stays stable as Table evolves.
var CodecMigration = gorp.CodecMigration[Key, tablev0.Table](codecMigrationKey)

// Migration lifts stored tables from the v1 blob layout to the typed v2 shape.
var Migration = gorp.NewEntryMigration[Key, Key, tablev0.Table, Table](
	"v55_lift_typed_table", MigrateTable, codecMigrationKey,
)

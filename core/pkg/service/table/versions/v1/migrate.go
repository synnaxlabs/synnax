// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/table/versions/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateTable transforms the previous Table snapshot (v0) into the v1 strongly-typed
// Table. autoMigrateTable handles the trivially-copyable Gorp-entry fields (Key, Name);
// the structural fields (Rows, Columns, Cells) are sourced from the opaque blob the
// Console used to persist alongside those fields, after legacy.MigrateData decodes it
// as legacy.Data. v0 is the last snapshot in which Table.Data is untyped; future
// migrations transform one typed snapshot into another and never need this blob
// handling.
func MigrateTable(ctx context.Context, old v0.Table) (Table, error) {
	out, err := autoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return Table{}, err
	}
	out.Rows = lo.Map(d.Layout.Rows, func(r legacy.Row, _ int) Row {
		return Row{
			Size: r.Size,
			Cells: lo.Map(r.Cells, func(c legacy.CellRef, _ int) string {
				return c.Key
			}),
		}
	})
	out.Columns = lo.Map(d.Layout.Columns, func(c legacy.Column, _ int) Column {
		return Column{Size: c.Size}
	})
	out.Cells = lo.MapValues(d.Cells, func(c legacy.Cell, _ string) Cell {
		return Cell{
			Key:     c.Key,
			Variant: c.Variant,
			Props:   msgpack.EncodedJSON(c.Props),
		}
	})
	return out, nil
}

// Migration lifts stored tables from the v0 blob layout to the typed v1 shape.
var Migration = gorp.NewEntryMigration("v55_lift_typed_table", MigrateTable)

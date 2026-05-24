// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"

	v55 "github.com/synnaxlabs/synnax/pkg/service/table/migrations/v55"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// blobData is the wire shape of the pre-typed Table.Data field, mirroring the
// console persisted state at console/src/table/types/v0.ts. Only the
// structural fields (layout, cells) are decoded; UI-only fields the console
// happens to ship in the blob (version, lastSelected, editable, remoteCreated,
// per-cell selected) are ignored and never materialize on the typed Table.
type blobData struct {
	Layout struct {
		Rows []struct {
			Size  float64       `json:"size"`
			Cells []blobCellRef `json:"cells"`
		} `json:"rows"`
		Columns []struct {
			Size float64 `json:"size"`
		} `json:"columns"`
	} `json:"layout"`
	Cells map[string]blobCell `json:"cells"`
}

type blobCellRef struct {
	Key string `json:"key"`
}

type blobCell struct {
	Key     string         `json:"key"`
	Variant string         `json:"variant"`
	Props   map[string]any `json:"props"`
}

// MigrateTable transforms the previous Table snapshot (v55) into the current
// strongly-typed Table. AutoMigrateTable handles the trivially-copyable
// gorp-entry fields (Key, Name); the structural fields (Rows, Columns, Cells)
// are sourced from the opaque blob the console used to persist alongside
// those fields. v55 is the last snapshot in which Table.Data is untyped;
// future migrations transform one typed snapshot into another and never need
// this blob handling.
func MigrateTable(ctx context.Context, old v55.Table) (Table, error) {
	out, err := AutoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	out.Rows = []Row{}
	out.Columns = []Column{}
	out.Cells = map[string]Cell{}
	if len(old.Data) == 0 {
		return out, nil
	}
	var d blobData
	if err := old.Data.Unmarshal(&d); err != nil {
		return Table{}, errors.Wrap(err, "decode legacy table data")
	}
	out.Rows = make([]Row, len(d.Layout.Rows))
	for i, r := range d.Layout.Rows {
		cells := make([]string, len(r.Cells))
		for j, c := range r.Cells {
			cells[j] = c.Key
		}
		out.Rows[i] = Row{Size: r.Size, Cells: cells}
	}
	out.Columns = make([]Column, len(d.Layout.Columns))
	for i, c := range d.Layout.Columns {
		out.Columns[i] = Column{Size: c.Size}
	}
	out.Cells = make(map[string]Cell, len(d.Cells))
	for k, c := range d.Cells {
		out.Cells[k] = Cell{
			Key:     c.Key,
			Variant: c.Variant,
			Props:   msgpack.EncodedJSON(c.Props),
		}
	}
	return out, nil
}

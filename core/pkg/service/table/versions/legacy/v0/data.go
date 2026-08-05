// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for Console table per-table state at version
// 0.0.0. Per-version Data structs in this directory tree are immutable snapshots of
// what Consoles actually persisted at that version. They are the JSON-decode targets
// for the storage migration chain that lifts older blobs forward into the typed
// table.Table.
package v0

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// Version is the version the Console stamped on this format.
const Version imex.Version = 0

// Data is the persisted per-table Console state at version 0.0.0. Only the structural
// fields (layout, cells) are decoded; UI-only fields the Console ships in the blob are
// ignored and never materialize on the typed Table.
type Data struct {
	// Layout is the row and column geometry.
	Layout Layout `json:"layout"`
	// Cells holds every cell body, keyed by cell key.
	Cells map[string]Cell `json:"cells"`
}

// Layout is the row/column geometry of the table.
type Layout struct {
	// Rows are the table rows, top to bottom.
	Rows []Row `json:"rows"`
	// Columns are the table columns, left to right.
	Columns []Column `json:"columns"`
}

// Row is a single horizontal slice of the table, holding ordered references to the
// cells it contains.
type Row struct {
	// Size is the row height in pixels.
	Size float64 `json:"size"`
	// Cells are ordered references to the cells in this row.
	Cells []CellRef `json:"cells"`
}

// Column is the width descriptor for a single column slot.
type Column struct {
	// Size is the column width in pixels.
	Size float64 `json:"size"`
}

// CellRef is the per-row pointer to a cell entry in Data.Cells. The Console serialized
// this as an object so it could carry transient UI flags (selected); only the key
// survives the migration.
type CellRef struct {
	// Key points at an entry in Data.Cells.
	Key string `json:"key"`
}

// Cell is the persisted body of a single cell.
type Cell struct {
	// Key is the cell's unique key within the table.
	Key string `json:"key"`
	// Variant selects the cell component (e.g. "text", "value").
	Variant string `json:"variant"`
	// Props is the variant-specific cell configuration.
	Props map[string]any `json:"props"`
}

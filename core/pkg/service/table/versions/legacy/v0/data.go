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
// what consoles actually persisted at that version. They are the JSON-decode targets
// for the storage migration chain that lifts older blobs forward into the typed
// table.Table.
package v0

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// Version is the ImEx schema version of table data at this state. The Console stamped
// it on the wire as the semver string "0.0.0", which legacy.MigrateData decodes onto
// this numeric version.
const Version imex.Version = 0

// Data mirrors the Console persisted state at console/src/table/types/v0.ts. Only the
// structural fields (layout, cells) are decoded; UI-only fields the console happens to
// ship in the blob (version, lastSelected, editable, remoteCreated, per-cell selected)
// are ignored and never materialize on the typed Table.
type Data struct {
	Layout Layout          `json:"layout"`
	Cells  map[string]Cell `json:"cells"`
}

// Layout is the row/column geometry of the table.
type Layout struct {
	Rows    []Row    `json:"rows"`
	Columns []Column `json:"columns"`
}

// Row is a single horizontal slice of the table, holding ordered references to the
// cells it contains.
type Row struct {
	Size  float64   `json:"size"`
	Cells []CellRef `json:"cells"`
}

// Column is the width descriptor for a single column slot.
type Column struct {
	Size float64 `json:"size"`
}

// CellRef is the per-row pointer to a cell entry in Data.Cells. The Console serialized
// this as an object so it could carry transient UI flags (selected); only the key
// survives the migration.
type CellRef struct {
	Key string `json:"key"`
}

// Cell is the persisted body of a single cell.
type Cell struct {
	Key     string         `json:"key"`
	Variant string         `json:"variant"`
	Props   map[string]any `json:"props"`
}

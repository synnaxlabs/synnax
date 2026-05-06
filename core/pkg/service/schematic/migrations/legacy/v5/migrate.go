// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v4"
)

// Migrate transforms v4 schematic data into v5. The console drops the type
// literal in this step and seeds the new per-schematic mode and toolbar UI
// state. Mode and Toolbar are UI-only and are dropped when the v6 wire form
// is lifted into the typed schematic.Schematic.
func Migrate(old v4.Data) Data {
	return Data{
		Version:         Version,
		Editable:        old.Editable,
		FitViewOnResize: old.FitViewOnResize,
		Snapshot:        old.Snapshot,
		RemoteCreated:   old.RemoteCreated,
		Viewport:        old.Viewport,
		Nodes:           old.Nodes,
		Edges:           old.Edges,
		Props:           old.Props,
		Control:         old.Control,
		Legend:          old.Legend,
		Key:             old.Key,
		ViewportMode:    old.ViewportMode,
		Authority:       old.Authority,
		Mode:            "select",
		Toolbar: v0.ToolbarState{
			ActiveTab:           "symbols",
			SelectedSymbolGroup: "general",
		},
	}
}

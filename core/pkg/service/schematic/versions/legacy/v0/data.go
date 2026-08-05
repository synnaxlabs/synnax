// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for Console schematic state at version 0.0.0.
// Per-version Data structs in this directory tree are immutable snapshots of what
// Consoles actually persisted at that version. They are the JSON-decode targets for the
// storage migration chain that lifts older blobs forward into the typed
// schematic.Schematic.
package v0

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 0

// XY is the planar coordinate shape Consoles persisted from version 0.0.0 onward.
// Identical to spatial.XY at the wire level.
type XY struct {
	// X is the horizontal coordinate.
	X float64 `json:"x"`
	// Y is the vertical coordinate.
	Y float64 `json:"y"`
}

// Measured holds the optional rendered dimensions of a node. Both fields are optional
// in the wire format and may be absent.
type Measured struct {
	// Width is the rendered width in pixels; absent when unmeasured.
	Width *float64 `json:"width,omitempty"`
	// Height is the rendered height in pixels; absent when unmeasured.
	Height *float64 `json:"height,omitempty"`
}

// Node is the wire shape of a schematic node at version 0.0.0.
type Node struct {
	// Key is the node's unique key.
	Key string `json:"key"`
	// Position is the node position on the canvas.
	Position XY `json:"position"`
	// ZIndex is the node stacking order.
	ZIndex *int `json:"zIndex,omitempty"`
	// Type is the ReactFlow node type; empty for default.
	Type string `json:"type,omitempty"`
	// Measured is the optional rendered size.
	Measured *Measured `json:"measured,omitempty"`
}

// Edge is the wire shape of a schematic edge at version 0.0.0. Shipped Console
// persisted edges in ReactFlow's flat form: source and target are node-key strings,
// with sourceHandle and targetHandle as optional sibling fields. Data is ReactFlow's
// per-edge data bag; Consoles have written it on every shipped version including v0..v2
// even though the v3 schema is the first to declare it explicitly. Preserving it here
// is what lets the v5 to v6 migration lift segments / color / variant out of older
// blobs without losing fidelity. The nested Handle{Node, Param} representation used by
// the typed Schematic is constructed during the v5 to v6 migration step.
type Edge struct {
	// Key is the edge's unique key.
	Key string `json:"key"`
	// Source is the source node key.
	Source string `json:"source"`
	// Target is the target node key.
	Target string `json:"target"`
	// SourceHandle is the optional source handle identifier.
	SourceHandle *string `json:"sourceHandle,omitempty"`
	// TargetHandle is the optional target handle identifier.
	TargetHandle *string `json:"targetHandle,omitempty"`
	// Data is ReactFlow's opaque per-edge data bag.
	Data json.RawMessage `json:"data,omitempty"`
}

// Viewport is the schematic editor's viewport position and zoom.
type Viewport struct {
	// Position is the viewport pan offset.
	Position XY `json:"position"`
	// Zoom is the viewport zoom factor.
	Zoom float64 `json:"zoom"`
}

// ToolbarState is the per-schematic toolbar UI state introduced at v5 in the Console.
// The field is declared at v0 because it is referenced unchanged by later versions.
type ToolbarState struct {
	// ActiveTab is the selected toolbar tab.
	ActiveTab string `json:"activeTab"`
	// SelectedSymbolGroup is the selected symbol group.
	SelectedSymbolGroup string `json:"selectedSymbolGroup"`
}

// Data is the persisted per-schematic state at version 0.0.0. Props values are kept as
// raw JSON because their shape is per-symbol-variant and opaque to the server.
type Data struct {
	// Version is the version stamped inside the blob.
	Version imex.Version `json:"version"`
	// Editable is UI-only edit-mode state; dropped on lift.
	Editable bool `json:"editable"`
	// FitViewOnResize is UI-only viewport behavior; dropped on lift.
	FitViewOnResize bool `json:"fitViewOnResize"`
	// Snapshot marks the schematic as a range snapshot.
	Snapshot bool `json:"snapshot"`
	// RemoteCreated is UI-only sync bookkeeping; dropped on lift.
	RemoteCreated bool `json:"remoteCreated"`
	// Viewport is the editor viewport position and zoom.
	Viewport Viewport `json:"viewport"`
	// Nodes are the schematic nodes.
	Nodes []Node `json:"nodes"`
	// Edges are the schematic edges.
	Edges []Edge `json:"edges"`
	// Props holds per-symbol configuration keyed by node key.
	Props map[string]json.RawMessage `json:"props"`
	// Control is UI-only control-mode state; dropped on lift.
	Control string `json:"control"`
}

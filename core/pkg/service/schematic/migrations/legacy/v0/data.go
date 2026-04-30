// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for console schematic state at version
// 0.0.0. Per-version Data structs in this directory tree are immutable snapshots
// of what consoles actually persisted at that version. They are the JSON-decode
// targets for the storage migration chain that lifts older blobs forward into
// the typed schematic.Schematic.
package v0

import "encoding/json"

// Version is the semantic version string written by the console at this state
// version.
const Version = "0.0.0"

// XY is the planar coordinate shape consoles persisted from version 0.0.0
// onward. Identical to spatial.XY at the wire level.
type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Measured holds the optional rendered dimensions of a node. Both fields are
// optional in the wire format and may be absent.
type Measured struct {
	Width  *float64 `json:"width,omitempty"`
	Height *float64 `json:"height,omitempty"`
}

// Node is the wire shape of a schematic node at version 0.0.0.
type Node struct {
	Key      string    `json:"key"`
	Position XY        `json:"position"`
	ZIndex   *int      `json:"zIndex,omitempty"`
	Type     string    `json:"type,omitempty"`
	Measured *Measured `json:"measured,omitempty"`
}

// Edge is the wire shape of a schematic edge at version 0.0.0. Shipped consoles
// persisted edges in ReactFlow's flat form: source and target are node-key
// strings, with sourceHandle and targetHandle as optional sibling fields. The
// nested Handle{Node, Param} representation used by the typed Schematic is
// constructed during the v5 to v6 migration step.
type Edge struct {
	Key          string  `json:"key"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	SourceHandle *string `json:"sourceHandle,omitempty"`
	TargetHandle *string `json:"targetHandle,omitempty"`
}

// Viewport is the schematic editor's viewport position and zoom.
type Viewport struct {
	Position XY      `json:"position"`
	Zoom     float64 `json:"zoom"`
}

// ToolbarState is the per-schematic toolbar UI state introduced at v5 in the
// console. The field is declared at v0 because it is referenced unchanged by
// later versions.
type ToolbarState struct {
	ActiveTab           string `json:"activeTab"`
	SelectedSymbolGroup string `json:"selectedSymbolGroup"`
}

// Data is the persisted per-schematic state at version 0.0.0. Props values are
// kept as raw JSON because their shape is per-symbol-variant and opaque to the
// server.
type Data struct {
	Version         string                     `json:"version"`
	Editable        bool                       `json:"editable"`
	FitViewOnResize bool                       `json:"fitViewOnResize"`
	Snapshot        bool                       `json:"snapshot"`
	RemoteCreated   bool                       `json:"remoteCreated"`
	Viewport        Viewport                   `json:"viewport"`
	Nodes           []Node                     `json:"nodes"`
	Edges           []Edge                     `json:"edges"`
	Props           map[string]json.RawMessage `json:"props"`
	Control         string                     `json:"control"`
}

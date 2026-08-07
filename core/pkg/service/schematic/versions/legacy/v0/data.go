// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for Console schematic state at version 0.
// Per-version Data structs in this directory tree are immutable snapshots of what
// Consoles actually persisted at that version. They are the JSON-decode targets for the
// storage migration chain that lifts older blobs forward into the typed
// schematic.Schematic.
package v0

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 0

// Node is the wire shape of a schematic node at version 0.
type Node struct {
	// Key is the node's unique key.
	Key string `json:"key"`
	// Position is the node position on the canvas.
	Position spatial.XY `json:"position"`
	// ZIndex is the node stacking order.
	ZIndex *int `json:"zIndex,omitempty"`
}

// Edge is the wire shape of a schematic edge at version 0. Shipped Console persisted
// edges in ReactFlow's flat form: source and target are node-key strings, with
// sourceHandle and targetHandle as optional sibling fields. Data is ReactFlow's
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

// Data is the persisted per-schematic state at version 0. Props values are kept as raw
// JSON because their shape is per-symbol-variant and opaque to the server.
type Data struct {
	// Version is the version stamped inside the blob.
	Version imex.Version `json:"version"`
	// Snapshot marks the schematic as a range snapshot.
	Snapshot bool `json:"snapshot"`
	// Nodes are the schematic nodes.
	Nodes []Node `json:"nodes"`
	// Edges are the schematic edges.
	Edges []Edge `json:"edges"`
	// Props holds per-symbol configuration keyed by node key.
	Props map[string]json.RawMessage `json:"props"`
}

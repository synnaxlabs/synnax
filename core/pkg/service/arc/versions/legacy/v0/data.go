// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for Console arc state at version 0.0.0.
// v0 persists edges in ReactFlow's flat form: node-key strings with optional
// sibling handle fields.
package v0

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/spatial"
)

// Version is the ImEx schema version of arc state at this version. The Console
// stamped it on the wire as the semver string "0.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 0

// Data is the wire shape of a Console arc state at version 0.0.0. Mode is the raw
// wire string ("graph" or "text"); the importer converts it to the typed enum.
type Data struct {
	// Graph is the graph-mode program body.
	Graph Graph `json:"graph"`
	// Text is the text-mode program body.
	Text Text `json:"text"`
	// Mode selects graph or text mode.
	Mode string `json:"mode"`
}

// Text is the raw text-mode program body.
type Text struct {
	// Raw is the raw program source.
	Raw string `json:"raw"`
}

// Graph is the graph-mode program body: nodes, edges, and per-node props.
type Graph struct {
	// Nodes are the graph nodes.
	Nodes []Node `json:"nodes"`
	// Edges are the graph edges.
	Edges []Edge `json:"edges"`
	// Props holds per-node function props keyed by node key.
	Props map[string]map[string]any `json:"props"`
}

// Node is the wire shape of a graph node.
type Node struct {
	// Key is the node's unique key.
	Key string `json:"key"`
	// Position is the node position on the canvas.
	Position spatial.XY `json:"position"`
}

// Edge is the v0 edge form: ReactFlow's node-key strings with optional sibling
// handle fields.
type Edge struct {
	// Key is the edge's unique key.
	Key string `json:"key"`
	// Source is the source node key.
	Source string `json:"source"`
	// Target is the target node key.
	Target string `json:"target"`
	// SourceHandle is the optional source param name.
	SourceHandle *string `json:"sourceHandle"`
	// TargetHandle is the optional target param name.
	TargetHandle *string `json:"targetHandle"`
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "encoding/json"

// Version is the numeric version for console schematic state v0.0.0.
// Computed via legacyToNumeric: 0*5 + 0*2 + 0 = 0.
const Version = 0

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Dimensions struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type NodeData struct {
	Key      string     `json:"key"`
	Position XY         `json:"position"`
	Measured Dimensions `json:"measured"`
}

// EdgeData is the wire shape of a schematic edge at version 0.0.0. Shipped
// consoles persisted edges in ReactFlow's flat form: source and target are
// node-key strings, with sourceHandle and targetHandle as optional sibling
// fields. The nested Handle{Node, Param} representation used by the server's
// Schematic struct is built from these flat fields by convertToSchematic.
type EdgeData struct {
	Key          string `json:"key"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

// Data holds schematic content at version 0.0.0. Props values are kept as
// raw JSON bytes since their shape is per-variant and opaque to the server.
type Data struct {
	Nodes         []NodeData                 `json:"nodes"`
	Edges         []EdgeData                 `json:"edges"`
	Props         map[string]json.RawMessage `json:"props"`
	Snapshot      bool                       `json:"snapshot"`
	RemoteCreated bool                       `json:"remote_created"`
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v3 holds the frozen wire format for console schematic state at version
// 3.0.0. v3 added segments[] to each edge.
package v3

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
)

const Version = "3.0.0"

// Segment is one orthogonal hop on an edge connector path.
type Segment struct {
	Direction string  `json:"direction"`
	Length    float64 `json:"length"`
}

// Edge is the wire shape of a schematic edge at version 3.0.0. Inherits the
// flat source/target form from v0, adds segments at the top level, and carries
// a catch-all Data field that captured ReactFlow's per-edge data bag in
// shipped consoles. The v6 migration reads segments, color, and variant out
// of either location and lifts them into the props map.
type Edge struct {
	Key          string          `json:"key"`
	Source       string          `json:"source"`
	Target       string          `json:"target"`
	SourceHandle *string         `json:"sourceHandle,omitempty"`
	TargetHandle *string         `json:"targetHandle,omitempty"`
	Segments     []Segment       `json:"segments,omitempty"`
	Data         json.RawMessage `json:"data,omitempty"`
}

// Data is the persisted per-schematic state at version 3.0.0.
type Data struct {
	Version         string                     `json:"version"`
	Editable        bool                       `json:"editable"`
	FitViewOnResize bool                       `json:"fitViewOnResize"`
	Snapshot        bool                       `json:"snapshot"`
	RemoteCreated   bool                       `json:"remoteCreated"`
	Viewport        v0.Viewport                `json:"viewport"`
	Nodes           []v0.Node                  `json:"nodes"`
	Edges           []Edge                     `json:"edges"`
	Props           map[string]json.RawMessage `json:"props"`
	Control         string                     `json:"control"`
	Legend          v1.Legend                  `json:"legend"`
	Key             string                     `json:"key"`
	Type            string                     `json:"type"`
	ViewportMode    string                     `json:"viewportMode"`
}

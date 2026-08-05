// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v3 holds the frozen wire format for Console schematic state at version 3.0.0.
// v3 added segments[] to each edge.
package v3

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
)

// Version is the ImEx schema version of schematic data at this state. The Console
// stamped it on the wire as the semver string "3.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 3

// Segment is one orthogonal hop on an edge connector path.
type Segment struct {
	// Direction is the hop direction.
	Direction string `json:"direction"`
	// Length is the hop length in pixels.
	Length float64 `json:"length"`
}

// Edge is the wire shape of a schematic edge at version 3.0.0. Inherits the flat
// source/target form from v0, adds segments at the top level, and carries a catch-all
// Data field that captured ReactFlow's per-edge data bag in shipped Consoles. The v6
// migration reads segments, color, and variant out of either location and lifts them
// into the props map.
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
	// Segments is the orthogonal connector path; empty for direct.
	Segments []Segment `json:"segments,omitempty"`
	// Data is ReactFlow's opaque per-edge data bag.
	Data json.RawMessage `json:"data,omitempty"`
}

// Data is the persisted per-schematic state at version 3.0.0.
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
	Viewport v0.Viewport `json:"viewport"`
	// Nodes are the schematic nodes.
	Nodes []v0.Node `json:"nodes"`
	// Edges are the schematic edges.
	Edges []Edge `json:"edges"`
	// Props holds per-symbol configuration keyed by node key.
	Props map[string]json.RawMessage `json:"props"`
	// Control is UI-only control-mode state; dropped on lift.
	Control string `json:"control"`
	// Legend is the control legend overlay configuration.
	Legend v1.Legend `json:"legend"`
	// Key is the Console-local schematic key.
	Key string `json:"key"`
	// Type is the literal "schematic" type marker.
	Type string `json:"type"`
	// ViewportMode is UI-only viewport interaction mode; dropped on lift.
	ViewportMode string `json:"viewportMode"`
}

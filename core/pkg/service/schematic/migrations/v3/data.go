// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v1"
)

// Version is the numeric version for console schematic state v3.0.0.
// Computed via legacyToNumeric: 3*5 + 0*2 + 0 = 15.
const Version = 15

type SegmentData struct {
	Direction string  `json:"direction"`
	Length    float64 `json:"length"`
}

// EdgeData extends the base flat edge (see v0.EdgeData) with orthogonal path
// segments. Source/target/handles remain flat strings — the diagram nested
// handle shape is a server-internal representation, not part of the wire
// format at any shipped state version.
type EdgeData struct {
	Key          string        `json:"key"`
	Source       string        `json:"source"`
	Target       string        `json:"target"`
	SourceHandle string        `json:"sourceHandle,omitempty"`
	TargetHandle string        `json:"targetHandle,omitempty"`
	Segments     []SegmentData `json:"segments"`
}

// Data holds schematic content at version 3.0.0. Edges now include segments.
type Data struct {
	Nodes         []v0.NodeData              `json:"nodes"`
	Edges         []EdgeData                 `json:"edges"`
	Props         map[string]json.RawMessage `json:"props"`
	Legend        v1.LegendData              `json:"legend"`
	Snapshot      bool                       `json:"snapshot"`
	RemoteCreated bool                       `json:"remote_created"`
	Key           string                     `json:"key"`
	Type          string                     `json:"type"`
}

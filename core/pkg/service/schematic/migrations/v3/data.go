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
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v1"
	"github.com/synnaxlabs/x/zyn"
)

// Version is the numeric version for console schematic state v3.0.0.
// Computed via legacyToNumeric: 3*5 + 0*2 + 0 = 15.
const Version = 15

type SegmentData struct {
	Direction string  `json:"direction"`
	Length    float64 `json:"length"`
}

// EdgeData extends the base edge with segments for orthogonal path routing.
type EdgeData struct {
	Key      string        `json:"key"`
	Source   v0.HandleData `json:"source"`
	Target   v0.HandleData `json:"target"`
	Segments []SegmentData `json:"segments"`
}

// Data holds schematic content at version 3.0.0. Edges now include segments.
type Data struct {
	Nodes         []v0.NodeData  `json:"nodes"`
	Edges         []EdgeData     `json:"edges"`
	Props         map[string]any `json:"props"`
	Legend        v1.LegendData  `json:"legend"`
	Snapshot      bool           `json:"snapshot"`
	RemoteCreated bool           `json:"remote_created"`
	Key           string         `json:"key"`
	Type          string         `json:"type"`
}

var segmentSchema = zyn.Object(map[string]zyn.Schema{
	"direction": zyn.String(),
	"length":    zyn.Number(),
})

var EdgeSchema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.String(),
	"source":   v0.HandleSchema,
	"target":   v0.HandleSchema,
	"segments": zyn.Array(segmentSchema).Optional(),
})

var Schema = zyn.Object(map[string]zyn.Schema{
	"nodes":          zyn.Array(v0.NodeSchema).Optional(),
	"edges":          zyn.Array(EdgeSchema).Optional(),
	"props":          zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{})).Optional(),
	"legend":         v1.LegendSchema.Optional(),
	"snapshot":       zyn.Bool().Optional(),
	"remote_created": zyn.Bool().Optional(),
	"key":            zyn.String().Optional(),
	"type":           zyn.String().Optional(),
})

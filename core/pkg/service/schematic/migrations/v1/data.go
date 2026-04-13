// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	"github.com/synnaxlabs/x/zyn"
)

// Version is the numeric version for console schematic state v1.0.0.
// Computed via legacyToNumeric: 1*5 + 0*2 + 0 = 5.
const Version = 5

type LegendPosition struct {
	X     float64            `json:"x"`
	Y     float64            `json:"y"`
	Units map[string]string  `json:"units"`
	Root  map[string]string  `json:"root"`
}

type LegendData struct {
	Visible  bool              `json:"visible"`
	Position LegendPosition    `json:"position"`
	Colors   map[string]string `json:"colors"`
}

// Data holds schematic content at version 1.0.0. Adds a structured legend.
type Data struct {
	Nodes         []v0.NodeData  `json:"nodes"`
	Edges         []v0.EdgeData  `json:"edges"`
	Props         map[string]any `json:"props"`
	Legend        LegendData     `json:"legend"`
	Snapshot      bool           `json:"snapshot"`
	RemoteCreated bool           `json:"remote_created"`
}

var LegendPositionSchema = zyn.Object(map[string]zyn.Schema{
	"x":     zyn.Number(),
	"y":     zyn.Number(),
	"units": zyn.Map(zyn.String(), zyn.String()).Optional(),
	"root":  zyn.Map(zyn.String(), zyn.String()).Optional(),
})

var LegendSchema = zyn.Object(map[string]zyn.Schema{
	"visible":  zyn.Bool().Optional(),
	"position": LegendPositionSchema.Optional(),
	"colors":   zyn.Map(zyn.String(), zyn.String()).Optional(),
})

var Schema = zyn.Object(map[string]zyn.Schema{
	"nodes":          zyn.Array(v0.NodeSchema).Optional(),
	"edges":          zyn.Array(v0.EdgeSchema).Optional(),
	"props":          zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{})).Optional(),
	"legend":         LegendSchema.Optional(),
	"snapshot":       zyn.Bool().Optional(),
	"remote_created": zyn.Bool().Optional(),
})

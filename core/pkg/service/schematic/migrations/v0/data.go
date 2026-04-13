// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/x/zyn"

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

type HandleData struct {
	Node  string `json:"node"`
	Param string `json:"param"`
}

type EdgeData struct {
	Key    string     `json:"key"`
	Source HandleData `json:"source"`
	Target HandleData `json:"target"`
}

// Data holds schematic content at version 0.0.0.
type Data struct {
	Nodes         []NodeData     `json:"nodes"`
	Edges         []EdgeData     `json:"edges"`
	Props         map[string]any `json:"props"`
	Snapshot      bool           `json:"snapshot"`
	RemoteCreated bool           `json:"remote_created"`
}

var xySchema = zyn.Object(map[string]zyn.Schema{
	"x": zyn.Number(),
	"y": zyn.Number(),
})

var dimensionsSchema = zyn.Object(map[string]zyn.Schema{
	"width":  zyn.Number(),
	"height": zyn.Number(),
}).Optional()

var NodeSchema = zyn.Object(map[string]zyn.Schema{
	"key":      zyn.String(),
	"position": xySchema,
	"measured": dimensionsSchema,
})

var HandleSchema = zyn.Object(map[string]zyn.Schema{
	"node":  zyn.String(),
	"param": zyn.String(),
})

var EdgeSchema = zyn.Object(map[string]zyn.Schema{
	"key":    zyn.String(),
	"source": HandleSchema,
	"target": HandleSchema,
})

var Schema = zyn.Object(map[string]zyn.Schema{
	"nodes":          zyn.Array(NodeSchema).Optional(),
	"edges":          zyn.Array(EdgeSchema).Optional(),
	"props":          zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{})).Optional(),
	"snapshot":       zyn.Bool().Optional(),
	"remote_created": zyn.Bool().Optional(),
})

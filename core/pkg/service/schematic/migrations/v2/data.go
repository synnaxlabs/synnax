// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v1"
	"github.com/synnaxlabs/x/zyn"
)

// Version is the numeric version for console schematic state v2.0.0.
// Computed via legacyToNumeric: 2*5 + 0*2 + 0 = 10.
const Version = 10

// Data holds schematic content at version 2.0.0. Adds key and type fields
// (UI-only, not used for server content).
type Data struct {
	Nodes         []v0.NodeData  `json:"nodes"`
	Edges         []v0.EdgeData  `json:"edges"`
	Props         map[string]any `json:"props"`
	Legend        v1.LegendData  `json:"legend"`
	Snapshot      bool           `json:"snapshot"`
	RemoteCreated bool           `json:"remote_created"`
	Key           string         `json:"key"`
	Type          string         `json:"type"`
}

var Schema = zyn.Object(map[string]zyn.Schema{
	"nodes":          zyn.Array(v0.NodeSchema).Optional(),
	"edges":          zyn.Array(v0.EdgeSchema).Optional(),
	"props":          zyn.Map(zyn.String(), zyn.Object(map[string]zyn.Schema{})).Optional(),
	"legend":         v1.LegendSchema.Optional(),
	"snapshot":       zyn.Bool().Optional(),
	"remote_created": zyn.Bool().Optional(),
	"key":            zyn.String().Optional(),
	"type":           zyn.String().Optional(),
})

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for console schematic state at version
// 1.0.0. v1 introduces the legend overlay configuration.
package v1

import (
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
)

const Version = "1.0.0"

// LegendUnits is the optional axis-by-axis unit hint for a legend position. May
// be absent in the wire format.
type LegendUnits struct {
	X string `json:"x"`
	Y string `json:"y"`
}

// LegendRoot is the optional axis-by-axis root anchor for a legend position.
type LegendRoot struct {
	X string `json:"x"`
	Y string `json:"y"`
}

// LegendPosition is the sticky position of the legend within the schematic.
type LegendPosition struct {
	X     float64      `json:"x"`
	Y     float64      `json:"y"`
	Units *LegendUnits `json:"units,omitempty"`
	Root  *LegendRoot  `json:"root,omitempty"`
}

// Legend is the control legend overlay configuration introduced at v1.
type Legend struct {
	Visible  bool              `json:"visible"`
	Position LegendPosition    `json:"position"`
	Colors   map[string]string `json:"colors"`
}

// Data is the persisted per-schematic state at version 1.0.0. Adds the legend
// field to v0.
type Data struct {
	Version         string                     `json:"version"`
	Editable        bool                       `json:"editable"`
	FitViewOnResize bool                       `json:"fitViewOnResize"`
	Snapshot        bool                       `json:"snapshot"`
	RemoteCreated   bool                       `json:"remoteCreated"`
	Viewport        v0.Viewport                `json:"viewport"`
	Nodes           []v0.Node                  `json:"nodes"`
	Edges           []v0.Edge                  `json:"edges"`
	Props           map[string]json.RawMessage `json:"props"`
	Control         string                     `json:"control"`
	Legend          Legend                     `json:"legend"`
}

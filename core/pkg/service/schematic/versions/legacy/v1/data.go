// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for Console schematic state at version 1.0.0.
// v1 introduces the legend overlay configuration.
package v1

import (
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 1

// LegendUnits is the optional axis-by-axis unit hint for a legend position. May be
// absent in the wire format.
type LegendUnits struct {
	// X is the horizontal unit hint.
	X string `json:"x"`
	// Y is the vertical unit hint.
	Y string `json:"y"`
}

// LegendRoot is the optional axis-by-axis root anchor for a legend position.
type LegendRoot struct {
	// X is the horizontal root anchor.
	X string `json:"x"`
	// Y is the vertical root anchor.
	Y string `json:"y"`
}

// LegendPosition is the sticky position of the legend within the schematic.
type LegendPosition struct {
	// X is the horizontal legend position.
	X float64 `json:"x"`
	// Y is the vertical legend position.
	Y float64 `json:"y"`
	// Units is the optional unit hint for the position.
	Units *LegendUnits `json:"units,omitempty"`
	// Root is the optional root anchor for the position.
	Root *LegendRoot `json:"root,omitempty"`
}

// Legend is the control legend overlay configuration introduced at v1.
type Legend struct {
	// Visible toggles legend display.
	Visible bool `json:"visible"`
	// Position is the sticky legend position.
	Position LegendPosition `json:"position"`
	// Colors maps legend entries to hex color strings.
	Colors map[string]string `json:"colors"`
}

// Data is the persisted per-schematic state at version 1.0.0. Adds the legend field to
// v0.
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
	Edges []v0.Edge `json:"edges"`
	// Props holds per-symbol configuration keyed by node key.
	Props map[string]json.RawMessage `json:"props"`
	// Control is UI-only control-mode state; dropped on lift.
	Control string `json:"control"`
	// Legend is the control legend overlay configuration.
	Legend Legend `json:"legend"`
}

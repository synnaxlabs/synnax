// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for Console line plot per-plot state at
// version 1.0.0. v1 introduces the legend overlay position.
package v1

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v0"
)

// Version is the ImEx schema version of line plot data at this state. The Console
// stamped it on the wire as the semver string "1.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 1

// StickyUnits is the optional axis-by-axis unit hint for a legend position.
type StickyUnits struct {
	// X is the horizontal unit hint.
	X string `json:"x"`
	// Y is the vertical unit hint.
	Y string `json:"y"`
}

// StickyRoot is the optional axis-by-axis root anchor for a legend position.
type StickyRoot struct {
	// X is the horizontal root anchor.
	X string `json:"x"`
	// Y is the vertical root anchor.
	Y string `json:"y"`
}

// LegendPosition is the sticky position of the legend within the plot.
type LegendPosition struct {
	// X is the horizontal legend position.
	X float64 `json:"x"`
	// Y is the vertical legend position.
	Y float64 `json:"y"`
	// Units is the optional unit hint for the position.
	Units *StickyUnits `json:"units,omitempty"`
	// Root is the optional root anchor for the position.
	Root *StickyRoot `json:"root,omitempty"`
}

// Legend is the plot legend configuration at v1. v1 adds Position.
type Legend struct {
	// Visible toggles legend display.
	Visible bool `json:"visible"`
	// Position is the sticky legend position.
	Position LegendPosition `json:"position"`
}

// Data is the wire shape of a per-plot line plot state at v1.0.0. All fields
// other than Legend are unchanged from v0.
type Data struct {
	// Version is the version stamped inside the blob.
	Version imex.Version `json:"version"`
	// Key is the Console-local plot key.
	Key string `json:"key"`
	// RemoteCreated is UI-only sync bookkeeping; dropped on lift.
	RemoteCreated bool `json:"remoteCreated"`
	// Title is the plot title configuration.
	Title v0.Title `json:"title"`
	// Legend is the plot legend configuration.
	Legend Legend `json:"legend"`
	// Channels binds channel keys to each axis.
	Channels v0.Channels `json:"channels"`
	// Ranges binds range keys to each x-axis.
	Ranges v0.Ranges `json:"ranges"`
	// Axes is the per-axis configuration container.
	Axes v0.AxesContainer `json:"axes"`
	// Lines are the per-line styling configurations.
	Lines []v0.Line `json:"lines"`
	// Rules are the annotation-line configurations.
	Rules []v0.Rule `json:"rules"`
}

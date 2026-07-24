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

import v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v0"

const Version = "1.0.0"

// StickyUnits is the optional axis-by-axis unit hint for a legend position.
type StickyUnits struct {
	X string `json:"x"`
	Y string `json:"y"`
}

// StickyRoot is the optional axis-by-axis root anchor for a legend position.
type StickyRoot struct {
	X string `json:"x"`
	Y string `json:"y"`
}

// LegendPosition is the sticky position of the legend within the plot.
type LegendPosition struct {
	X     float64      `json:"x"`
	Y     float64      `json:"y"`
	Units *StickyUnits `json:"units,omitempty"`
	Root  *StickyRoot  `json:"root,omitempty"`
}

// Legend is the plot legend configuration at v1. v1 adds Position.
type Legend struct {
	Visible  bool           `json:"visible"`
	Position LegendPosition `json:"position"`
}

// Data is the wire shape of a per-plot line plot state at v1.0.0. All fields
// other than Legend are unchanged from v0.
type Data struct {
	Version       string           `json:"version"`
	Key           string           `json:"key"`
	RemoteCreated bool             `json:"remoteCreated"`
	Title         v0.Title         `json:"title"`
	Legend        Legend           `json:"legend"`
	Channels      v0.Channels      `json:"channels"`
	Ranges        v0.Ranges        `json:"ranges"`
	Axes          v0.AxesContainer `json:"axes"`
	Lines         []v0.Line        `json:"lines"`
	Rules         []v0.Rule        `json:"rules"`
}

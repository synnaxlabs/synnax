// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v0"

// ZeroLegend is the default legend used when a v0 payload is migrated forward. Mirrors
// the Console's ZERO_LEGEND_STATE at v1, which replaces the v0 legend wholesale
// (including the user's visibility choice). Preserving exact Console behavior matters
// since the same migration runs client-side before this code existed.
var ZeroLegend = Legend{
	Visible: true,
	Position: LegendPosition{
		X:     50,
		Y:     50,
		Root:  &StickyRoot{X: "left", Y: "top"},
		Units: &StickyUnits{X: "px", Y: "px"},
	},
}

// Migrate transforms v0 line plot data into v1 by attaching the default legend.
func Migrate(old v0.Data) Data {
	return Data{
		Version:       Version,
		Key:           old.Key,
		RemoteCreated: old.RemoteCreated,
		Title:         old.Title,
		Legend:        ZeroLegend,
		Channels:      old.Channels,
		Ranges:        old.Ranges,
		Axes:          old.Axes,
		Lines:         old.Lines,
		Rules:         old.Rules,
	}
}

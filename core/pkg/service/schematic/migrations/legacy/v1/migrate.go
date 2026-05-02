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
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
)

// ZeroLegend is the default legend used when a v0 payload is migrated forward.
// Mirrors the console's ZERO_LEGEND_STATE at v1.
var ZeroLegend = Legend{
	Visible: true,
	Position: LegendPosition{
		X:     50,
		Y:     50,
		Units: &LegendUnits{X: "px", Y: "px"},
	},
	Colors: map[string]string{},
}

// Migrate transforms v0 schematic data into v1 by attaching the default legend.
func Migrate(old v0.Data) Data {
	return Data{
		Version:         Version,
		Editable:        old.Editable,
		FitViewOnResize: old.FitViewOnResize,
		Snapshot:        old.Snapshot,
		RemoteCreated:   old.RemoteCreated,
		Viewport:        old.Viewport,
		Nodes:           old.Nodes,
		Edges:           old.Edges,
		Props:           old.Props,
		Control:         old.Control,
		Legend:          ZeroLegend,
	}
}

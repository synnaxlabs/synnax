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
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v1"
)

// Migrate transforms v1 line plot data into v2 by setting the x-axes' Type to
// "time" and flipping the y-axes' labelDirection to "y". Mirrors the console's
// v1 to v2 stateMigration.
func Migrate(old v1.Data) Data {
	return Data{
		Version:       Version,
		Key:           old.Key,
		RemoteCreated: old.RemoteCreated,
		Title:         old.Title,
		Legend:        old.Legend,
		Channels:      old.Channels,
		Ranges:        old.Ranges,
		Axes: AxesContainer{
			RenderTrigger:    old.Axes.RenderTrigger,
			HasHadChannelSet: old.Axes.HasHadChannelSet,
			Axes: Axes{
				X1: upgradeXAxis(old.Axes.Axes.X1),
				X2: upgradeXAxis(old.Axes.Axes.X2),
				Y1: upgradeYAxis(old.Axes.Axes.Y1),
				Y2: upgradeYAxis(old.Axes.Axes.Y2),
				Y3: upgradeYAxis(old.Axes.Axes.Y3),
				Y4: upgradeYAxis(old.Axes.Axes.Y4),
			},
		},
		Lines: old.Lines,
		Rules: old.Rules,
	}
}

func upgradeXAxis(a v0.Axis) Axis {
	return Axis{
		Key:            a.Key,
		Label:          a.Label,
		LabelDirection: a.LabelDirection,
		LabelLevel:     a.LabelLevel,
		Bounds:         a.Bounds,
		AutoBounds:     a.AutoBounds,
		TickSpacing:    a.TickSpacing,
		Type:           "time",
	}
}

func upgradeYAxis(a v0.Axis) Axis {
	return Axis{
		Key:            a.Key,
		Label:          a.Label,
		LabelDirection: "y",
		LabelLevel:     a.LabelLevel,
		Bounds:         a.Bounds,
		AutoBounds:     a.AutoBounds,
		TickSpacing:    a.TickSpacing,
	}
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v2 holds the frozen wire format for console line plot per-plot state
// at version 2.0.0. v2 introduces the optional axis tick Type (linear/time) and
// flips y-axis labelDirection to "y" on migration.
package v2

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v1"
)

const Version = "2.0.0"

// Axis is the per-axis configuration at v2. Type is optional; absent (or empty
// after decode) means the default (linear). Legacy JSON blobs that store
// "type": null or omit the field both decode to "".
type Axis struct {
	Key            string        `json:"key"`
	Label          string        `json:"label"`
	LabelDirection string        `json:"labelDirection"`
	LabelLevel     string        `json:"labelLevel"`
	Bounds         v0.Bounds     `json:"bounds"`
	AutoBounds     v0.AutoBounds `json:"autoBounds"`
	TickSpacing    float64       `json:"tickSpacing"`
	Type           string        `json:"type,omitempty"`
}

// Axes bundles every axis configuration at v2.
type Axes struct {
	X1 Axis `json:"x1"`
	X2 Axis `json:"x2"`
	Y1 Axis `json:"y1"`
	Y2 Axis `json:"y2"`
	Y3 Axis `json:"y3"`
	Y4 Axis `json:"y4"`
}

// AxesContainer mirrors the console's AxesState wrapper.
type AxesContainer struct {
	RenderTrigger    int  `json:"renderTrigger"`
	HasHadChannelSet bool `json:"hasHadChannelSet"`
	Axes             Axes `json:"axes"`
}

// Data is the wire shape of a per-plot line plot state at v2.0.0.
type Data struct {
	Version       string        `json:"version"`
	Key           string        `json:"key"`
	RemoteCreated bool          `json:"remoteCreated"`
	Title         v0.Title      `json:"title"`
	Legend        v1.Legend     `json:"legend"`
	Channels      v0.Channels   `json:"channels"`
	Ranges        v0.Ranges     `json:"ranges"`
	Axes          AxesContainer `json:"axes"`
	Lines         []v0.Line     `json:"lines"`
	Rules         []v0.Rule     `json:"rules"`
}

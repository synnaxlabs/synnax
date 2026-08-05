// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v2 holds the frozen wire format for Console line plot per-plot state at
// version 2. v2 introduces the optional axis tick Type (linear/time) and flips y-axis
// labelDirection to "y" on migration.
package v2

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v1"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 2

// Axis is the per-axis configuration at v2. Type is optional; absent (or empty after
// decode) means the default (linear). Legacy JSON blobs that store "type": null or omit
// the field both decode to "".
type Axis struct {
	// Key identifies the axis (e.g. "x1", "y1").
	Key string `json:"key"`
	// Label is the axis label text.
	Label string `json:"label"`
	// LabelDirection is the label orientation.
	LabelDirection string `json:"labelDirection"`
	// LabelLevel is the typography level of the label.
	LabelLevel string `json:"labelLevel"`
	// Bounds are the manual axis bounds.
	Bounds v0.Bounds `json:"bounds"`
	// AutoBounds reports which bounds derive from data.
	AutoBounds v0.AutoBounds `json:"autoBounds"`
	// TickSpacing is the spacing between axis ticks in pixels.
	TickSpacing float64 `json:"tickSpacing"`
	// Type is the optional tick type; empty means linear.
	Type string `json:"type,omitempty"`
}

// Axes bundles every axis configuration at v2.
type Axes struct {
	// X1 is the primary x-axis configuration.
	X1 Axis `json:"x1"`
	// X2 is the secondary x-axis configuration.
	X2 Axis `json:"x2"`
	// Y1 is the first y-axis configuration.
	Y1 Axis `json:"y1"`
	// Y2 is the second y-axis configuration.
	Y2 Axis `json:"y2"`
	// Y3 is the third y-axis configuration.
	Y3 Axis `json:"y3"`
	// Y4 is the fourth y-axis configuration.
	Y4 Axis `json:"y4"`
}

// AxesContainer mirrors the Console's AxesState wrapper.
type AxesContainer struct {
	// RenderTrigger is UI-only render bookkeeping; dropped on lift.
	RenderTrigger int `json:"renderTrigger"`
	// HasHadChannelSet is UI-only bookkeeping; dropped on lift.
	HasHadChannelSet bool `json:"hasHadChannelSet"`
	// Axes bundles every axis configuration.
	Axes Axes `json:"axes"`
}

// Data is the wire shape of a per-plot line plot state at version 2.
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
	Legend v1.Legend `json:"legend"`
	// Channels binds channel keys to each axis.
	Channels v0.Channels `json:"channels"`
	// Ranges binds range keys to each x-axis.
	Ranges v0.Ranges `json:"ranges"`
	// Axes is the per-axis configuration container.
	Axes AxesContainer `json:"axes"`
	// Lines are the per-line styling configurations.
	Lines []v0.Line `json:"lines"`
	// Rules are the annotation-line configurations.
	Rules []v0.Rule `json:"rules"`
}

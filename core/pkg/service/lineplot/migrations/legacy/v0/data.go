// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for console line plot per-plot state at
// version 0.0.0. Per-version Data structs in this directory tree are immutable
// snapshots of what consoles actually persisted at that version. They are the
// JSON-decode targets for the storage migration chain that lifts older blobs
// forward into the typed lineplot.LinePlot.
package v0

import (
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/color"
)

// Version is the semantic version string written by the console at this state
// version.
const Version = "0.0.0"

// XY is the planar coordinate shape consoles persisted from version 0.0.0
// onward. Identical to spatial.XY at the wire level.
type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Bounds is the closed-open numeric interval used for axis bounds.
type Bounds struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// AutoBounds reports which edges of an axis derive their bound from data.
type AutoBounds struct {
	Lower bool `json:"lower"`
	Upper bool `json:"upper"`
}

// Title is the plot title configuration at v0.
type Title struct {
	Level   string `json:"level"`
	Visible bool   `json:"visible"`
}

// Legend is the plot legend configuration at v0. v0 has no position field; v1
// introduces it.
type Legend struct {
	Visible bool `json:"visible"`
}

// Axis is the per-axis configuration at v0. v2 introduces the optional Type
// field.
type Axis struct {
	Key            string     `json:"key"`
	Label          string     `json:"label"`
	LabelDirection string     `json:"labelDirection"`
	LabelLevel     string     `json:"labelLevel"`
	Bounds         Bounds     `json:"bounds"`
	AutoBounds     AutoBounds `json:"autoBounds"`
	TickSpacing    float64    `json:"tickSpacing"`
}

// Axes bundles every axis configuration. The console persists this nested
// under the AxesContainer below so the renderTrigger/hasHadChannelSet bookkeeping
// stays alongside.
type Axes struct {
	X1 Axis `json:"x1"`
	X2 Axis `json:"x2"`
	Y1 Axis `json:"y1"`
	Y2 Axis `json:"y2"`
	Y3 Axis `json:"y3"`
	Y4 Axis `json:"y4"`
}

// AxesContainer mirrors the console's AxesState wrapper. RenderTrigger and
// HasHadChannelSet are UI bookkeeping; only Axes survives into the typed body.
type AxesContainer struct {
	RenderTrigger    int  `json:"renderTrigger"`
	HasHadChannelSet bool `json:"hasHadChannelSet"`
	Axes             Axes `json:"axes"`
}

// Channels binds channel keys to each axis.
type Channels struct {
	X1 channel.Key   `json:"x1"`
	X2 channel.Key   `json:"x2"`
	Y1 []channel.Key `json:"y1"`
	Y2 []channel.Key `json:"y2"`
	Y3 []channel.Key `json:"y3"`
	Y4 []channel.Key `json:"y4"`
}

// Ranges binds opaque range keys to each x-axis. Strings (not UUIDs) because
// the console persists synthetic rolling-window keys (e.g. "recent",
// "rolling1m") alongside persisted ranges.
type Ranges struct {
	X1 []string `json:"x1"`
	X2 []string `json:"x2"`
}

// Line is the per-line styling configuration at v0. Label is optional in the
// wire format; absent means derive from the channel name at render time.
// Color was persisted as a hex string at v0; color.Color's UnmarshalJSON
// parses that on decode so the in-memory shape matches v5.
type Line struct {
	Key            string      `json:"key"`
	Label          *string     `json:"label,omitempty"`
	Color          color.Color `json:"color"`
	StrokeWidth    float64     `json:"strokeWidth"`
	Downsample     uint32      `json:"downsample"`
	DownsampleMode string      `json:"downsampleMode"`
}

// Rule is the annotation-line configuration at v0. Selected is UI-only and
// dropped at the final lift. Color was persisted as a hex string at v0;
// color.Color's UnmarshalJSON parses that on decode so the in-memory shape
// matches v5.
type Rule struct {
	Selected  *bool       `json:"selected,omitempty"`
	Key       string      `json:"key"`
	Label     string      `json:"label"`
	Color     color.Color `json:"color"`
	Axis      string      `json:"axis"`
	LineWidth float64     `json:"lineWidth"`
	LineDash  float64     `json:"lineDash"`
	Units     string      `json:"units"`
	Position  float64     `json:"position"`
}

// Data is the wire shape of a per-plot line plot state at v0.0.0. UI-only
// fields persisted alongside the model fields (viewport, selection) are
// silently ignored on decode since they do not survive the lift to the typed
// body.
type Data struct {
	Version       string        `json:"version"`
	Key           string        `json:"key"`
	RemoteCreated bool          `json:"remoteCreated"`
	Title         Title         `json:"title"`
	Legend        Legend        `json:"legend"`
	Channels      Channels      `json:"channels"`
	Ranges        Ranges        `json:"ranges"`
	Axes          AxesContainer `json:"axes"`
	Lines         []Line        `json:"lines"`
	Rules         []Rule        `json:"rules"`
}

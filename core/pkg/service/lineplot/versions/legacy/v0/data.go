// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v0 holds the frozen wire format for Console line plot per-plot state at
// version 0. Per-version Data structs in this directory tree are immutable snapshots of
// what Consoles actually persisted at that version. They are the JSON-decode targets
// for the storage migration chain that lifts older blobs forward into the typed
// lineplot.LinePlot.
package v0

import (
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	color "github.com/synnaxlabs/x/color/versions/v0"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 0

// XY is the planar coordinate shape Consoles persisted from version 0 onward. Identical
// to spatial.XY at the wire level.
type XY struct {
	// X is the horizontal coordinate.
	X float64 `json:"x"`
	// Y is the vertical coordinate.
	Y float64 `json:"y"`
}

// Bounds is the closed-open numeric interval used for axis bounds.
type Bounds struct {
	// Lower is the inclusive lower bound.
	Lower float64 `json:"lower"`
	// Upper is the exclusive upper bound.
	Upper float64 `json:"upper"`
}

// AutoBounds reports which edges of an axis derive their bound from data.
type AutoBounds struct {
	// Lower reports whether the lower bound derives from data.
	Lower bool `json:"lower"`
	// Upper reports whether the upper bound derives from data.
	Upper bool `json:"upper"`
}

// Title is the plot title configuration at v0.
type Title struct {
	// Level is the typography level of the title.
	Level string `json:"level"`
	// Visible toggles title display.
	Visible bool `json:"visible"`
}

// Legend is the plot legend configuration at v0. v0 has no position field; v1
// introduces it.
type Legend struct {
	// Visible toggles legend display.
	Visible bool `json:"visible"`
}

// Axis is the per-axis configuration at v0. v2 introduces the optional Type
// field.
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
	Bounds Bounds `json:"bounds"`
	// AutoBounds reports which bounds derive from data.
	AutoBounds AutoBounds `json:"autoBounds"`
	// TickSpacing is the spacing between axis ticks in pixels.
	TickSpacing float64 `json:"tickSpacing"`
}

// Axes bundles every axis configuration. The Console persists this nested under the
// AxesContainer below so the renderTrigger/hasHadChannelSet bookkeeping stays
// alongside.
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

// AxesContainer mirrors the Console's AxesState wrapper. RenderTrigger and
// HasHadChannelSet are UI bookkeeping; only Axes survives into the typed body.
type AxesContainer struct {
	// RenderTrigger is UI-only render bookkeeping; dropped on lift.
	RenderTrigger int `json:"renderTrigger"`
	// HasHadChannelSet is UI-only bookkeeping; dropped on lift.
	HasHadChannelSet bool `json:"hasHadChannelSet"`
	// Axes bundles every axis configuration.
	Axes Axes `json:"axes"`
}

// Channels binds channel keys to each axis.
type Channels struct {
	// X1 is the channel bound to the primary x-axis.
	X1 channel.Key `json:"x1"`
	// X2 is the channel bound to the secondary x-axis.
	X2 channel.Key `json:"x2"`
	// Y1 holds the channels plotted on the first y-axis.
	Y1 []channel.Key `json:"y1"`
	// Y2 holds the channels plotted on the second y-axis.
	Y2 []channel.Key `json:"y2"`
	// Y3 holds the channels plotted on the third y-axis.
	Y3 []channel.Key `json:"y3"`
	// Y4 holds the channels plotted on the fourth y-axis.
	Y4 []channel.Key `json:"y4"`
}

// Ranges binds opaque range keys to each x-axis. Strings (not UUIDs) because the
// Console persists synthetic rolling-window keys (e.g. "recent", "rolling1m") alongside
// persisted ranges.
type Ranges struct {
	// X1 holds the range keys bound to the primary x-axis.
	X1 []string `json:"x1"`
	// X2 holds the range keys bound to the secondary x-axis.
	X2 []string `json:"x2"`
}

// Line is the per-line styling configuration at v0. Label is optional in the wire
// format; absent means derive from the channel name at render time. Color was persisted
// as a hex string at v0; color.Color's UnmarshalJSON parses that on decode so the
// in-memory shape matches v5.
type Line struct {
	// Key is the line's unique key.
	Key string `json:"key"`
	// Label overrides the derived line label; absent derives it.
	Label *string `json:"label,omitempty"`
	// Color is the line color.
	Color color.Color `json:"color"`
	// StrokeWidth is the line width in pixels.
	StrokeWidth float64 `json:"strokeWidth"`
	// Downsample is the downsampling factor.
	Downsample uint32 `json:"downsample"`
	// DownsampleMode selects the downsampling strategy.
	DownsampleMode string `json:"downsampleMode"`
}

// Rule is the annotation-line configuration at v0. Selected is UI-only and dropped at
// the final lift. Color was persisted as a hex string at v0; color.Color's
// UnmarshalJSON parses that on decode so the in-memory shape matches v5.
type Rule struct {
	// Selected is UI-only selection state; dropped on lift.
	Selected *bool `json:"selected,omitempty"`
	// Key is the rule's unique key.
	Key string `json:"key"`
	// Label is the rule label text.
	Label string `json:"label"`
	// Color is the rule line color.
	Color color.Color `json:"color"`
	// Axis is the key of the axis the rule is bound to.
	Axis string `json:"axis"`
	// LineWidth is the rule line width in pixels.
	LineWidth float64 `json:"lineWidth"`
	// LineDash is the rule dash spacing.
	LineDash float64 `json:"lineDash"`
	// Units is the unit label displayed with the position.
	Units string `json:"units"`
	// Position is the rule position in axis units.
	Position float64 `json:"position"`
}

// Data is the wire shape of a per-plot line plot state at version 0. UI-only fields
// persisted alongside the model fields (viewport, selection) are silently ignored on
// decode since they do not survive the lift to the typed body.
type Data struct {
	// Version is the version stamped inside the blob.
	Version imex.Version `json:"version"`
	// Key is the Console-local plot key.
	Key string `json:"key"`
	// RemoteCreated is UI-only sync bookkeeping; dropped on lift.
	RemoteCreated bool `json:"remoteCreated"`
	// Title is the plot title configuration.
	Title Title `json:"title"`
	// Legend is the plot legend configuration.
	Legend Legend `json:"legend"`
	// Channels binds channel keys to each axis.
	Channels Channels `json:"channels"`
	// Ranges binds range keys to each x-axis.
	Ranges Ranges `json:"ranges"`
	// Axes is the per-axis configuration container.
	Axes AxesContainer `json:"axes"`
	// Lines are the per-line styling configurations.
	Lines []Line `json:"lines"`
	// Rules are the annotation-line configurations.
	Rules []Rule `json:"rules"`
}

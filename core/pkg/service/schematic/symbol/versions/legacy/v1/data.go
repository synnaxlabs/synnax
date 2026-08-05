// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for the Console schematic symbol export at
// version 1. The Console retrieves the symbol from the Core and writes it out directly,
// stamping the symbol's old persisted version field as the envelope version. The body
// is therefore the specification in camelCase, which is why v1 has no Migrate into the
// storage chain — only the ImEx decode path reads it.
package v1

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/spatial"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 1

// Region is a style region as the Console serializes it.
type Region struct {
	// Key is the region's unique key.
	Key string `json:"key" msgpack:"key"`
	// Name is the region name.
	Name string `json:"name" msgpack:"name"`
	// Selectors are the SVG selectors the region targets.
	Selectors []string `json:"selectors" msgpack:"selectors"`
	// StrokeColor is the optional stroke color override.
	StrokeColor *string `json:"strokeColor" msgpack:"strokeColor"`
	// FillColor is the optional fill color override.
	FillColor *string `json:"fillColor" msgpack:"fillColor"`
}

// State is a named visual state as the Console serializes it.
type State struct {
	// Key is the state's unique key.
	Key string `json:"key" msgpack:"key"`
	// Name is the state name.
	Name string `json:"name" msgpack:"name"`
	// Regions are the SVG regions the state styles.
	Regions []Region `json:"regions" msgpack:"regions"`
}

// Handle is a connection point as the Console serializes it.
type Handle struct {
	// Key is the handle's unique key.
	Key string `json:"key" msgpack:"key"`
	// Position is the handle position in the symbol's local space.
	Position spatial.XY `json:"position" msgpack:"position"`
	// Orientation is the direction the handle faces.
	Orientation spatial.OuterLocation `json:"orientation" msgpack:"orientation"`
}

// Spec is the symbol specification as the Console serializes it.
type Spec struct {
	// SVG is the symbol's SVG markup.
	SVG string `json:"svg" msgpack:"svg"`
	// States are the symbol's visual states.
	States []State `json:"states" msgpack:"states"`
	// Variant is the symbol variant identifier.
	Variant string `json:"variant" msgpack:"variant"`
	// Handles are the connection handle definitions.
	Handles []Handle `json:"handles" msgpack:"handles"`
	// Scale is the symbol scale factor.
	Scale float64 `json:"scale" msgpack:"scale"`
	// ScaleStroke reports whether strokes scale with the symbol.
	ScaleStroke bool `json:"scaleStroke" msgpack:"scaleStroke"`
	// PreviewViewport is the optional preview viewport.
	PreviewViewport *spatial.Viewport `json:"previewViewport" msgpack:"previewViewport"`
}

// Data is the Console symbol export at version 1.
type Data struct {
	// Spec is the persisted symbol specification.
	Spec Spec `json:"data" msgpack:"data"`
}

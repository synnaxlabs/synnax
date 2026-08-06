// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package console holds the frozen wire format of the Console Arc export: the typed Arc
// the Console retrieved from the Core, spread into the file in camelCase. Releases
// 0.46.0 onward write it whenever the exported Arc was not open; those before 0.54.0
// also carry the Arc's own "0.0.0" version. Lift converts it into the shape the Arc
// migrations take.
package console

import (
	graph "github.com/synnaxlabs/arc/graph/versions/v0"
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	text "github.com/synnaxlabs/arc/text/versions/v1"
	types "github.com/synnaxlabs/arc/types/versions/v0"
)

// Data is the typed Arc export the Console wrote.
type Data struct {
	// Mode is the raw mode string ("graph" or "text").
	Mode string `json:"mode" msgpack:"mode"`
	// Graph is the graph-mode document.
	Graph Graph `json:"graph" msgpack:"graph"`
	// Text is the text-mode document.
	Text text.Text `json:"text" msgpack:"text"`
}

// Graph mirrors the pre-lift graph shape the export carries: keyless edges and per-node
// config records that the graph migration folds into Inputs.
type Graph struct {
	// Viewport is the editor viewport.
	Viewport graph.Viewport `json:"viewport" msgpack:"viewport"`
	// Functions are the graph's functions.
	Functions []Function `json:"functions" msgpack:"functions"`
	// Edges are the graph edges.
	Edges ir.Edges `json:"edges" msgpack:"edges"`
	// Nodes are the graph nodes.
	Nodes graph.Nodes `json:"nodes" msgpack:"nodes"`
}

// Function mirrors an IR function as the export serializes it.
type Function struct {
	// Key is the function's unique key.
	Key string `json:"key" msgpack:"key"`
	// Body is the function body.
	Body ir.Body `json:"body" msgpack:"body"`
	// Config is the config params.
	Config Params `json:"config" msgpack:"config"`
	// Inputs are the input params.
	Inputs Params `json:"inputs" msgpack:"inputs"`
	// Outputs are the output params.
	Outputs Params `json:"outputs" msgpack:"outputs"`
	// Channels are the channels the function reads and writes.
	Channels types.Channels `json:"channels" msgpack:"channels"`
}

// Param mirrors a type param as the export serializes it.
type Param struct {
	// Name is the param name.
	Name string `json:"name" msgpack:"name"`
	// Type is the param type.
	Type Type `json:"type" msgpack:"type"`
	// Value is the configured param value.
	Value any `json:"value" msgpack:"value"`
}

// Params is a collection of params.
type Params []Param

// Type mirrors an Arc type as the export serializes it: camelCase keys, recursing
// through elem, constraint, and function params.
type Type struct {
	// Inputs are the input params.
	Inputs Params `json:"inputs" msgpack:"inputs"`
	// Outputs are the output params.
	Outputs Params `json:"outputs" msgpack:"outputs"`
	// Config is the config params.
	Config Params `json:"config" msgpack:"config"`
	// Kind is the type kind.
	Kind types.Kind `json:"kind" msgpack:"kind"`
	// Name is the type name.
	Name string `json:"name" msgpack:"name"`
	// Elem is the element type for containers; nil otherwise.
	Elem *Type `json:"elem" msgpack:"elem"`
	// Unit is the optional physical unit.
	Unit *types.Unit `json:"unit" msgpack:"unit"`
	// Constraint is the optional type constraint.
	Constraint *Type `json:"constraint" msgpack:"constraint"`
	// ChanDirection is the channel direction for channel kinds.
	ChanDirection types.ChanDirection `json:"chanDirection" msgpack:"chanDirection"`
}

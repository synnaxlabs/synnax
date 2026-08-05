// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"

	"github.com/samber/lo"
	graphv0 "github.com/synnaxlabs/arc/graph/versions/v0"
	graphv1 "github.com/synnaxlabs/arc/graph/versions/v1"
	irv0 "github.com/synnaxlabs/arc/ir/versions/v0"
	"github.com/synnaxlabs/arc/text"
	typesv0 "github.com/synnaxlabs/arc/types/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// consoleType mirrors typesv0.Type as Console-written files serialize it:
// camelCase keys, recursing through elem, constraint, and function params.
// Frozen; Console files no longer evolve.
type consoleType struct {
	// Inputs are the input params.
	Inputs consoleParams `json:"inputs" msgpack:"inputs"`
	// Outputs are the output params.
	Outputs consoleParams `json:"outputs" msgpack:"outputs"`
	// Config is the config params.
	Config consoleParams `json:"config" msgpack:"config"`
	// Kind is the type kind.
	Kind typesv0.Kind `json:"kind" msgpack:"kind"`
	// Name is the type name.
	Name string `json:"name" msgpack:"name"`
	// Elem is the element type for containers; nil otherwise.
	Elem *consoleType `json:"elem" msgpack:"elem"`
	// Unit is the optional physical unit.
	Unit *typesv0.Unit `json:"unit" msgpack:"unit"`
	// Constraint is the optional type constraint.
	Constraint *consoleType `json:"constraint" msgpack:"constraint"`
	// ChanDirection is the channel direction for channel kinds.
	ChanDirection typesv0.ChanDirection `json:"chanDirection" msgpack:"chanDirection"`
}

func (t consoleType) lift() typesv0.Type {
	out := typesv0.Type{
		Kind: t.Kind, Name: t.Name, Unit: t.Unit, ChanDirection: t.ChanDirection,
	}
	out.Inputs = t.Inputs.lift()
	out.Outputs = t.Outputs.lift()
	out.Config = t.Config.lift()
	if t.Elem != nil {
		elem := t.Elem.lift()
		out.Elem = &elem
	}
	if t.Constraint != nil {
		constraint := t.Constraint.lift()
		out.Constraint = &constraint
	}
	return out
}

// consoleParam mirrors typesv0.Param as Console-written files serialize it.
type consoleParam struct {
	// Name is the param name.
	Name string `json:"name" msgpack:"name"`
	// Type is the param type.
	Type consoleType `json:"type" msgpack:"type"`
	// Value is the configured param value.
	Value any `json:"value" msgpack:"value"`
}

type consoleParams []consoleParam

func (p consoleParams) lift() typesv0.Params {
	return lo.Map(p, func(cp consoleParam, _ int) typesv0.Param {
		return typesv0.Param{Name: cp.Name, Type: cp.Type.lift(), Value: cp.Value}
	})
}

// consoleFunction mirrors irv0.Function as Console-written files serialize it.
type consoleFunction struct {
	// Key is the function's unique key.
	Key string `json:"key" msgpack:"key"`
	// Body is the function body.
	Body irv0.Body `json:"body" msgpack:"body"`
	// Config is the config params.
	Config consoleParams `json:"config" msgpack:"config"`
	// Inputs are the input params.
	Inputs consoleParams `json:"inputs" msgpack:"inputs"`
	// Outputs are the output params.
	Outputs consoleParams `json:"outputs" msgpack:"outputs"`
	// Channels are the channels the function reads and writes.
	Channels typesv0.Channels `json:"channels" msgpack:"channels"`
}

func (f consoleFunction) lift() irv0.Function {
	return irv0.Function{
		Key: f.Key, Body: f.Body, Config: f.Config.lift(),
		Inputs: f.Inputs.lift(), Outputs: f.Outputs.lift(), Channels: f.Channels,
	}
}

// consoleGraph mirrors graphv0.Graph, the pre-lift shape Console files carry:
// keyless edges and per-node config records that MigrateGraph folds into Inputs.
type consoleGraph struct {
	// Viewport is the editor viewport.
	Viewport graphv0.Viewport `json:"viewport" msgpack:"viewport"`
	// Functions are the graph's functions.
	Functions []consoleFunction `json:"functions" msgpack:"functions"`
	// Edges are the graph edges.
	Edges irv0.Edges `json:"edges" msgpack:"edges"`
	// Nodes are the graph nodes.
	Nodes graphv0.Nodes `json:"nodes" msgpack:"nodes"`
}

func (g consoleGraph) lift() graphv0.Graph {
	return graphv0.Graph{
		Viewport: g.Viewport,
		Functions: lo.Map(g.Functions, func(f consoleFunction, _ int) irv0.Function {
			return f.lift()
		}),
		Edges: g.Edges,
		Nodes: g.Nodes,
	}
}

// consoleTyped mirrors the typed Arc export written by the Console.
type consoleTyped struct {
	// Mode is the raw mode string ("graph" or "text").
	Mode Mode `json:"mode" msgpack:"mode"`
	// Graph is the graph-mode document.
	Graph consoleGraph `json:"graph" msgpack:"graph"`
	// Text is the text-mode document.
	Text text.Text `json:"text" msgpack:"text"`
}

// DecodeImport materializes the envelope's body as a current-version Arc. Envelopes
// stamped at or above Floor decode through the generated migration chain; older
// ones are Console-era files — camelCase typed exports or console states — and are
// lifted forward. An envelope newer than Latest is rejected with a path-scoped
// validation error.
func DecodeImport(ctx context.Context, env imex.Envelope) (Arc, error) {
	if env.Version >= Floor {
		return decodeMigrate(ctx, env)
	}
	// Typed exports always carry a top-level name; console states never do.
	// Console typed exports are versionless with camelCase keys and the
	// pre-lift v0 graph shape.
	if env.BodyNamed() {
		ct, err := imex.Decode[consoleTyped](ctx, env)
		if err != nil {
			return Arc{}, err
		}
		g, err := graphv1.MigrateGraph(ctx, ct.Graph.lift())
		if err != nil {
			return Arc{}, err
		}
		mode := ct.Mode
		if mode == "" {
			mode = ModeGraph
		}
		return Arc{Mode: mode, Graph: g, Text: ct.Text}, nil
	}
	// "0.0.0".."2.0.0" console states embed the graph inline. Nothing newer
	// exists: the shipped Console never wrote a later state format.
	body, err := imex.Decode[msgpack.EncodedJSON](ctx, env)
	if err != nil {
		return Arc{}, err
	}
	doc, err := legacy.MigrateData(body)
	if err != nil {
		return Arc{}, err
	}
	mode := Mode(doc.Mode)
	if mode == "" {
		mode = ModeGraph
	}
	return Arc{Mode: mode, Graph: doc.Graph, Text: doc.Text}, nil
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package console

import (
	"github.com/samber/lo"
	graph "github.com/synnaxlabs/arc/graph/versions/v0"
	ir "github.com/synnaxlabs/arc/ir/versions/v0"
	types "github.com/synnaxlabs/arc/types/versions/v0"
)

// Lift converts the exported graph into the v0 graph the Arc migrations start from.
func (g Graph) Lift() graph.Graph {
	return graph.Graph{
		Viewport: g.Viewport,
		Functions: lo.Map(g.Functions, func(f Function, _ int) ir.Function {
			return f.lift()
		}),
		// Payloads before 0.54 carried no kind; every edge they wrote was continuous.
		Edges: lo.Map(g.Edges, func(e ir.Edge, _ int) ir.Edge {
			if e.Kind == ir.EdgeKindUnspecified {
				e.Kind = ir.EdgeKindContinuous
			}
			return e
		}),
		Nodes: g.Nodes,
	}
}

func (f Function) lift() ir.Function {
	return ir.Function{
		Key: f.Key, Body: f.Body, Config: f.Config.lift(),
		Inputs: f.Inputs.lift(), Outputs: f.Outputs.lift(), Channels: f.Channels,
	}
}

func (p Params) lift() types.Params {
	return lo.Map(p, func(param Param, _ int) types.Param {
		return types.Param{
			Name:  param.Name,
			Type:  param.Type.lift(),
			Value: param.Value,
		}
	})
}

func (t Type) lift() types.Type {
	out := types.Type{
		Kind:          t.Kind,
		Name:          t.Name,
		Unit:          t.Unit,
		ChanDirection: t.ChanDirection,
		Inputs:        t.Inputs.lift(),
		Outputs:       t.Outputs.lift(),
		Config:        t.Config.lift(),
	}
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

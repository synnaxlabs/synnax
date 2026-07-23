// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer

import (
	"github.com/synnaxlabs/arc/analyzer/constraints"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/set"
)

// ResolveNodeTypes checks type compatibility across edges, unifies the constraint
// system, applies substitutions to resolve concrete types in node inputs and
// outputs, and verifies that every required input is
// satisfied by an incoming edge. A required input is one whose parameter has no
// default Value; leaving it unconnected would force the runtime to materialize a
// series from a nil value, so it is rejected here as a diagnostic instead. It also
// clears the default Value of any input fed by an edge, leaving a nil Value as the
// sole marker of an edge-fed input.
func ResolveNodeTypes(
	nodes ir.Nodes,
	edges ir.Edges,
	cs *constraints.System,
	diag *diagnostics.Diagnostics,
) bool {
	for _, edge := range edges {
		sourceNode, ok := nodes.Find(edge.Source.Node)
		if !ok {
			diag.Add(diagnostics.Errorf(nil,
				"edge source node '%s' not found", edge.Source.Node))
			return false
		}
		sourceParam, ok := sourceNode.Outputs.Get(edge.Source.Param)
		if !ok {
			diag.Add(diagnostics.Errorf(nil,
				"output '%s' not found in node '%s' (%s)",
				edge.Source.Param, edge.Source.Node, sourceNode.Type))
			return false
		}
		targetNode, ok := nodes.Find(edge.Target.Node)
		if !ok {
			diag.Add(diagnostics.Errorf(nil,
				"edge target node '%s' not found", edge.Target.Node))
			return false
		}
		targetParam, ok := targetNode.Inputs.Get(edge.Target.Param)
		if !ok {
			continue
		}
		if err := atypes.Check(cs, sourceParam.Type, targetParam.Type, nil, ""); err != nil {
			diag.Add(diagnostics.Error(err, nil))
			return false
		}
	}
	if err := cs.Unify(); err != nil {
		addUnificationError(diag, err, nil)
		return false
	}
	for idx, n := range nodes {
		for j, p := range n.Outputs {
			nodes[idx].Outputs[j].Type = cs.ApplySubstitutions(p.Type)
		}
		for j, p := range n.Inputs {
			nodes[idx].Inputs[j].Type = cs.ApplySubstitutions(p.Type)
		}
	}
	connected := set.New[ir.Handle]()
	for _, edge := range edges {
		connected.Add(edge.Target)
	}
	missingRequiredInput := false
	for ni := range nodes {
		for pi := range nodes[ni].Inputs {
			p := nodes[ni].Inputs[pi]
			if connected.Contains(ir.Handle{Node: nodes[ni].Key, Param: p.Name}) {
				// An edge supplies this input's value, so drop any default; a nil
				// Value then marks the input as edge-fed.
				nodes[ni].Inputs[pi].Value = nil
				continue
			}
			// A var ref input is bound to its variable's node at runtime.
			if p.Type.Kind == types.KindVarRef {
				continue
			}
			if p.Value != nil {
				continue
			}
			diag.Add(diagnostics.Errorf(
				nil,
				"node '%s' (%s) missing required input '%s'",
				nodes[ni].Key,
				nodes[ni].Type,
				p.Name,
			))
			missingRequiredInput = true
		}
	}
	return !missingRequiredInput
}

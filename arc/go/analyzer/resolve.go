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
)

// ResolveNodeTypes checks type compatibility across edges, unifies the constraint
// system, and applies substitutions to resolve concrete types in node inputs,
// outputs, and config parameters.
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
			resolved := cs.ApplySubstitutions(p.Type)
			// Inputs that remain unresolved after unification have no upstream
			// edge constraining them (e.g., a synthetic expression node's
			// trigger input on a top-level expression-to-channel flow). Default
			// the placeholder to u8 so runtime default-value allocation has a
			// concrete type to build from.
			if resolved.Kind == types.KindVariable {
				resolved = types.U8()
				if nodes[idx].Inputs[j].Value == nil {
					nodes[idx].Inputs[j].Value = uint8(0)
				}
			}
			nodes[idx].Inputs[j].Type = resolved
		}
		for j, p := range n.Config {
			nodes[idx].Config[j].Type = cs.ApplySubstitutions(p.Type)
		}
	}
	return true
}

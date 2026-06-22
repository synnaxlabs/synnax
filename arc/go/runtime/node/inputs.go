// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
)

// ResolvedInputs reads a node's inputs, transparently handling both static brace
// values and edge-fed expression values. It lets host nodes accept an expression
// (e.g. `time = time.now() + 5s`) for any input without per-input bookkeeping.
type ResolvedInputs struct {
	node  ir.Node
	edges []edgeInput
}

type edgeInput struct {
	name string
	idx  int
}

// ResolveInputs splits n's inputs: an input is edge-fed when its Value is nil (set by
// ResolveNodeTypes) and it is not a channel-typed param (a direct channel binding).
func ResolveInputs(state *State, n ir.Node) (ResolvedInputs, error) {
	ri := ResolvedInputs{node: n}
	for _, p := range n.Inputs {
		if p.Value != nil || p.Type.Kind == types.KindChan {
			continue
		}
		idx, err := state.ResolveInput(p.Name)
		if err != nil {
			return ResolvedInputs{}, err
		}
		ri.edges = append(ri.edges, edgeInput{name: p.Name, idx: idx})
	}
	return ri, nil
}

// HasEdges reports whether any input is edge-fed.
func (r ResolvedInputs) HasEdges() bool { return len(r.edges) > 0 }

// ValueMap returns the node's static input values overlaid with the latest value of
// each edge-fed input. Call after RefreshInputs so the edge values are current.
func (r ResolvedInputs) ValueMap(state *State) map[string]any {
	m := r.node.Inputs.ValueMap()
	for _, e := range r.edges {
		s := state.Input(e.idx)
		if s.Len() == 0 {
			continue
		}
		if v := s.AtAny(int(s.Len() - 1)); v != nil {
			m[e.name] = v
		}
	}
	return m
}

// ValidationMap returns the static input values with a typed-zero placeholder for each
// edge-fed input, for Create-time validation before the edge values exist.
func (r ResolvedInputs) ValidationMap() map[string]any {
	m := r.node.Inputs.ValueMap()
	for _, e := range r.edges {
		if p, ok := r.node.Inputs.Get(e.name); ok {
			m[e.name] = zeroValue(p.Type)
		}
	}
	return m
}

// zeroValue returns the zero value of the Go type matching t, used as a placeholder
// for an edge-fed input during Create-time validation.
func zeroValue(t types.Type) any {
	switch t.Kind {
	case types.KindString:
		return ""
	case types.KindF32:
		return float32(0)
	case types.KindF64:
		return float64(0)
	case types.KindU8:
		return uint8(0)
	case types.KindU16:
		return uint16(0)
	case types.KindU32:
		return uint32(0)
	case types.KindU64:
		return uint64(0)
	case types.KindI8:
		return int8(0)
	case types.KindI16:
		return int16(0)
	case types.KindI32:
		return int32(0)
	default:
		return int64(0)
	}
}

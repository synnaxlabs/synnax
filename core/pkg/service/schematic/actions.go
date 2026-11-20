// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// schematic.
func (CreatePayload) Handle(Schematic) (Schematic, error) {
	return Schematic{}, actions.CoreOnlyError("a schematic create")
}

// Handle moves the named node to the given position. No-op if no node matches.
func (p SetNodePositionPayload) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == p.Key {
			state.Nodes[i].Position = p.Position
			break
		}
	}
	return state, nil
}

// Handle inserts the node if no node with the same key exists, otherwise replaces the
// existing node and its config in place.
func (p SetNodePayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Nodes, func(n Node) bool { return n.Key == p.Node.Key })
	if i == -1 {
		state.Nodes = append(state.Nodes, p.Node)
		return state, nil
	}
	state.Nodes[i] = p.Node
	return state, nil
}

// Handle removes the node with the matching key and the edges that connect to it. The
// children of a removed group box move to the parent of the group box. No-op if no node
// matches.
func (p RemoveNodePayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Nodes, func(n Node) bool { return n.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	parent := state.Nodes[i].Parent
	state.Nodes = slices.Delete(state.Nodes, i, i+1)
	state.Edges = slices.DeleteFunc(
		state.Edges,
		func(e Edge) bool { return e.Source.Node == p.Key || e.Target.Node == p.Key },
	)
	for j, n := range state.Nodes {
		if n.Parent != nil && *n.Parent == p.Key {
			state.Nodes[j].Parent = parent
		}
	}
	return state, nil
}

// Handle appends the edge to the schematic. No-op when an edge with the same key
// already exists, or when the source or target node does not exist.
func (p AddEdgePayload) Handle(state Schematic) (Schematic, error) {
	if hasEdge(state, p.Edge.Key) ||
		!hasNode(state, p.Edge.Source.Node) ||
		!hasNode(state, p.Edge.Target.Node) {
		return state, nil
	}
	state.Edges = append(state.Edges, p.Edge)
	return state, nil
}

// Handle removes the edge with the matching key. No-op if no edge matches.
func (p RemoveEdgePayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Edges, func(e Edge) bool { return e.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	state.Edges = slices.Delete(state.Edges, i, i+1)
	return state, nil
}

func hasNode(state Schematic, key string) bool {
	return slices.ContainsFunc(state.Nodes, func(n Node) bool { return n.Key == key })
}

func hasEdge(state Schematic, key string) bool {
	return slices.ContainsFunc(state.Edges, func(e Edge) bool { return e.Key == key })
}

// Handle applies the set to the schematic's name, group, or project, or to the config
// of a node or edge.
func (p SetPayload) Handle(state Schematic) (Schematic, error) {
	return apply(state, patch.Set(p))
}

// Handle applies the remove to the schematic's group or project, or to the config of a
// node or edge.
func (p RemovePayload) Handle(state Schematic) (Schematic, error) {
	return apply(state, patch.Remove(p))
}

// Handle applies the insert to the config of a node or edge.
func (p InsertPayload) Handle(state Schematic) (Schematic, error) {
	return apply(state, patch.Insert(p))
}

// Handle applies the move to the config of a node or edge.
func (p MovePayload) Handle(state Schematic) (Schematic, error) {
	return apply(state, patch.Move(p))
}

// apply applies a path action to the schematic's name, group, or project, to the config
// or parent of a node, or to the config of an edge. Any other path fails, except that a
// path through a node or edge that does not exist does nothing.
func apply[A patch.Action](state Schematic, a A) (Schematic, error) {
	c, err := patch.Begin(a)
	if err != nil {
		return Schematic{}, err
	}
	if err = state.Patch(c); err != nil {
		return Schematic{}, err
	}
	return state, nil
}

func newSetAction(s patch.Set) Action { return NewSetAction(SetPayload(s)) }

func newRemoveAction(r patch.Remove) Action { return NewRemoveAction(RemovePayload(r)) }

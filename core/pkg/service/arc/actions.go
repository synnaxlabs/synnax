// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import "github.com/synnaxlabs/arc/ir"

// Handle replaces the Arc module's name.
func (p RenamePayload) Handle(state Arc) (Arc, error) {
	state.Name = p.Name
	return state, nil
}

// Handle switches the module between text and graph representation.
func (p SetModePayload) Handle(state Arc) (Arc, error) {
	state.Mode = p.Mode
	return state, nil
}

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place.
func (p SetNodePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Node.Key {
			state.Graph.Nodes[i] = p.Node
			return state, nil
		}
	}
	state.Graph.Nodes = append(state.Graph.Nodes, p.Node)
	return state, nil
}

// Handle moves the named node to the given canvas position. No-op if no node
// matches.
func (p SetNodePositionPayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Key {
			state.Graph.Nodes[i].Position = p.Position
			break
		}
	}
	return state, nil
}

// Handle replaces the configuration of the named node. No-op if no node
// matches.
func (p SetNodeConfigPayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Key {
			state.Graph.Nodes[i].Config = p.Config
			break
		}
	}
	return state, nil
}

// Handle removes the node with the matching key along with any edges connected
// to it.
func (p RemoveNodePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Nodes {
		if state.Graph.Nodes[i].Key == p.Key {
			state.Graph.Nodes = append(state.Graph.Nodes[:i], state.Graph.Nodes[i+1:]...)
			break
		}
	}
	kept := make(ir.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Source.Node == p.Key || e.Target.Node == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

// Handle appends the edge to the graph. No-op when an edge with the same source
// and target handles already exists.
func (p AddEdgePayload) Handle(state Arc) (Arc, error) {
	for _, e := range state.Graph.Edges {
		if e.Source == p.Edge.Source && e.Target == p.Edge.Target {
			return state, nil
		}
	}
	state.Graph.Edges = append(state.Graph.Edges, p.Edge)
	return state, nil
}

// Handle removes the edge matching the given source and target handles, if
// present.
func (p RemoveEdgePayload) Handle(state Arc) (Arc, error) {
	kept := make(ir.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Source == p.Source && e.Target == p.Target {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

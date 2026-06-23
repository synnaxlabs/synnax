// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"maps"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// Handle replaces the Arc module's name.
func (p RenamePayload) Handle(state Arc) (Arc, error) {
	state.Name = p.Name
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

// Handle merges the payload config into the configs entry for the given key in
// the graph configs map. Top-level fields present in the payload overwrite
// existing fields; fields absent from the payload are preserved.
func (p SetNodeConfigPayload) Handle(state Arc) (Arc, error) {
	if state.Graph.Configs == nil {
		state.Graph.Configs = make(map[string]msgpack.EncodedJSON)
	}
	if existing := state.Graph.Configs[p.Key]; existing != nil {
		merged := make(msgpack.EncodedJSON, len(existing)+len(p.Config))
		maps.Copy(merged, existing)
		maps.Copy(merged, p.Config)
		state.Graph.Configs[p.Key] = merged
		return state, nil
	}
	state.Graph.Configs[p.Key] = p.Config
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
	delete(state.Graph.Configs, p.Key)
	kept := make(graph.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Source.Node == p.Key || e.Target.Node == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

// Handle is the materialization seam for a character insertion. Collaborative edits are
// relayed between clients without the server materializing them into the module's text,
// so it returns the state unchanged; durable materialization into Arc.Text is a
// follow-on concern (see SY-4393).
func (p InsertCharPayload) Handle(state Arc) (Arc, error) { return state, nil }

// Handle appends the edge to the graph. No-op when an edge with the same source
// and target handles already exists, so concurrent additions of the same
// connection converge regardless of differing keys.
func (p AddEdgePayload) Handle(state Arc) (Arc, error) {
	for _, e := range state.Graph.Edges {
		if e.Source == p.Edge.Source && e.Target == p.Edge.Target {
			return state, nil
		}
	}
	state.Graph.Edges = append(state.Graph.Edges, p.Edge)
	return state, nil
}

// Handle removes the edge with the given key, if present.
func (p RemoveEdgePayload) Handle(state Arc) (Arc, error) {
	kept := make(graph.Edges, 0, len(state.Graph.Edges))
	for _, e := range state.Graph.Edges {
		if e.Key == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Graph.Edges = kept
	return state, nil
}

// Handle rewrites the endpoints of the edge with the given key, preserving its
// key and kind. No-op when no edge with the key exists.
func (p ReconnectEdgePayload) Handle(state Arc) (Arc, error) {
	for i := range state.Graph.Edges {
		if state.Graph.Edges[i].Key == p.Key {
			state.Graph.Edges[i].Source = p.Source
			state.Graph.Edges[i].Target = p.Target
			break
		}
	}
	return state, nil
}

// Handle is the materialization seam for a character deletion. See InsertCharPayload.Handle.
func (p DeleteCharPayload) Handle(state Arc) (Arc, error) { return state, nil }

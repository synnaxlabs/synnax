// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import "github.com/google/uuid"

// ScopedAction wraps an action payload with the schematic key and originating
// session ID for broadcast. Clients use the session ID to skip their own
// broadcasts (self-dedup).
type ScopedAction struct {
	Key        uuid.UUID `json:"key" msgpack:"key"`
	SessionKey string    `json:"session_key" msgpack:"session_key"`
	Actions    []Action  `json:"actions" msgpack:"actions"`
}

func (s SetNodePosition) Handle(state Schematic) (Schematic, error) {
	for i, node := range state.Nodes {
		if node.Key == s.Key {
			state.Nodes[i].Position = s.Position
			break
		}
	}
	return state, nil
}

func (a AddNode) Handle(state Schematic) (Schematic, error) {
	state.Nodes = append(state.Nodes, a.Node)
	if a.Props != nil {
		if state.Props == nil {
			state.Props = make(map[string]any)
		}
		state.Props[a.Node.Key] = a.Props
	}
	return state, nil
}

func (r RemoveNode) Handle(state Schematic) (Schematic, error) {
	for i, node := range state.Nodes {
		if node.Key == r.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			break
		}
	}
	delete(state.Props, r.Key)
	return state, nil
}

func (s SetEdge) Handle(state Schematic) (Schematic, error) {
	for i, edge := range state.Edges {
		if edge.Key == s.Edge.Key {
			state.Edges[i] = s.Edge
			return state, nil
		}
	}
	state.Edges = append(state.Edges, s.Edge)
	return state, nil
}

func (r RemoveEdge) Handle(state Schematic) (Schematic, error) {
	for i, edge := range state.Edges {
		if edge.Key == r.Key {
			state.Edges = append(state.Edges[:i], state.Edges[i+1:]...)
			break
		}
	}
	return state, nil
}

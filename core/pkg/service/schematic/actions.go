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
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// ScopedAction wraps an action sequence with the targeted schematic key and the
// originating client's session key. Subscribers to the action signal channel
// compare SessionKey against their own client key to skip self-originated
// updates (optimistic-UI dedup).
type ScopedAction struct {
	Key        uuid.UUID `json:"key" msgpack:"key"`
	SessionKey string    `json:"session_key" msgpack:"session_key"`
	Actions    []Action  `json:"actions" msgpack:"actions"`
}

// Handle moves the named node to the given position. No-op if no node matches.
func (a SetNodePosition) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == a.Key {
			state.Nodes[i].Position = a.Position
			break
		}
	}
	return state, nil
}

// Handle appends the node and, if Props is non-nil, seeds the props map under
// the node's key.
func (a AddNode) Handle(state Schematic) (Schematic, error) {
	state.Nodes = append(state.Nodes, a.Node)
	if a.Props != nil {
		if state.Props == nil {
			state.Props = make(map[string]msgpack.EncodedJSON)
		}
		state.Props[a.Node.Key] = a.Props
	}
	return state, nil
}

// Handle removes the node with the matching key and discards any props entry
// stored under that key.
func (a RemoveNode) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == a.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			break
		}
	}
	delete(state.Props, a.Key)
	return state, nil
}

// Handle inserts the edge if no edge with the same key exists, otherwise
// replaces the existing edge in place.
func (a SetEdge) Handle(state Schematic) (Schematic, error) {
	for i := range state.Edges {
		if state.Edges[i].Key == a.Edge.Key {
			state.Edges[i] = a.Edge
			return state, nil
		}
	}
	state.Edges = append(state.Edges, a.Edge)
	return state, nil
}

// Handle removes the edge with the matching key. No-op if no edge matches.
func (a RemoveEdge) Handle(state Schematic) (Schematic, error) {
	for i := range state.Edges {
		if state.Edges[i].Key == a.Key {
			state.Edges = append(state.Edges[:i], state.Edges[i+1:]...)
			break
		}
	}
	return state, nil
}

// Handle sets the props entry for the given key, replacing any prior value.
func (a SetProps) Handle(state Schematic) (Schematic, error) {
	if state.Props == nil {
		state.Props = make(map[string]msgpack.EncodedJSON)
	}
	state.Props[a.Key] = a.Props
	return state, nil
}

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

//go:generate go run github.com/synnaxlabs/x/relapse/gen -state Schematic

import "github.com/synnaxlabs/x/spatial"

// AddNode is the payload for adding a node to the schematic.
type AddNode struct {
	Key   string         `json:"key"`
	Node  Node           `json:"node"`
	Props map[string]any `json:"props,omitempty"`
}

func (a AddNode) Handle(state Schematic) (Schematic, error) {
	state.Nodes = append(state.Nodes, a.Node)
	if a.Props != nil {
		if state.Props == nil {
			state.Props = make(map[string]map[string]any)
		}
		state.Props[a.Key] = a.Props
	}
	return state, nil
}

// SetNodeProps is the payload for setting node properties.
type SetNodeProps struct {
	Key   string         `json:"key"`
	Props map[string]any `json:"props"`
}

func (s SetNodeProps) Handle(state Schematic) (Schematic, error) {
	if state.Props == nil {
		state.Props = make(map[string]map[string]any)
	}
	state.Props[s.Key] = s.Props
	return state, nil
}

// SetNodePosition is the payload for setting node position.
type SetNodePosition struct {
	Key      string     `json:"key"`
	Position spatial.XY `json:"position"`
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

// RemoveNode is the payload for removing a node.
type RemoveNode struct {
	Key string `json:"key"`
}

func (r RemoveNode) Handle(state Schematic) (Schematic, error) {
	for i, node := range state.Nodes {
		if node.Key == r.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			break
		}
	}
	return state, nil
}

// RemoveEdge is the payload for removing an edge.
type RemoveEdge struct {
	Key string `json:"key"`
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

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

// Handle records the rendered pixel size of the named node. No-op if no node
// matches.
func (a SetNodeMeasured) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == a.Key {
			state.Nodes[i].Measured = a.Measured
			break
		}
	}
	return state, nil
}

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place. If Config is non-nil, it is stored
// under the node's key.
func (a SetNode) Handle(state Schematic) (Schematic, error) {
	replaced := false
	for i := range state.Nodes {
		if state.Nodes[i].Key == a.Node.Key {
			state.Nodes[i] = a.Node
			replaced = true
			break
		}
	}
	if !replaced {
		state.Nodes = append(state.Nodes, a.Node)
	}
	if a.Config != nil {
		if state.Configs == nil {
			state.Configs = make(map[string]msgpack.EncodedJSON)
		}
		state.Configs[a.Node.Key] = a.Config
	}
	return state, nil
}

// Handle removes the node with the matching key and discards any config entry
// stored under that key.
func (a RemoveNode) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == a.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			break
		}
	}
	delete(state.Configs, a.Key)
	return state, nil
}

// Handle appends the edge to the schematic. No-op when an edge with the
// same key already exists.
func (a AddEdge) Handle(state Schematic) (Schematic, error) {
	for i := range state.Edges {
		if state.Edges[i].Key == a.Edge.Key {
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

// Handle merges the payload config into the configs entry for the given key.
// Top-level fields present in the payload overwrite existing fields; fields
// absent from the payload are preserved. When no entry exists yet and the
// key matches an edge whose source node carries a color, the source color
// overrides whatever color (if any) was in the payload.
func (a SetConfig) Handle(state Schematic) (Schematic, error) {
	if state.Configs == nil {
		state.Configs = make(map[string]msgpack.EncodedJSON)
	}
	if existing := state.Configs[a.Key]; existing != nil {
		merged := make(msgpack.EncodedJSON, len(existing)+len(a.Config))
		for k, v := range existing {
			merged[k] = v
		}
		for k, v := range a.Config {
			merged[k] = v
		}
		state.Configs[a.Key] = merged
		return state, nil
	}
	cfg := a.Config
	for _, e := range state.Edges {
		if e.Key != a.Key {
			continue
		}
		srcCfg := state.Configs[e.Source.Node]
		if srcCfg == nil {
			break
		}
		c, ok := srcCfg["color"]
		if !ok || c == nil {
			break
		}
		next := make(msgpack.EncodedJSON, len(cfg)+1)
		for k, v := range cfg {
			next[k] = v
		}
		next["color"] = c
		cfg = next
		break
	}
	state.Configs[a.Key] = cfg
	return state, nil
}

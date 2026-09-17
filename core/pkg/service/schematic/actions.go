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
	"maps"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/schematic/versions"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// Handle replaces the document with its created state.
func (p CreatePayload) Handle(Schematic) (Schematic, error) {
	return p.Schematic, nil
}

// Handle replaces the schematic's name.
func (p RenamePayload) Handle(state Schematic) (Schematic, error) {
	state.Name = p.Name
	return state, nil
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

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place. If Config is non-nil, it is stored
// under the node's key.
func (p SetNodePayload) Handle(state Schematic) (Schematic, error) {
	replaced := false
	for i := range state.Nodes {
		if state.Nodes[i].Key == p.Node.Key {
			state.Nodes[i] = p.Node
			replaced = true
			break
		}
	}
	if !replaced {
		state.Nodes = append(state.Nodes, p.Node)
	}
	if p.Config != nil {
		cfg, err := versions.DecodeElementConfig(versions.NormalizeConfigKeys(p.Config))
		if err != nil {
			return state, err
		}
		if state.Configs == nil {
			state.Configs = make(map[string]ElementConfig)
		}
		state.Configs[p.Node.Key] = cfg
	}
	return state, nil
}

// Handle removes the node with the matching key, discards any config entry
// stored under that key, and splices the key out of every group's members.
func (p RemoveNodePayload) Handle(state Schematic) (Schematic, error) {
	removed := false
	for i := range state.Nodes {
		if state.Nodes[i].Key == p.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			removed = true
			break
		}
	}
	delete(state.Configs, p.Key)
	if !removed {
		return state, nil
	}
	for key, cfg := range state.Configs {
		group, ok := cfg.Variant.(GroupBoxElementConfig)
		if !ok {
			continue
		}
		members := slices.DeleteFunc(
			slices.Clone(group.Members),
			func(m string) bool { return m == p.Key },
		)
		if len(members) == len(group.Members) {
			continue
		}
		group.Members = members
		state.Configs[key] = ElementConfig{Variant: group}
	}
	return state, nil
}

// Handle appends the edge to the schematic. No-op when an edge with the
// same key already exists.
func (p AddEdgePayload) Handle(state Schematic) (Schematic, error) {
	for i := range state.Edges {
		if state.Edges[i].Key == p.Edge.Key {
			return state, nil
		}
	}
	state.Edges = append(state.Edges, p.Edge)
	return state, nil
}

// Handle removes the edge with the matching key and discards any config entry
// stored under that key.
func (p RemoveEdgePayload) Handle(state Schematic) (Schematic, error) {
	for i := range state.Edges {
		if state.Edges[i].Key == p.Key {
			state.Edges = append(state.Edges[:i], state.Edges[i+1:]...)
			break
		}
	}
	delete(state.Configs, p.Key)
	return state, nil
}

// Handle merges the payload config into the configs entry for the given key.
// Top-level fields present in the payload overwrite existing fields; fields
// absent from the payload are preserved. When no entry exists yet and the
// key matches an edge whose source node carries a color, the source color
// overrides whatever color (if any) was in the payload.
func (p SetConfigPayload) Handle(state Schematic) (Schematic, error) {
	if existing, ok := state.Configs[p.Key]; ok {
		fields, err := versions.ElementConfigFields(existing)
		if err != nil {
			return state, err
		}
		maps.Copy(fields, versions.NormalizeConfigKeys(p.Config))
		merged, err := versions.DecodeElementConfig(fields)
		if err != nil {
			return state, err
		}
		state.Configs[p.Key] = merged
		return state, nil
	}
	raw := versions.NormalizeConfigKeys(p.Config)
	for _, e := range state.Edges {
		if e.Key != p.Key {
			continue
		}
		srcCfg, ok := state.Configs[e.Source.Node]
		if !ok {
			break
		}
		srcFields, err := versions.ElementConfigFields(srcCfg)
		if err != nil {
			return state, err
		}
		c, ok := srcFields["color"]
		if !ok || c == nil {
			break
		}
		next := make(msgpack.EncodedJSON, len(raw)+1)
		maps.Copy(next, raw)
		next["color"] = c
		raw = next
		break
	}
	cfg, err := versions.DecodeElementConfig(raw)
	if err != nil {
		return state, err
	}
	if state.Configs == nil {
		state.Configs = make(map[string]ElementConfig)
	}
	state.Configs[p.Key] = cfg
	return state, nil
}

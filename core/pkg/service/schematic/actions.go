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
	"encoding/json"
	"maps"

	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

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

// Handle records the rendered pixel size of the named node. No-op if no node
// matches.
func (p SetNodeMeasuredPayload) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == p.Key {
			state.Nodes[i].Measured = p.Measured
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
		cfg, err := decodeElementConfig(normalizeConfigKeys(p.Config))
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

// decodeElementConfig validates an opaque config payload against the element
// config union. It returns a validate-style error when the payload's variant
// is unknown or its fields do not conform to the variant's shape.
func decodeElementConfig(raw msgpack.EncodedJSON) (ElementConfig, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return ElementConfig{}, err
	}
	var cfg ElementConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ElementConfig{}, errors.Wrap(err, "invalid element config")
	}
	return cfg, nil
}

// elementConfigFields returns a config's wire fields as an opaque map for
// partial merging.
func elementConfigFields(cfg ElementConfig) (msgpack.EncodedJSON, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m msgpack.EncodedJSON
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Handle removes the node with the matching key and discards any config entry
// stored under that key.
func (p RemoveNodePayload) Handle(state Schematic) (Schematic, error) {
	for i := range state.Nodes {
		if state.Nodes[i].Key == p.Key {
			state.Nodes = append(state.Nodes[:i], state.Nodes[i+1:]...)
			break
		}
	}
	delete(state.Configs, p.Key)
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
		fields, err := elementConfigFields(existing)
		if err != nil {
			return state, err
		}
		maps.Copy(fields, normalizeConfigKeys(p.Config))
		merged, err := decodeElementConfig(fields)
		if err != nil {
			return state, err
		}
		state.Configs[p.Key] = merged
		return state, nil
	}
	raw := normalizeConfigKeys(p.Config)
	for _, e := range state.Edges {
		if e.Key != p.Key {
			continue
		}
		srcCfg, ok := state.Configs[e.Source.Node]
		if !ok {
			break
		}
		srcFields, err := elementConfigFields(srcCfg)
		if err != nil {
			return state, err
		}
		c, ok := srcFields["color"]
		if !ok || c == nil {
			break
		}
		if b, err := json.Marshal(c); err == nil {
			var srcColor color.Color
			if err := json.Unmarshal(b, &srcColor); err != nil || srcColor.IsZero() {
				break
			}
		}
		next := make(msgpack.EncodedJSON, len(raw)+1)
		maps.Copy(next, raw)
		next["color"] = c
		raw = next
		break
	}
	cfg, err := decodeElementConfig(raw)
	if err != nil {
		return state, err
	}
	if state.Configs == nil {
		state.Configs = make(map[string]ElementConfig)
	}
	state.Configs[p.Key] = cfg
	return state, nil
}

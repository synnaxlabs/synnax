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
	"encoding/json/v2"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

const configsField = "configs"

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

// Handle inserts the node if no node with the same key exists, otherwise
// replaces the existing node in place. If Config is non-nil, it is stored
// under the node's key.
func (p SetNodePayload) Handle(state Schematic) (Schematic, error) {
	if p.Config != nil && p.Config.Variant == nil {
		return Schematic{}, noVariantError(p.Node.Key)
	}
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
		if state.Configs == nil {
			state.Configs = make(map[string]NodeConfig)
		}
		state.Configs[p.Node.Key] = *p.Config
	}
	return state, nil
}

// Handle removes the node with the matching key, its config, and the edges that connect
// to it, and splices the key out of every group's members. No-op if no node matches.
func (p RemoveNodePayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Nodes, func(n Node) bool { return n.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	state.Nodes = slices.Delete(slices.Clone(state.Nodes), i, i+1)
	delete(state.Configs, p.Key)
	kept := make([]Edge, 0, len(state.Edges))
	for _, e := range state.Edges {
		if e.Source.Node == p.Key || e.Target.Node == p.Key {
			continue
		}
		kept = append(kept, e)
	}
	state.Edges = kept
	for key, cfg := range state.Configs {
		group, ok := cfg.Variant.(GroupBoxNodeConfig)
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
		state.Configs[key] = NodeConfig{Variant: group}
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
	state.Edges = slices.Delete(slices.Clone(state.Edges), i, i+1)
	return state, nil
}

// Handle inserts the edge if no edge with the same key exists, otherwise replaces the
// existing edge in place. No-op when the source or target node does not exist.
func (p SetEdgePayload) Handle(state Schematic) (Schematic, error) {
	if !hasNode(state, p.Edge.Source.Node) || !hasNode(state, p.Edge.Target.Node) {
		return state, nil
	}
	i := slices.IndexFunc(state.Edges, func(e Edge) bool { return e.Key == p.Edge.Key })
	state.Edges = slices.Clone(state.Edges)
	if i == -1 {
		state.Edges = append(state.Edges, p.Edge)
	} else {
		state.Edges[i] = p.Edge
	}
	return state, nil
}

// Handle replaces the segments of the edge with the matching key. No-op if no edge
// matches.
func (p SetEdgeSegmentsPayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Edges, func(e Edge) bool { return e.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	state.Edges = slices.Clone(state.Edges)
	state.Edges[i].Segments = p.Segments
	return state, nil
}

// Handle sets each style field that the payload holds on the edge with the matching
// key. No-op if no edge matches.
func (p SetEdgeStylePayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Edges, func(e Edge) bool { return e.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	state.Edges = slices.Clone(state.Edges)
	e := &state.Edges[i]
	if p.Variant != nil {
		e.Variant = *p.Variant
	}
	if p.StrokeColor != nil {
		e.StrokeColor = p.StrokeColor
	}
	if p.StrokeWidth != nil {
		e.StrokeWidth = *p.StrokeWidth
	}
	return state, nil
}

// Handle removes the stroke color of the edge with the matching key. No-op if no edge
// matches.
func (p ClearEdgeStrokeColorPayload) Handle(state Schematic) (Schematic, error) {
	i := slices.IndexFunc(state.Edges, func(e Edge) bool { return e.Key == p.Key })
	if i == -1 {
		return state, nil
	}
	state.Edges = slices.Clone(state.Edges)
	state.Edges[i].StrokeColor = nil
	return state, nil
}

// Handle replaces the configs entry stored under the given key. A config naming no
// variant is rejected: stored, it would be a null entry no client can read back. No-op
// when no node has the key.
func (p SetConfigPayload) Handle(state Schematic) (Schematic, error) {
	if p.Config.Variant == nil {
		return Schematic{}, noVariantError(p.Key)
	}
	if !hasNode(state, p.Key) {
		return state, nil
	}
	if state.Configs == nil {
		state.Configs = make(map[string]NodeConfig)
	}
	state.Configs[p.Key] = p.Config
	return state, nil
}

func hasNode(state Schematic, key string) bool {
	return slices.ContainsFunc(state.Nodes, func(n Node) bool { return n.Key == key })
}

func hasEdge(state Schematic, key string) bool {
	return slices.ContainsFunc(state.Edges, func(e Edge) bool { return e.Key == key })
}

func noVariantError(key string) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"[Schematic] - config for %q names no variant",
		key,
	)
}

// rawSetNodePayload is the wire shape of SetNodePayload. The config stays raw so the
// decode can tell a field the client omitted from one it set to zero.
type rawSetNodePayload struct {
	// Node is the node to insert or replace.
	Node Node `json:"node" msgpack:"node"`
	// Config is the node's config as the client sent it. Nil when the payload carries
	// none, which leaves the stored config untouched.
	Config msgpack.EncodedJSON `json:"config,omitzero" msgpack:"config,omitempty"`
}

// rawSetConfigPayload is the wire shape of SetConfigPayload. See rawSetNodePayload.
type rawSetConfigPayload struct {
	// Key is the node key the config is stored under.
	Key string `json:"key" msgpack:"key"`
	// Config is the config as the client sent it. It must name a variant.
	Config msgpack.EncodedJSON `json:"config" msgpack:"config"`
}

// UnmarshalJSON decodes the payload with the schema defaults filled for every config
// field it omits. A field the payload carries keeps its value, zero included.
func (p *SetNodePayload) UnmarshalJSON(data []byte) error {
	var raw rawSetNodePayload
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	return p.fromRaw(raw)
}

// UnmarshalMsgpack decodes the payload the way UnmarshalJSON does.
func (p *SetNodePayload) UnmarshalMsgpack(data []byte) error {
	var raw rawSetNodePayload
	if err := msgpack.Codec.Decode(data, &raw); err != nil {
		return err
	}
	return p.fromRaw(raw)
}

func (p *SetNodePayload) fromRaw(raw rawSetNodePayload) error {
	*p = SetNodePayload{Node: raw.Node}
	if raw.Config == nil {
		return nil
	}
	cfg, err := decodeConfig(raw.Config)
	if err != nil {
		return err
	}
	p.Config = &cfg
	return nil
}

// UnmarshalJSON decodes the payload with the schema defaults filled for every config
// field it omits. A field the payload carries keeps its value, zero included.
func (p *SetConfigPayload) UnmarshalJSON(data []byte) error {
	var raw rawSetConfigPayload
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	return p.fromRaw(raw)
}

// UnmarshalMsgpack decodes the payload the way UnmarshalJSON does.
func (p *SetConfigPayload) UnmarshalMsgpack(data []byte) error {
	var raw rawSetConfigPayload
	if err := msgpack.Codec.Decode(data, &raw); err != nil {
		return err
	}
	return p.fromRaw(raw)
}

func (p *SetConfigPayload) fromRaw(raw rawSetConfigPayload) error {
	cfg, err := decodeConfig(raw.Config)
	if err != nil {
		return err
	}
	*p = SetConfigPayload{Key: raw.Key, Config: cfg}
	return nil
}

func decodeConfig(raw msgpack.EncodedJSON) (NodeConfig, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return NodeConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid node config: %s", err,
		)
	}
	var cfg NodeConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return NodeConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid node config: %s", err,
		)
	}
	return cfg, nil
}

// Handle applies the set to the schematic's name, group, project, or node configs.
func (p SetPayload) Handle(state Schematic) (Schematic, error) {
	return edit(state, p.Path, patch.Set(p))
}

// Handle applies the remove to the schematic's name, group, project, or node configs.
func (p RemovePayload) Handle(state Schematic) (Schematic, error) {
	return edit(state, append(slices.Clone(p.Path), p.Key), patch.Remove(p))
}

// Handle applies the insert to the schematic's name, group, project, or node configs.
func (p InsertPayload) Handle(state Schematic) (Schematic, error) {
	return edit(state, p.Path, patch.Insert(p))
}

// Handle applies the move to the schematic's name, group, project, or node configs.
func (p MovePayload) Handle(state Schematic) (Schematic, error) {
	return edit(state, p.Path, patch.Move(p))
}

// document is the part of a schematic, outside its node configs, that path actions
// edit.
type document struct {
	Name    string       `json:"name"`
	Group   *group.Key   `json:"group,omitzero"`
	Project *project.Key `json:"project,omitzero"`
}

// edit applies a path action to the schematic's name, group, or project, or to the node
// config that its path names. An action on the config of a node that does not exist
// does nothing. A config that names no variant or fails its schema is rejected, as is a
// path to another field.
func edit(state Schematic, path []string, a patch.Action) (Schematic, error) {
	if patch.Field(a) != configsField {
		d, err := patch.Edit(
			document{Name: state.Name, Group: state.Group, Project: state.Project},
			a.Apply,
		)
		if err != nil {
			return Schematic{}, err
		}
		state.Name, state.Group, state.Project = d.Name, d.Group, d.Project
		return state, nil
	}
	if len(path) > 1 && !hasNode(state, path[1]) {
		return state, nil
	}
	configs, err := patch.EditMap(state.Configs, []string{configsField}, a, checkConfig)
	if err != nil {
		return Schematic{}, err
	}
	state.Configs = configs
	return state, nil
}

func newSetAction(s patch.Set) Action { return NewSetAction(SetPayload(s)) }

func newRemoveAction(r patch.Remove) Action { return NewRemoveAction(RemovePayload(r)) }

func checkConfig(key string, cfg NodeConfig) error {
	if cfg.Variant == nil {
		return noVariantError(key)
	}
	return validate.PathedError(cfg.Validate(), configsField, key)
}

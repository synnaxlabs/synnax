// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy decodes Console arc state files ("0.0.0".."2.0.0") and lifts them
// into the typed document parts of an Arc. v0 states persist edges in ReactFlow's
// flat form; v1 and v2 share the nested-handle form and differ only by the
// set_status props rename applied on the way through. "3.0.0" states park the
// document under pendingUpload instead and are handled by the importer directly.
package legacy

import (
	"encoding/json"
	"maps"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
)

// Document is the typed body a Console state lifts into. Mode is the raw wire
// string ("graph" or "text"); the importer converts it to the typed enum.
type Document struct {
	Graph graph.Graph
	Text  text.Text
	Mode  string
}

// state is the shared shape of Console arc states v0-v2. The edge form differs
// between v0 and v1, so edges stay raw until the version is known.
type state struct {
	Version string     `json:"version"`
	Graph   stateGraph `json:"graph"`
	Text    stateText  `json:"text"`
	Mode    string     `json:"mode"`
}

type stateText struct {
	Raw string `json:"raw"`
}

type stateGraph struct {
	Nodes []stateNode               `json:"nodes"`
	Edges json.RawMessage           `json:"edges"`
	Props map[string]map[string]any `json:"props"`
}

type stateNode struct {
	Key      string     `json:"key"`
	Position spatial.XY `json:"position"`
}

// flatEdge is the v0 edge form: ReactFlow's node-key strings with optional
// sibling handle fields.
type flatEdge struct {
	Key          string  `json:"key"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	SourceHandle *string `json:"sourceHandle"`
	TargetHandle *string `json:"targetHandle"`
}

// handleEdge is the v1/v2 edge form with nested Handle objects.
type handleEdge struct {
	Key    string    `json:"key"`
	Source ir.Handle `json:"source"`
	Target ir.Handle `json:"target"`
}

// Migrate decodes blob as a Console arc state, dispatching on the version string
// inside the body. States below "2.0.0" have their deprecated set_status node
// props renamed to status.set, mirroring the Console migration for the removed
// STL symbol.
func Migrate(blob msgpack.EncodedJSON) (Document, error) {
	var s state
	if err := blob.Unmarshal(&s); err != nil {
		return Document{}, errors.Wrap(err, "decode arc state")
	}
	var (
		edges graph.Edges
		err   error
	)
	switch s.Version {
	case "2.0.0", "1.0.0":
		edges, err = liftHandleEdges(s.Graph.Edges)
	case "0.0.0", "":
		edges, err = liftFlatEdges(s.Graph.Edges)
	default:
		return Document{}, errors.Newf("unknown arc state version %q", s.Version)
	}
	if err != nil {
		return Document{}, err
	}
	props := s.Graph.Props
	if s.Version != "2.0.0" {
		props = rewriteSetStatus(props)
	}
	inputs := liftInputs(props)
	nodes := make(graph.Nodes, len(s.Graph.Nodes))
	for i, n := range s.Graph.Nodes {
		nodes[i] = graph.Node{Key: n.Key, Position: n.Position}
	}
	return Document{
		Graph: graph.Graph{Nodes: nodes, Edges: edges, Inputs: inputs},
		Text:  text.Text{Raw: s.Text.Raw},
		Mode:  s.Mode,
	}, nil
}

func liftHandleEdges(raw json.RawMessage) (graph.Edges, error) {
	var in []handleEdge
	if err := decodeEdges(raw, &in); err != nil {
		return nil, err
	}
	out := make(graph.Edges, len(in))
	for i, e := range in {
		out[i] = graph.Edge{
			Key: e.Key,
			Edge: ir.Edge{
				Source: e.Source,
				Target: e.Target,
				Kind:   ir.EdgeKindContinuous,
			},
		}
	}
	return out, nil
}

func liftFlatEdges(raw json.RawMessage) (graph.Edges, error) {
	var in []flatEdge
	if err := decodeEdges(raw, &in); err != nil {
		return nil, err
	}
	out := make(graph.Edges, len(in))
	for i, e := range in {
		out[i] = graph.Edge{
			Key: e.Key,
			Edge: ir.Edge{
				Source: ir.Handle{Node: e.Source, Param: stringOrEmpty(e.SourceHandle)},
				Target: ir.Handle{Node: e.Target, Param: stringOrEmpty(e.TargetHandle)},
				Kind:   ir.EdgeKindContinuous,
			},
		}
	}
	return out, nil
}

func decodeEdges(raw json.RawMessage, into any) error {
	if len(raw) == 0 {
		return nil
	}
	return errors.Wrap(json.Unmarshal(raw, into), "decode arc state edges")
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// rewriteSetStatus renames the deprecated set_status node props to status.set.
// The bare set_status STL symbol was removed; saved graphs must remap the props.
func rewriteSetStatus(
	props map[string]map[string]any,
) map[string]map[string]any {
	out := make(map[string]map[string]any, len(props))
	for k, p := range props {
		if p["key"] != "set_status" {
			out[k] = p
			continue
		}
		out[k] = map[string]any{
			"key":         "status.set",
			"key_or_name": stringOr(p["statusKey"], ""),
			"variant":     stringOr(p["variant"], "success"),
			"message":     stringOr(p["message"], ""),
		}
	}
	return out
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return def
}

// liftInputs converts per-node props (function type under "key") into the typed
// graph inputs map (function type under "type"), keyed by node key.
func liftInputs(props map[string]map[string]any) map[string]msgpack.EncodedJSON {
	if len(props) == 0 {
		return nil
	}
	out := make(map[string]msgpack.EncodedJSON, len(props))
	for k, p := range props {
		entry := make(map[string]any, len(p))
		maps.Copy(entry, p)
		entry["type"] = entry["key"]
		delete(entry, "key")
		out[k] = msgpack.EncodedJSON(entry)
	}
	return out
}

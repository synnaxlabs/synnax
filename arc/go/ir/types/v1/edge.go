// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/vmihailenco/msgpack/v5"
)

// Predicate constructors for edge queries.
func sourceEquals(handle Handle) func(Edge) bool {
	return func(e Edge) bool { return e.Source == handle }
}

func targetEquals(handle Handle) func(Edge) bool {
	return func(e Edge) bool { return e.Target == handle }
}

func targetNodeEquals(nodeKey string) func(Edge, int) bool {
	return func(e Edge, _ int) bool { return e.Target.Node == nodeKey }
}

func sourceNodeEquals(nodeKey string) func(Edge, int) bool {
	return func(e Edge, _ int) bool { return e.Source.Node == nodeKey }
}

// GetBySource returns the edge with the given source handle. Panics if not found.
func (e Edges) GetBySource(handle Handle) Edge {
	return e.get(sourceEquals(handle))
}

// GetByTarget returns the edge with the given target handle. Panics if not found.
func (e Edges) GetByTarget(handle Handle) Edge {
	return e.get(targetEquals(handle))
}

func (e Edges) find(f func(e Edge) bool) (Edge, bool) {
	return lo.Find(e, f)
}

func (e Edges) get(f func(e Edge) bool) Edge {
	return lo.Must(e.find(f))
}

func (e Edges) filter(f func(e Edge, _ int) bool) []Edge {
	return lo.Filter(e, f)
}

// FindBySource searches for an edge with the given source handle.
func (e Edges) FindBySource(handle Handle) (Edge, bool) {
	return e.find(sourceEquals(handle))
}

// FindByTarget searches for an edge with the given target handle.
func (e Edges) FindByTarget(handle Handle) (Edge, bool) {
	return e.find(targetEquals(handle))
}

// GetInputs returns all edges targeting the given node.
func (e Edges) GetInputs(nodeKey string) []Edge {
	return e.filter(targetNodeEquals(nodeKey))
}

// GetOutputs returns all edges sourced from the given node.
func (e Edges) GetOutputs(nodeKey string) []Edge {
	return e.filter(sourceNodeEquals(nodeKey))
}

// GetByKind returns all edges with the specified kind.
func (e Edges) GetByKind(kind EdgeKind) Edges {
	return lo.Filter(e, func(edge Edge, _ int) bool { return edge.Kind == kind })
}

// String returns the string representation of the edge.
// Format: "source.param -> target.param (continuous)" or "source.param => target.param (conditional)"
func (e Edge) String() string {
	arrow := "->"
	if e.Kind == EdgeKindConditional {
		arrow = "=>"
	}
	return fmt.Sprintf("%s %s %s (%s)", e.Source, arrow, e.Target, e.Kind)
}

// DecodeMsgpack implements msgpack.CustomDecoder, supporting both legacy uppercase
// Go field names and new lowercase msgpack tag names for backward compatibility.
func (e *Edge) DecodeMsgpack(dec *msgpack.Decoder) error {
	type alias Edge
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if err = msgpack.Unmarshal(raw, (*alias)(e)); err != nil {
		return err
	}
	if len(e.Source.Node) == 0 {
		var legacy struct {
			Source Handle
			Target Handle
			Kind   EdgeKind
		}
		if err = msgpack.Unmarshal(raw, &legacy); err != nil {
			return err
		}
		e.Source = legacy.Source
		e.Target = legacy.Target
		e.Kind = legacy.Kind
	}
	return nil
}

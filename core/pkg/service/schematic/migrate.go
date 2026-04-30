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
	"context"
	"encoding/json"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v3"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v5"
	v55 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v55"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
)

// MigrateSchematic transforms the previous schematic snapshot (v55) into the
// current strongly-typed Schematic. AutoMigrateSchematic handles the
// trivially-copyable gorp-entry fields (Key, Name, Snapshot); the body
// fields are sourced from the per-schematic blob the console used to
// persist alongside those gorp fields, after v5.Lift walks the legacy
// migration chain owned by each migrations/legacy/v* subfolder. UI-only
// fields (editable, fitViewOnResize, viewport, mode, toolbar, control,
// viewportMode, the wire-format key) are dropped; Authority defaults to 1
// when the source carries zero. Edges flip from the flat source /
// sourceHandle pair into nested Handle objects, edge.data segments / color
// / variant lift into the props map keyed by edge id, and node-prop "key"
// renames to "variant" so the lifted shape matches the EdgeProps /
// NodeProps schema declared in schematic.oracle. v55 is the last snapshot
// in which Schematic.Data is untyped; future migrations transform one
// typed snapshot into another and never need this blob handling.
func MigrateSchematic(ctx context.Context, old v55.Schematic) (Schematic, error) {
	out, err := AutoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	d, err := v5.Lift(old.Data)
	if err != nil {
		return Schematic{}, err
	}
	out.Nodes = make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		out.Nodes[i] = migrateNode(n)
	}
	out.Props, err = migrateProps(d.Props)
	if err != nil {
		return Schematic{}, err
	}
	out.Edges = make([]Edge, len(d.Edges))
	for i, e := range d.Edges {
		edge, edgeProps, err := migrateEdge(e)
		if err != nil {
			return Schematic{}, errors.Wrapf(err, "edge %s", e.Key)
		}
		out.Edges[i] = edge
		if edgeProps != nil {
			if out.Props == nil {
				out.Props = make(map[string]msgpack.EncodedJSON)
			}
			out.Props[edge.Key] = edgeProps
		}
	}
	out.Authority = control.Authority(d.Authority)
	if out.Authority == 0 {
		out.Authority = 1
	}
	out.Legend = migrateLegend(d.Legend)
	return out, nil
}

func migrateNode(n v0.Node) Node {
	out := Node{
		Key:      n.Key,
		Position: spatial.XY{X: n.Position.X, Y: n.Position.Y},
	}
	if n.ZIndex != nil {
		out.ZIndex = int16(*n.ZIndex)
	}
	return out
}

// migrateEdge reshapes a v5 edge into the typed Edge with nested Handles and,
// when the edge carries a ReactFlow-style data bag, lifts its segments /
// color / variant fields into a props map entry keyed by the edge id.
// Returns the typed edge plus an EncodedJSON payload (or nil when there is
// nothing to lift).
func migrateEdge(e v3.Edge) (Edge, msgpack.EncodedJSON, error) {
	out := Edge{
		Key:    e.Key,
		Source: Handle{Node: e.Source, Param: stringOrEmpty(e.SourceHandle)},
		Target: Handle{Node: e.Target, Param: stringOrEmpty(e.TargetHandle)},
	}
	if len(e.Data) == 0 {
		return out, nil, nil
	}
	var bag map[string]any
	if err := json.Unmarshal(e.Data, &bag); err != nil {
		return out, nil, errors.Wrap(err, "decode edge data bag")
	}
	lifted := make(map[string]any, 3)
	for _, key := range []string{"segments", "color", "variant"} {
		if v, ok := bag[key]; ok && v != nil {
			lifted[key] = v
		}
	}
	if len(lifted) == 0 {
		return out, nil, nil
	}
	return out, lifted, nil
}

func migrateLegend(l v1.Legend) Legend {
	out := Legend{
		Visible: l.Visible,
		Position: spatial.StickyXY{
			X: l.Position.X,
			Y: l.Position.Y,
		},
	}
	if l.Position.Units != nil {
		out.Position.Units.X = spatial.StickyUnit(l.Position.Units.X)
		out.Position.Units.Y = spatial.StickyUnit(l.Position.Units.Y)
	}
	if l.Position.Root != nil {
		out.Position.Root.X = spatial.XLocation(l.Position.Root.X)
		out.Position.Root.Y = spatial.YLocation(l.Position.Root.Y)
	}
	if len(l.Colors) > 0 {
		out.Colors = make(map[string]color.Color, len(l.Colors))
		for k, hex := range l.Colors {
			c, err := color.FromHex(hex)
			if err != nil {
				continue
			}
			out.Colors[k] = c
		}
	}
	return out
}

// migrateProps decodes each opaque prop entry from raw JSON bytes into the
// in-memory map[string]any shape that msgpack.EncodedJSON wraps, renaming
// the v0..v5 node-prop "key" field to "variant" to match the v6 NodeProps
// schema declared in schematic.oracle. Empty entries are dropped because
// msgpack.EncodedJSON is nil-equivalent to "no entry".
func migrateProps(in map[string]json.RawMessage) (map[string]msgpack.EncodedJSON, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make(map[string]msgpack.EncodedJSON, len(in))
	for k, raw := range in {
		if len(raw) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, errors.Wrapf(err, "decode props[%q]", k)
		}
		if v, ok := m["key"]; ok {
			if _, hasVariant := m["variant"]; !hasVariant {
				m["variant"] = v
			}
			delete(m, "key")
		}
		out[k] = m
	}
	return out, nil
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7

import (
	"context"
	"encoding/json"
	"math"

	"github.com/synnaxlabs/synnax/pkg/service/schematic/types/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/types/legacy/v0"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/types/legacy/v3"
	schematicv0 "github.com/synnaxlabs/synnax/pkg/service/schematic/types/v6"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/spatial"
)

// migrateSchematic transforms the previous schematic snapshot (v0) into the
// current strongly-typed Schematic. AutoMigrateSchematic handles the
// trivially-copyable gorp-entry fields (Key, Name, Snapshot); the body
// fields are sourced from the per-schematic blob the console used to
// persist alongside those gorp fields, after legacy.MigrateData walks the
// legacy migration chain up to v5.Data. UI-only fields (editable,
// fitViewOnResize, viewport, mode, toolbar, control, viewportMode, authority,
// legend, the wire-format key) are dropped; the latter live on the console
// slice and never reach the server. Edges flip from the flat source /
// sourceHandle pair into nested Handle objects, edge.data segments / color /
// variant lift into the props map keyed by edge id, and node-prop "key"
// renames to "variant" so the lifted shape matches the EdgeProps / NodeProps
// schema declared in schematic.oracle. v0 is the last snapshot in which
// Schematic.Data is untyped; future migrations transform one typed snapshot
// into another and never need this blob handling.
func migrateSchematic(
	ctx context.Context,
	old schematicv0.Schematic,
) (Schematic, error) {
	out, err := AutoMigrateSchematic(ctx, old)
	if err != nil {
		return Schematic{}, err
	}
	d, err := legacy.MigrateData(old.Data)
	if err != nil {
		return Schematic{}, err
	}
	out.Nodes = make([]Node, len(d.Nodes))
	for i, n := range d.Nodes {
		out.Nodes[i] = migrateNode(n)
	}
	out.Configs, err = migrateProps(d.Props)
	if err != nil {
		return Schematic{}, err
	}
	if out.Configs == nil {
		out.Configs = make(map[string]msgpack.EncodedJSON)
	}
	out.Edges = make([]Edge, len(d.Edges))
	for i, e := range d.Edges {
		edge, edgeProps, err := migrateEdge(e)
		if err != nil {
			return Schematic{}, errors.Wrapf(err, "edge %s", e.Key)
		}
		out.Edges[i] = edge
		if edgeProps != nil {
			out.Configs[edge.Key] = edgeProps
		}
	}
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
// Mirrors the console v6 migrateEdge: a missing or null data field produces
// no edgeProps; any non-null data object produces an EncodedJSON payload
// with EdgeProps defaults applied (segments defaults to [], variant defaults
// to "pipe") so the lifted shape always parses cleanly under the v6
// EdgeProps schema. Returns the typed edge plus the payload (or nil when
// there is nothing to lift).
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
	if bag == nil {
		return out, nil, nil
	}
	lifted := map[string]any{
		"segments": []any{},
		"variant":  "pipe",
	}
	for _, key := range []string{"segments", "color", "variant"} {
		if v, ok := bag[key]; ok && v != nil {
			lifted[key] = v
		}
	}
	if raw, ok := lifted["segments"].([]any); ok {
		lifted["segments"] = stripLegacyStumps(raw)
	}
	return out, lifted, nil
}

// stumpLength mirrors STUMP_LENGTH in the pluto segmented connector: the fixed
// length of the auto-generated stub that leaves each handle.
const stumpLength = 10.0

// segment is the decoded form of one entry in an edge's segments list.
type segment struct {
	direction string
	length    float64
}

// stripLegacyStumps converts a pre-0.56 full edge path into the middle-only form
// the current renderer expects. Pre-0.56 edges stored the complete path including
// both stumps; the current model stores only the middle and re-derives the stumps
// on render, so leaving them in doubles the stumps and folds a pigtail over the
// target. The stumps are the first and last segments. Mirrors stripLegacyStumps in
// the console v6 migration. The input is returned unchanged if it does not parse as
// a segment list.
func stripLegacyStumps(raw []any) []any {
	segs, ok := parseSegments(raw)
	if !ok || len(segs) == 0 {
		return raw
	}
	if hasNoStrippableMiddle(segs) {
		return []any{}
	}
	return segmentsToRaw(extractMiddle(segs))
}

// hasNoStrippableMiddle reports whether an edge's stumps overlap, leaving no middle
// to preserve: a single segment shorter than two stumps, or a first/last segment
// shorter than one stump. Subtracting a full stump from those would flip the segment
// and fold a spur, so such edges are cleared to auto-route instead.
func hasNoStrippableMiddle(segs []segment) bool {
	if len(segs) == 1 {
		return math.Abs(segs[0].length) <= 2*stumpLength-0.5
	}
	return math.Abs(segs[0].length) <= stumpLength-0.5 ||
		math.Abs(segs[len(segs)-1].length) <= stumpLength-0.5
}

// extractMiddle removes the source and target stumps from a full path, leaving the
// middle segments. The stumps share the direction and sign of the first and last
// segments, so a same-signed stump length is subtracted from each end; a resulting
// near-zero segment is dropped entirely.
func extractMiddle(segs []segment) []segment {
	result := append([]segment(nil), segs...)
	srcStump := sign(result[0].length) * stumpLength
	if rem := result[0].length - srcStump; math.Abs(rem) < 0.5 {
		result = result[1:]
	} else {
		result[0].length = rem
	}
	if len(result) == 0 {
		return result
	}
	tgtStump := sign(segs[len(segs)-1].length) * stumpLength
	last := len(result) - 1
	if rem := result[last].length - tgtStump; math.Abs(rem) < 0.5 {
		result = result[:last]
	} else {
		result[last].length = rem
	}
	return result
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// parseSegments decodes the opaque segments list into typed segments, returning
// false if any entry is not a well-formed {direction, length} object.
func parseSegments(raw []any) ([]segment, bool) {
	out := make([]segment, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		direction, ok := m["direction"].(string)
		if !ok {
			return nil, false
		}
		length, ok := m["length"].(float64)
		if !ok {
			return nil, false
		}
		out = append(out, segment{direction: direction, length: length})
	}
	return out, true
}

func segmentsToRaw(segs []segment) []any {
	out := make([]any, len(segs))
	for i, s := range segs {
		out[i] = map[string]any{"direction": s.direction, "length": s.length}
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
		// Mirrors the console v6 migrateProps: variant is always set from
		// the v0..v5 "key" field, overwriting any prior variant. v0..v5
		// NodeProps schemas declare key (not variant), so production data
		// never carries both, but the always-overwrite contract matches
		// the console's single source of truth.
		if v, ok := m["key"]; ok {
			m["variant"] = v
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

// codecMigration re-encodes stored schematics from msgpack to orc. It is
// pinned to the v6 shape so its output stays stable as Schematic evolves.
var codecMigration = gorp.CodecMigration[Key, schematicv0.Schematic]("msgpack_to_orc")

// liftMigration lifts stored schematics from the v6 blob layout to the typed v7
// shape.
var liftMigration = gorp.NewEntryMigration(
	"v55_lift_typed_schematic", migrateSchematic, codecMigration.Key(),
)

// Migrations is the ordered set of migrations introduced at this version.
var Migrations = []migrate.Migration{codecMigration, liftMigration}

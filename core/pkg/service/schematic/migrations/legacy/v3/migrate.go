// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v2"
)

// Migrate transforms v2 schematic data into v3 by attaching an empty segments
// slice to every edge. Mirrors the console's v2 -> v3 step. The opaque Data
// bag carried on every v0/v1/v2 edge is passed through unchanged so the
// v5 -> v6 lift step can recover ReactFlow's per-edge segments / color /
// variant from blobs that predate the v3 schema.
func Migrate(old v2.Data) Data {
	edges := make([]Edge, len(old.Edges))
	for i, e := range old.Edges {
		edges[i] = upgradeEdge(e)
	}
	return Data{
		Version:         Version,
		Editable:        old.Editable,
		FitViewOnResize: old.FitViewOnResize,
		Snapshot:        old.Snapshot,
		RemoteCreated:   old.RemoteCreated,
		Viewport:        old.Viewport,
		Nodes:           old.Nodes,
		Edges:           edges,
		Props:           old.Props,
		Control:         old.Control,
		Legend:          old.Legend,
		Key:             old.Key,
		Type:            old.Type,
		ViewportMode:    old.ViewportMode,
	}
}

func upgradeEdge(e v0.Edge) Edge {
	return Edge{
		Key:          e.Key,
		Source:       e.Source,
		Target:       e.Target,
		SourceHandle: e.SourceHandle,
		TargetHandle: e.TargetHandle,
		Segments:     []Segment{},
		Data:         e.Data,
	}
}

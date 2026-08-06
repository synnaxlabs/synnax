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
	"github.com/samber/lo"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v2"
)

// Migrate transforms v2 schematic data into v3 by attaching an empty segments slice to
// every edge. Mirrors the Console's v2 -> v3 step. The opaque Data bag carried on every
// v0/v1/v2 edge is passed through unchanged so the v5 -> v6 lift step can recover
// ReactFlow's per-edge segments / color / variant from blobs that predate the v3
// schema.
func Migrate(old v2.Data) Data {
	return Data{
		Version:  Version,
		Snapshot: old.Snapshot,
		Nodes:    old.Nodes,
		Edges: lo.Map(old.Edges, func(e v0.Edge, _ int) Edge {
			return Edge{
				Key:          e.Key,
				Source:       e.Source,
				Target:       e.Target,
				SourceHandle: e.SourceHandle,
				TargetHandle: e.TargetHandle,
				Segments:     []Segment{},
				Data:         e.Data,
			}
		}),
		Props: old.Props,
	}
}

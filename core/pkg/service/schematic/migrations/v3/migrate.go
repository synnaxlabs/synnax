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
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/v2"
)

// Migrate transforms v2 schematic data into v3 by adding empty segments to edges.
func Migrate(old v2.Data) (Data, error) {
	edges := make([]EdgeData, len(old.Edges))
	for i, e := range old.Edges {
		edges[i] = EdgeData{
			Key:      e.Key,
			Source:   e.Source,
			Target:   e.Target,
			Segments: []SegmentData{},
		}
	}
	return Data{
		Nodes:         old.Nodes,
		Edges:         edges,
		Props:         old.Props,
		Legend:        old.Legend,
		Snapshot:      old.Snapshot,
		RemoteCreated: old.RemoteCreated,
		Key:           old.Key,
		Type:          old.Type,
	}, nil
}

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
	"github.com/synnaxlabs/arc/ir"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v0"
)

// Migrate transforms v0 arc state into v1 by lifting the flat ReactFlow edge form
// into nested Handle objects. An absent handle field becomes an empty Param.
func Migrate(old v0.Data) Data {
	edges := make([]Edge, len(old.Graph.Edges))
	for i, e := range old.Graph.Edges {
		edges[i] = Edge{
			Key:    e.Key,
			Source: ir.Handle{Node: e.Source, Param: stringOrEmpty(e.SourceHandle)},
			Target: ir.Handle{Node: e.Target, Param: stringOrEmpty(e.TargetHandle)},
		}
	}
	return Data{
		Graph: Graph{
			Nodes: old.Graph.Nodes,
			Edges: edges,
			Props: old.Graph.Props,
		},
		Text: old.Text,
		Mode: old.Mode,
	}
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

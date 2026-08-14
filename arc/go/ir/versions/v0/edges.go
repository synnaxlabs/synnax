// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/samber/lo"

// FindByTarget searches for an edge with the given target handle.
func (e Edges) FindByTarget(handle Handle) (Edge, bool) {
	return lo.Find(e, func(edge Edge) bool { return edge.Target == handle })
}

// GetInputs returns all edges targeting the given node.
func (e Edges) GetInputs(nodeKey string) []Edge {
	return lo.Filter(
		e,
		func(edge Edge, _ int) bool { return edge.Target.Node == nodeKey },
	)
}

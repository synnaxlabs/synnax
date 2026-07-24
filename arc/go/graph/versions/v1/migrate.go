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
	"context"

	v0 "github.com/synnaxlabs/arc/graph/versions/v0"
	irv0 "github.com/synnaxlabs/arc/ir/versions/v0"
)

// MigrateGraph lifts a v1 graph into the current shape; arc's generated
// migrations consume it cross-package.
func MigrateGraph(ctx context.Context, old v0.Graph) (Graph, error) {
	return autoMigrateGraph(ctx, old)
}

// migrateEdge lifts a v1 IR edge into the current keyed graph edge, assigning a
// fresh key.
func migrateEdge(ctx context.Context, old irv0.Edge) (Edge, error) {
	return autoMigrateEdge(ctx, old)
}

// migrateNode lifts a v1 node into the current shape, dropping the config and type
// fields that moved to Graph.Inputs.
func migrateNode(ctx context.Context, old v0.Node) (Node, error) {
	return autoMigrateNode(ctx, old)
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"

	v1 "github.com/synnaxlabs/arc/graph/types/v1"
)

// MigrateGraph lifts a v1 graph into the current shape; arc's generated
// migrations consume it cross-package.
func MigrateGraph(ctx context.Context, old v1.Graph) (Graph, error) {
	return autoMigrateGraph(ctx, old)
}

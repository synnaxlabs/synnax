// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"

	"github.com/synnaxlabs/x/telem"
)

// Context carries runtime execution state and callbacks for node execution.
// It embeds context.Context for cancellation, deadlines, and values.
type Context struct {
	context.Context
	// Elapsed is the time elapsed since the runtime started.
	// Used by time-based nodes (interval, wait) to track timing.
	Elapsed telem.TimeSpan
	// MarkChanged signals that an output parameter has new data.
	// This triggers dependent nodes to execute in the next scheduler pass.
	MarkChanged func(output string)
	// ReportError reports a runtime error without stopping execution.
	// The node should continue where possible, using safe defaults.
	ReportError func(err error)
	// ActivateStage transitions to the stage that the given node belongs to.
	// Used by stage_entry nodes to trigger stage transitions.
	ActivateStage func(nodeKey string)
}

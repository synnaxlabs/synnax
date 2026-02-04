// Copyright 2026 Synnax Labs, Inc.
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

// RunReason indicates what triggered the current scheduler run.
type RunReason int

//go:generate stringer -type=RunReason
const (
	// ReasonTimerTick indicates the scheduler ran due to timer expiration.
	ReasonTimerTick RunReason = iota
	// ReasonChannelInput indicates the scheduler ran due to input data arrival.
	ReasonChannelInput
)

// Context carries runtime execution state and callbacks for node execution.
// It embeds context.Context for cancellation, deadlines, and values.
type Context struct {
	context.Context
	// MarkChanged signals that an output parameter has new data.
	// This triggers dependent nodes to execute in the next scheduler pass.
	MarkChanged func(output string)
	// ReportError reports a runtime error without stopping execution.
	// The node should continue where possible, using safe defaults.
	ReportError func(err error)
	// ActivateStage transitions to the stage associated with the currently
	// executing node. Used by stage_entry nodes to trigger stage transitions.
	// The scheduler uses the current node key to look up the target stage.
	ActivateStage func()
	// Elapsed is the time elapsed since the runtime started.
	// Used by time-based nodes (interval, wait) to track timing.
	Elapsed telem.TimeSpan
	// Tolerance is the timing tolerance for interval/wait comparisons.
	// Allows firing up to this amount early to handle OS scheduling jitter.
	Tolerance telem.TimeSpan
	// Reason indicates what triggered this scheduler run (timer tick or channel input).
	// Time-based nodes should only fire when Reason is ReasonTimerTick.
	Reason RunReason
}

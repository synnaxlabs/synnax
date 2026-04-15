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
	// MarkSelfChanged requests that this node be re-executed on the next
	// scheduler cycle without requiring upstream re-triggering. Used by
	// nodes like Wait that need multiple ticks to complete after activation.
	MarkSelfChanged func()
	// SetDeadline reports the absolute elapsed time at which this node next
	// needs a TimerTick. The scheduler tracks the minimum across all nodes
	// and uses it to wake the loop at the right time.
	SetDeadline func(telem.TimeSpan)
	// ReportError reports a runtime error without stopping execution.
	// The node should continue where possible, using safe defaults.
	ReportError func(err error)
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

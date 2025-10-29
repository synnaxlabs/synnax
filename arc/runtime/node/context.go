// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import "context"

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
}

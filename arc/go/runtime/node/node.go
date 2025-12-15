// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package node defines the core execution interface for arc runtime nodes.
//
// Nodes are the fundamental building blocks of arc programs. Each node represents
// a computational unit that transforms input data into output data. Nodes are
// organized into strata (layers) based on their dependencies and executed by the
// scheduler in topological order.
//
// The Node interface defines two lifecycle methods:
//   - Init: Called once when the scheduler initializes, only for stratum-0 nodes
//   - Next: Called each execution cycle when the node or its inputs change
//
// Factories create node instances from IR (intermediate representation) definitions.
// MultiFactory allows composition of multiple factories with fallback behavior.
package node

// Node executes computational operations within the arc runtime.
// Implementations transform input data into output data and signal changes
// to downstream nodes via the Context.MarkChanged callback.
type Node interface {
	// Init performs one-time initialization for source nodes.
	// Called only for stratum-0 nodes during scheduler initialization.
	Init(ctx Context)
	// Next executes the node's computational logic.
	// Called each cycle when the node is in stratum-0 or when marked as changed.
	Next(ctx Context)
	// Reset is called when a stage containing this node is activated.
	// It resets internal state (e.g., timers) and one-shot edge tracking.
	// Nodes that embed *state.Node get a default implementation that clears
	// one-shot state. Nodes with custom state should override and call the
	// embedded Reset() first.
	Reset()
}

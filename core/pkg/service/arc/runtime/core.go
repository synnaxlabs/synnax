// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// Core implements stratified reactive execution for Arc dataflow graphs.
// Stratification enables single-pass, glitch-free evaluation with predictable performance.
type Core struct {
	module     arc.Module
	nodes      map[string]stage.Node
	strata     map[string]int               // Pre-computed from analyzer
	maxStratum int                          // Maximum stratum level
	values     map[channel.Key]telem.Series // Channel value cache (ambient state)
}

// Flow starts all node goroutines and executes the initial evaluation.
// Constants emit during this phase, then we run a complete stratified pass.
func (c *Core) Flow(ctx signal.Context) {
	// Start all node goroutines (constants will emit immediately)
	for _, node := range c.nodes {
		node.Flow(ctx)
	}

	// Execute initial stratified pass (propagate constants)
	c.evaluateStrata(ctx)
}

// Next processes a new telemetry frame and executes stratified evaluation.
func (c *Core) Next(ctx context.Context, fr core.Frame) {
	// Update channel cache (ambient state)
	for key, series := range fr.Entries() {
		c.values[key] = series
	}

	// Execute complete stratified pass
	c.evaluateStrata(ctx)
}

// evaluateStrata executes all nodes in stratified order (single pass).
// Each node evaluates exactly once per frame in dependency order.
func (c *Core) evaluateStrata(ctx context.Context) {
	// Evaluate by stratum: 0, 1, 2, ..., maxStratum
	for stratum := 0; stratum <= c.maxStratum; stratum++ {
		for nodeKey, node := range c.nodes {
			if c.strata[nodeKey] == stratum {
				node.Next(ctx)
			}
		}
	}
}

// createOutputHandler creates the output handler for a node.
// When a node produces output, this handler propagates it to downstream nodes via Load().
func (c *Core) createOutputHandler(nodeKey string) stage.OutputHandler {
	return func(ctx context.Context, sourceParam string, val value.Value) {
		// Find all outgoing edges from this node's parameter
		for _, edge := range c.module.Edges {
			if edge.Source.Node == nodeKey && edge.Source.Param == sourceParam {
				// Load value into target node's parameter
				targetNode := c.nodes[edge.Target.Node]
				targetNode.Load(edge.Target.Param, val)
			}
		}
	}
}

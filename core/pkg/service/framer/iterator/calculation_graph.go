// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// dependencyGraph represents the dependency relationships between calculated channels.
type dependencyGraph struct {
	// nodes maps channel keys to their graph nodes
	nodes map[channel.Key]*dependencyNode
	// calculatedKeys is the set of all calculated channel keys in the graph
	calculatedKeys set.Set[channel.Key]
	// concreteBaseKeys is the set of all concrete (non-calculated) base channel keys
	concreteBaseKeys set.Set[channel.Key]
}

// dependencyNode represents a single calculated channel in the dependency graph.
type dependencyNode struct {
	key        channel.Key
	channel    channel.Channel
	calculator *calculation.Calculator
	// dependencies are the channel keys this node depends on
	dependencies []channel.Key
	// visited tracks if this node has been visited during topological sort
	visited bool
	// inStack tracks if this node is in the current DFS stack (for cycle detection)
	inStack bool
}

// buildDependencyGraph recursively builds a complete dependency graph for the given
// calculated channels and returns:
// - A topologically sorted list of calculators (dependencies first)
// - The set of all calculated channel keys (for exclusion from base iterator)
// - The set of concrete base channel keys (for base iterator to fetch)
func (s *Service) buildDependencyGraph(
	ctx context.Context,
	requestedChannels []channel.Channel,
) ([]*calculation.Calculator, set.Set[channel.Key], set.Set[channel.Key], error) {
	graph := &dependencyGraph{
		nodes:            make(map[channel.Key]*dependencyNode),
		calculatedKeys:   make(set.Set[channel.Key]),
		concreteBaseKeys: make(set.Set[channel.Key]),
	}

	// Process each requested channel that is calculated
	for _, ch := range requestedChannels {
		if ch.IsCalculated() && !ch.IsLegacyCalculated() {
			if err := s.addChannelToGraph(ctx, graph, ch); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	// Perform topological sort to order calculators
	sortedCalculators, err := graph.topologicalSort()
	if err != nil {
		return nil, nil, nil, err
	}

	return sortedCalculators, graph.calculatedKeys, graph.concreteBaseKeys, nil
}

// addChannelToGraph recursively adds a calculated channel and all its dependencies
// to the graph. If a dependency is also calculated, it recursively processes it.
func (s *Service) addChannelToGraph(
	ctx context.Context,
	graph *dependencyGraph,
	ch channel.Channel,
) error {
	// If we've already processed this channel, skip it
	if _, exists := graph.nodes[ch.Key()]; exists {
		return nil
	}

	// Open a calculator for this channel
	calculator, err := s.openCalculator(ctx, ch)
	if err != nil {
		return err
	}

	// Get the channel keys this calculator depends on
	dependencies := calculator.ReadFrom()

	// Create a node for this channel
	node := &dependencyNode{
		key:          ch.Key(),
		channel:      ch,
		calculator:   calculator,
		dependencies: dependencies,
	}
	graph.nodes[ch.Key()] = node
	graph.calculatedKeys.Add(ch.Key())

	// Fetch dependency channel metadata
	var dependencyChannels []channel.Channel
	if len(dependencies) > 0 {
		if err := s.cfg.Channel.NewRetrieve().
			Entries(&dependencyChannels).
			WhereKeys(dependencies...).
			Exec(ctx, nil); err != nil {
			return err
		}
	}

	// Process each dependency
	for _, depCh := range dependencyChannels {
		if depCh.IsCalculated() && !depCh.IsLegacyCalculated() {
			// Recursively process calculated dependencies
			if err = s.addChannelToGraph(ctx, graph, depCh); err != nil {
				return err
			}
		} else {
			// This is a concrete base channel (leaf node)
			graph.concreteBaseKeys.Add(depCh.Key())
		}
	}

	return nil
}

// topologicalSort performs a topological sort on the dependency graph using DFS.
// Returns calculators in dependency order (dependencies first) or an error if
// a circular dependency is detected.
func (g *dependencyGraph) topologicalSort() ([]*calculation.Calculator, error) {
	var sorted []*calculation.Calculator
	var stack []channel.Key

	// Visit each node
	for key := range g.nodes {
		if !g.nodes[key].visited {
			if err := g.dfsVisit(key, &sorted, &stack); err != nil {
				return nil, err
			}
		}
	}

	return sorted, nil
}

// dfsVisit performs a depth-first search visit for topological sorting.
// It also detects cycles by tracking nodes in the current stack.
func (g *dependencyGraph) dfsVisit(
	key channel.Key,
	sorted *[]*calculation.Calculator,
	stack *[]channel.Key,
) error {
	node := g.nodes[key]

	// Check for circular dependency
	if node.inStack {
		// Build cycle path for error message
		cycleStart := -1
		for i, k := range *stack {
			if k == key {
				cycleStart = i
				break
			}
		}
		cyclePath := append([]channel.Key{}, (*stack)[cycleStart:]...)
		cyclePath = append(cyclePath, key)
		return errors.Newf("circular dependency detected: %v", cyclePath)
	}

	// If already visited, no need to process again
	if node.visited {
		return nil
	}

	// Mark as in current DFS stack
	node.inStack = true
	*stack = append(*stack, key)

	// Visit all dependencies first (that are also calculated)
	for _, depKey := range node.dependencies {
		if _, exists := g.nodes[depKey]; exists {
			if err := g.dfsVisit(depKey, sorted, stack); err != nil {
				return err
			}
		}
	}

	// Mark as visited and remove from stack
	node.visited = true
	node.inStack = false
	*stack = (*stack)[:len(*stack)-1]

	// Add to sorted list (dependencies are already added, so this comes after)
	*sorted = append(*sorted, node.calculator)

	return nil
}

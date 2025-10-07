// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package archive

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Runtime is the stateful execution engine for Arc programs
type Runtime struct {
	// Graph is the compiled IR containing stages, nodes, edges, and strata
	graph ir.IR

	// Channel state for all channels in the graph
	channelState map[uint32]*ChannelState

	// Node state for all nodes in the graph
	nodeState map[string]*NodeState

	// Edge buffers for stratum-to-stratum data passing
	edgeBuffers map[ir.Edge][]any

	// WASM runtime
	wasmRuntime wazero.Runtime

	// Current frame number (for debugging/metadata)
	frameNumber int64

	// Context for WASM execution
	ctx context.Context
}

// ChannelState holds the runtime state for a single channel
type ChannelState struct {
	// Identity (immutable)
	ID      uint32
	Type    ir.Type
	IndexID string // For future alignment support

	// Data queue (mutable)
	SeriesQueue  []telem.Series
	LatestSample any // Typed value (float64, int32, etc.)

	// Metadata (mutable)
	HasData     bool
	LastUpdated int64 // Frame number when last updated

	// Dependency tracking (immutable after initialization)
	Subscribers []string // Node keys that read from this channel
	Writers     []string // Node keys that write to this channel
}

// NodeState holds the runtime state for a single node
type NodeState struct {
	Key               string
	ChannelWaterMarks map[uint32]telem.Alignment
	FirstActivated    bool            // Has first activation completed?
	RequiredInputs    set.Set[uint32] // Channel IDs needed for first activation
	EverReceived      set.Set[uint32] // Which channels have ever had data
	Instance          api.Module
	State             map[string]any
}

// Sample represents a single execution input with all parameter values
type Sample struct {
	Alignment telem.Alignment // Alignment for channel inputs
	Values    map[string]any  // paramName → value
}

// NewRuntime creates a new runtime from compiled IR
func NewRuntime(ctx context.Context, program ir.IR) (*Runtime, error) {
	r := &Runtime{
		graph:        program,
		channelState: make(map[uint32]*ChannelState),
		nodeState:    make(map[string]*NodeState),
		edgeBuffers:  make(map[ir.Edge][]any),
		wasmRuntime:  wazero.NewRuntime(ctx),
		frameNumber:  0,
		ctx:          ctx,
	}

	// Initialize channel state
	if err := r.initializeChannels(); err != nil {
		return nil, err
	}

	// Initialize node state
	if err := r.initializeNodes(); err != nil {
		return nil, err
	}

	// Discover subscribers and writers
	r.discoverDependencies()

	return r, nil
}

// initializeChannels creates ChannelState for all channels referenced in the graph
func (r *Runtime) initializeChannels() error {
	// Collect all channel IDs from nodes
	channelIDs := make(set.Set[uint32])
	for _, node := range r.graph.Nodes {
		for channelID := range node.Channels.Read {
			channelIDs.Add(channelID)
		}
		for channelID := range node.Channels.Write {
			channelIDs.Add(channelID)
		}
	}

	// Create state for each channel
	for channelID := range channelIDs {
		// TODO: Look up channel type from symbol table
		// For now, use a placeholder type
		r.channelState[channelID] = &ChannelState{
			ID:           channelID,
			Type:         ir.F64{}, // TODO: Get from symbol resolver
			SeriesQueue:  []telem.Series{},
			LatestSample: 0.0, // TODO: Zero value for type
			HasData:      false,
			LastUpdated:  0,
			Subscribers:  []string{},
			Writers:      []string{},
		}
	}

	return nil
}

// initializeNodes creates NodeState for all nodes in the graph
func (r *Runtime) initializeNodes() error {
	for _, node := range r.graph.Nodes {
		// Find the stage definition
		stage, ok := r.graph.GetStage(node.Type)
		if !ok {
			return errors.Newf("stage %s not found for node %s", node.Type, node.Key)
		}

		// Create node state
		nodeState := &NodeState{
			Key:               node.Key,
			ChannelWaterMarks: make(map[uint32]telem.Alignment),
			FirstActivated:    false,
			RequiredInputs:    make(set.Set[uint32]),
			EverReceived:      make(set.Set[uint32]),
			State:             make(map[string]any),
		}

		// Copy required inputs from node channels
		for channelID := range node.Channels.Read {
			nodeState.RequiredInputs.Add(channelID)
		}

		// Initialize water marks to zero for all input channels
		for channelID := range node.Channels.Read {
			nodeState.ChannelWaterMarks[channelID] = telem.Alignment(0)
		}

		// Initialize stateful variables to zero values
		for name, varType := range stage.StatefulVariables.Iter() {
			nodeState.State[name] = zeroValueForType(varType)
		}

		r.nodeState[node.Key] = nodeState
	}

	return nil
}

// discoverDependencies populates Subscribers and Writers lists for channels
func (r *Runtime) discoverDependencies() {
	// Discover subscribers (nodes that read from channels)
	for _, node := range r.graph.Nodes {
		for channelID := range node.Channels.Read {
			if state, ok := r.channelState[channelID]; ok {
				state.Subscribers = append(state.Subscribers, node.Key)
			}
		}
	}

	// Discover writers (nodes that write to channels)
	for _, node := range r.graph.Nodes {
		for channelID := range node.Channels.Write {
			if state, ok := r.channelState[channelID]; ok {
				state.Writers = append(state.Writers, node.Key)
			}
		}
	}
}

// Next processes a single frame through the Arc program
func (r *Runtime) Next(frame telem.Frame[uint32]) error {
	r.frameNumber++

	// Step 1: Ingest frame and identify affected nodes
	affectedNodes := r.ingestFrame(frame)

	// Step 2: Execute affected nodes in stratified order
	if err := r.executeStrata(affectedNodes); err != nil {
		return err
	}

	// Step 3: Garbage collect consumed series
	r.gcSeries()

	// Step 4: Clear edge buffers (don't persist across frames)
	r.clearEdgeBuffers()

	return nil
}

// ingestFrame updates channel state with new series and identifies affected nodes
func (r *Runtime) ingestFrame(frame telem.Frame[uint32]) set.Set[string] {
	affectedNodes := make(set.Set[string])

	// Iterate over all series in the frame using the Entries iterator
	for channelID, series := range frame.Entries() {
		state, ok := r.channelState[channelID]
		if !ok {
			// Unknown channel, skip
			continue
		}

		// Append series to queue
		state.SeriesQueue = append(state.SeriesQueue, series)

		// Update metadata
		state.HasData = true
		state.LastUpdated = r.frameNumber

		// Extract last sample for LatestSample
		if series.Len() > 0 {
			state.LatestSample = valueAtIndex(series, series.Len()-1)
		}

		// Mark all subscribers as affected
		for _, nodeKey := range state.Subscribers {
			affectedNodes.Add(nodeKey)
		}
	}

	return affectedNodes
}

// executeStrata processes affected nodes in stratified order
func (r *Runtime) executeStrata(affectedNodes set.Set[string]) error {
	// Execute each stratum in order
	for stratum := 0; stratum <= r.graph.Strata.Max; stratum++ {
		// Get all nodes at this stratum that are affected
		stratumNodes := []string{}
		for nodeKey := range affectedNodes {
			if r.graph.Strata.Get(nodeKey) == stratum {
				stratumNodes = append(stratumNodes, nodeKey)
			}
		}

		// Execute each node in this stratum
		for _, nodeKey := range stratumNodes {
			if err := r.executeNode(nodeKey); err != nil {
				return errors.Wrapf(err, "failed to execute node %s", nodeKey)
			}

			// After node executes, mark downstream nodes as affected
			r.markDownstreamAffected(nodeKey, affectedNodes)
		}
	}

	return nil
}

// markDownstreamAffected marks all nodes connected by edges as affected
func (r *Runtime) markDownstreamAffected(sourceNode string, affectedNodes set.Set[string]) {
	for _, edge := range r.graph.Edges {
		if edge.Source.Node == sourceNode {
			affectedNodes.Add(edge.Target.Node)
		}
	}
}

// executeNode processes all samples for a single node
func (r *Runtime) executeNode(nodeKey string) error {
	return r.executeNodeImpl(nodeKey)
}

// gcSeries removes series that all subscribers have consumed
func (r *Runtime) gcSeries() {
	for _, channelState := range r.channelState {
		if len(channelState.Subscribers) == 0 {
			// No subscribers - keep only latest series for observability
			if len(channelState.SeriesQueue) > 1 {
				channelState.SeriesQueue = channelState.SeriesQueue[len(channelState.SeriesQueue)-1:]
			}
			continue
		}

		// Find minimum water mark across all subscribers
		minWaterMark := telem.MaxAlignment
		for _, nodeKey := range channelState.Subscribers {
			nodeState := r.nodeState[nodeKey]
			waterMark := nodeState.ChannelWaterMarks[channelState.ID]
			if waterMark < minWaterMark {
				minWaterMark = waterMark
			}
		}

		// Keep only series with samples beyond minWaterMark
		newQueue := []telem.Series{}
		for _, series := range channelState.SeriesQueue {
			// Calculate the alignment of the last sample in the series
			lastAlignment := series.Alignment + telem.Alignment(series.Len()-1)
			if lastAlignment > minWaterMark {
				newQueue = append(newQueue, series)
			}
		}
		channelState.SeriesQueue = newQueue
	}
}

// clearEdgeBuffers removes all edge buffers (don't persist across frames)
func (r *Runtime) clearEdgeBuffers() {
	r.edgeBuffers = make(map[ir.Edge][]any)
}

// Close releases resources held by the runtime
func (r *Runtime) Close() error {
	return r.wasmRuntime.Close(r.ctx)
}

// Helper functions

// zeroValueForType returns the zero value for a given IR type
func zeroValueForType(t ir.Type) any {
	switch t.(type) {
	case ir.I8, ir.I16, ir.I32, ir.I64:
		return int64(0)
	case ir.U8, ir.U16, ir.U32, ir.U64:
		return uint64(0)
	case ir.F32:
		return float32(0)
	case ir.F64:
		return float64(0)
	case ir.String:
		return ""
	default:
		return nil
	}
}

// valueAtIndex extracts a typed value from a series at the given index
func valueAtIndex(series telem.Series, index int64) any {
	// Use the DataType to unmarshal the correct type
	switch series.DataType {
	case telem.Float64T:
		return telem.ValueAt[float64](series, int(index))
	case telem.Float32T:
		return telem.ValueAt[float32](series, int(index))
	case telem.Int64T:
		return telem.ValueAt[int64](series, int(index))
	case telem.Int32T:
		return telem.ValueAt[int32](series, int(index))
	case telem.Int16T:
		return telem.ValueAt[int16](series, int(index))
	case telem.Int8T:
		return telem.ValueAt[int8](series, int(index))
	case telem.Uint64T:
		return telem.ValueAt[uint64](series, int(index))
	case telem.Uint32T:
		return telem.ValueAt[uint32](series, int(index))
	case telem.Uint16T:
		return telem.ValueAt[uint16](series, int(index))
	case telem.Uint8T:
		return telem.ValueAt[uint8](series, int(index))
	case telem.TimeStampT:
		return telem.ValueAt[telem.TimeStamp](series, int(index))
	default:
		// Unknown type, return nil
		return nil
	}
}

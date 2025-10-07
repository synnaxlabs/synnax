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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// executeNode processes all samples for a single node
func (r *Runtime) executeNodeImpl(nodeKey string) error {
	node := lo.Must(lo.Find(r.graph.Nodes, func(n ir.Node) bool {
		return n.Key == nodeKey
	}))
	nodeState := r.nodeState[nodeKey]
	stage := lo.Must(r.graph.GetStage(node.Type))

	// Step 1: Check if node can activate (first-activation check)
	if !r.canActivate(nodeKey) {
		return nil // Not ready yet
	}

	// Step 2: Collect samples from all inputs
	samples, err := r.collectSamples(node, nodeState, stage)
	if err != nil {
		return err
	}

	// If no samples, nothing to do
	if len(samples) == 0 {
		return nil
	}

	// Step 3: Execute WASM for each sample
	outputSamples := []map[string]any{}
	for _, sample := range samples {
		output, err := r.executeWASM(node, nodeState, stage, sample)
		if err != nil {
			return errors.Wrapf(err, "WASM execution failed")
		}
		outputSamples = append(outputSamples, output)
	}

	// Step 4: Write outputs to edge buffers and channels
	return r.writeOutputs(node, stage, outputSamples)
}

// canActivate checks if a node is ready to activate
func (r *Runtime) canActivate(nodeKey string) bool {
	nodeState := r.nodeState[nodeKey]

	// If already activated, always allow
	if nodeState.FirstActivated {
		return true
	}

	// For first activation, need all required inputs to have received data
	for channelID := range nodeState.RequiredInputs {
		if !nodeState.EverReceived.Contains(channelID) {
			channelState := r.channelState[channelID]
			if !channelState.HasData {
				return false // Still waiting for this channel
			}
			// Mark as received
			nodeState.EverReceived.Add(channelID)
		}
	}

	// All required inputs have data, mark as activated
	nodeState.FirstActivated = true
	return true
}

// collectSamples gathers all input samples for a node execution
func (r *Runtime) collectSamples(
	node ir.Node,
	nodeState *NodeState,
	stage ir.Stage,
) ([]Sample, error) {
	// Build map of param name → samples
	inputStreams := make(map[string][]any)
	maxLength := 0

	// Step 1: Collect from CHANNEL inputs
	for channelID := range node.Channels.Read {
		// Find which parameter this channel maps to
		paramName := r.findParamForChannel(stage, channelID)
		if paramName == "" {
			continue // Channel not used in this stage (shouldn't happen)
		}

		samples := r.collectSamplesFromChannel(channelID, nodeState)
		inputStreams[paramName] = samples
		if len(samples) > maxLength {
			maxLength = len(samples)
		}
	}

	// Step 2: Collect from EDGE inputs
	for _, edge := range r.graph.Edges {
		if edge.Target.Node == node.Key {
			// This edge feeds into this node
			if values, ok := r.edgeBuffers[edge]; ok {
				inputStreams[edge.Target.Param] = values
				if len(values) > maxLength {
					maxLength = len(values)
				}
			}
		}
	}

	// If no inputs collected, return empty (shouldn't happen for activated nodes)
	if maxLength == 0 {
		return []Sample{}, nil
	}

	// Step 3: Align to max length (repeat LatestSample for short inputs)
	samples := make([]Sample, maxLength)
	for i := 0; i < maxLength; i++ {
		sampleValues := make(map[string]any)

		for paramName, paramSamples := range inputStreams {
			if i < len(paramSamples) {
				// Use actual sample
				sampleValues[paramName] = paramSamples[i]
			} else {
				// Use LatestSample (repeat last value)
				if len(paramSamples) > 0 {
					sampleValues[paramName] = paramSamples[len(paramSamples)-1]
				} else {
					// No samples at all - use zero value
					// Find parameter type from stage definition
					paramType, ok := stage.Params.Get(paramName)
					if ok {
						sampleValues[paramName] = zeroValueForType(paramType)
					}
				}
			}
		}

		samples[i] = Sample{
			Values: sampleValues,
		}
	}

	return samples, nil
}

// collectSamplesFromChannel extracts unconsumed samples from a channel
func (r *Runtime) collectSamplesFromChannel(
	channelID uint32,
	nodeState *NodeState,
) []any {
	channelState := r.channelState[channelID]
	waterMark := nodeState.ChannelWaterMarks[channelID]

	samples := []any{}
	lastAlignment := waterMark

	// Process all series in the queue
	for _, series := range channelState.SeriesQueue {
		// Process each sample in the series
		for i := int64(0); i < series.Len(); i++ {
			alignment := series.Alignment + telem.Alignment(i)
			if alignment > waterMark {
				value := valueAtIndex(series, i)
				samples = append(samples, value)
				lastAlignment = alignment
			}
		}
	}

	// Update water mark to last consumed alignment
	if len(samples) > 0 {
		nodeState.ChannelWaterMarks[channelID] = lastAlignment
	}

	return samples
}

// findParamForChannel finds which parameter name corresponds to a channel ID
func (r *Runtime) findParamForChannel(stage ir.Stage, channelID uint32) string {
	// For now, we need to infer this from the graph structure
	// In the future, we should store explicit channel→param mappings in the IR
	// For MVP, assume channel IDs map directly to parameter names (this is a simplification)
	// TODO: Fix this with proper channel→param mapping
	return "input" // Placeholder
}

// executeWASM invokes the WASM module for a single sample
func (r *Runtime) executeWASM(
	node ir.Node,
	nodeState *NodeState,
	stage ir.Stage,
	sample Sample,
) (map[string]any, error) {
	// TODO: Implement WASM execution
	// For now, return empty outputs
	outputs := make(map[string]any)

	// If stage has outputs, create placeholder values
	for name, outputType := range stage.Outputs.Iter() {
		outputs[name] = zeroValueForType(outputType)
	}

	return outputs, nil
}

// writeOutputs distributes node outputs to edge buffers and channels
func (r *Runtime) writeOutputs(
	node ir.Node,
	stage ir.Stage,
	outputSamples []map[string]any,
) error {
	// Group outputs by destination (edge vs channel)

	// Step 1: Write to edge buffers (node-to-node connections)
	for _, edge := range r.graph.Edges {
		if edge.Source.Node == node.Key {
			// This edge originates from this node
			// Get or create edge buffer
			if r.edgeBuffers[edge] == nil {
				r.edgeBuffers[edge] = []any{}
			}

			// Extract values for this output from all samples
			for _, sample := range outputSamples {
				if value, ok := sample[edge.Source.Param]; ok {
					r.edgeBuffers[edge] = append(r.edgeBuffers[edge], value)
				}
			}
		}
	}

	// Step 2: Write to output channels
	for channelID := range node.Channels.Write {
		// Find which output name this channel corresponds to
		// TODO: Proper output→channel mapping
		// For now, assume single output named "output" or first output
		outputName := ""
		if stage.Outputs.Count() == 1 {
			outputName, _ = stage.Outputs.At(0)
		} else {
			// Multiple outputs - need proper mapping
			// For now, skip
			continue
		}

		// Collect values for this output
		values := []any{}
		for _, sample := range outputSamples {
			if value, ok := sample[outputName]; ok {
				values = append(values, value)
			}
		}

		if len(values) == 0 {
			continue
		}

		// Create a series and write to channel
		// TODO: Proper series creation with correct type, time range, alignment
		series := r.createOutputSeries(channelID, values)
		r.writeToChannel(channelID, series)
	}

	return nil
}

// createOutputSeries creates a series from output values
func (r *Runtime) createOutputSeries(channelID uint32, values []any) telem.Series {
	// TODO: Implement proper series creation
	// For now, return a placeholder
	channelState := r.channelState[channelID]

	// Calculate alignment (continue from last alignment)
	lastAlignment := telem.Alignment(0)
	if len(channelState.SeriesQueue) > 0 {
		lastSeries := channelState.SeriesQueue[len(channelState.SeriesQueue)-1]
		lastAlignment = lastSeries.Alignment + telem.Alignment(lastSeries.Len())
	}

	return telem.Series{
		Alignment: lastAlignment,
		// TODO: Fill in Data, DataType, TimeRange
	}
}

// writeToChannel appends a series to a channel's queue
func (r *Runtime) writeToChannel(channelID uint32, series telem.Series) {
	channelState := r.channelState[channelID]

	// Append to queue
	channelState.SeriesQueue = append(channelState.SeriesQueue, series)

	// Update LatestSample
	if series.Len() > 0 {
		channelState.LatestSample = valueAtIndex(series, series.Len()-1)
	}

	// Update metadata
	channelState.HasData = true
	channelState.LastUpdated = r.frameNumber
}

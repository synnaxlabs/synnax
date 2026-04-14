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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type value struct {
	data telem.Series
	time telem.Series
}

// ProgramState manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type ProgramState struct {
	ir      ir.IR
	outputs map[ir.Handle]*value
}

// New creates a state manager from the given configuration.
// It initializes output storage for all node outputs and maps channel keys
// to their indexes.
func New(inter ir.IR) *ProgramState {
	s := &ProgramState{
		ir:      inter,
		outputs: make(map[ir.Handle]*value),
	}
	for _, node := range inter.Nodes {
		for _, p := range node.Outputs {
			s.outputs[ir.Handle{Node: node.Key, Param: p.Name}] = &value{
				data: telem.Series{DataType: types.ToTelem(p.Type)},
				time: telem.Series{DataType: telem.TimeStampT},
			}
		}
	}
	return s
}

// Node creates a node-specific state accessor for the given node key.
// It initializes alignment buffers and watermark tracking for the node's
// inputs.
func (s *ProgramState) Node(key string) *State {
	var (
		n            = s.ir.Nodes.Get(key)
		inputs       = make([]ir.Edge, len(n.Inputs))
		alignedData  = make([]telem.Series, len(n.Inputs))
		alignedTime  = make([]telem.Series, len(alignedData))
		accumulated  = make([]inputEntry, len(n.Inputs))
		inputSources = make([]*value, len(n.Inputs))
	)
	for i := range alignedData {
		alignedTime[i] = telem.Series{DataType: telem.TimeStampT}
	}
	for i, p := range n.Inputs {
		edge, found := s.ir.Edges.FindByTarget(
			ir.Handle{Node: key, Param: p.Name},
		)
		if found {
			inputs[i] = edge
			alignedData[i] = telem.Series{
				DataType: s.outputs[edge.Source].data.DataType,
			}
			inputSources[i] = s.outputs[edge.Source]
		} else {
			syntheticSource := ir.Handle{
				Node:  "__default_" + key + "_" + p.Name,
				Param: ir.DefaultOutputParam,
			}
			inputs[i] = ir.Edge{
				Source: syntheticSource,
				Target: ir.Handle{Node: key, Param: p.Name},
			}
			data := telem.NewSeriesFromAny(p.Value, types.ToTelem(p.Type))
			time := telem.NewSeriesV[telem.TimeStamp](0)
			alignedData[i] = data
			alignedTime[i] = time
			accumulated[i] = inputEntry{
				data:          data,
				time:          time,
				lastTimestamp: 0,
				consumed:      false,
			}
			if _, exists := s.outputs[syntheticSource]; !exists {
				s.outputs[syntheticSource] = &value{data: data, time: time}
			}
			inputSources[i] = s.outputs[syntheticSource]
		}
	}

	outputCache := make([]*value, len(n.Outputs))
	for i, p := range n.Outputs {
		handle := ir.Handle{Node: key, Param: p.Name}
		outputCache[i] = s.outputs[handle]
	}

	nd := &State{}
	nd.ir.inputs = inputs
	nd.ir.outputs = lo.Map(n.Outputs, func(item types.Param, _ int) ir.Handle {
		return ir.Handle{Node: key, Param: item.Name}
	})
	nd.aligned.data = alignedData
	nd.aligned.time = alignedTime
	nd.nodeOutputs = s.outputs
	nd.accumulated = accumulated
	nd.inputSources = inputSources
	nd.outputCache = outputCache
	return nd
}

type inputEntry struct {
	data          telem.Series
	time          telem.Series
	lastTimestamp telem.TimeStamp
	consumed      bool
}

// Node provides node-specific access to state, handling input alignment and
// output storage.
type State struct {
	ir struct {
		inputs  []ir.Edge
		outputs []ir.Handle
	}
	accumulated []inputEntry
	aligned     struct {
		data []telem.Series
		time []telem.Series
	}
	nodeOutputs  map[ir.Handle]*value
	inputSources []*value
	outputCache  []*value
}

// Reset is called by the scheduler when the stage containing this node is
// activated.
func (n *State) Reset() {}

// RefreshInputs performs temporal alignment of node inputs and returns whether
// the node should execute.
func (n *State) RefreshInputs() (recalculate bool) {
	if len(n.ir.inputs) == 0 {
		return true
	}
	hasUnconsumed := false
	for i := range n.ir.inputs {
		src := n.inputSources[i]
		if src != nil && src.time.Len() > 0 {
			ts := telem.ValueAt[telem.TimeStamp](src.time, -1)
			if ts > n.accumulated[i].lastTimestamp {
				n.accumulated[i] = inputEntry{
					data:          src.data,
					time:          src.time,
					lastTimestamp: ts,
					consumed:      false,
				}
			}
		}
		if n.accumulated[i].data.Len() == 0 {
			return false
		}
		if !n.accumulated[i].consumed {
			hasUnconsumed = true
		}
	}
	if !hasUnconsumed {
		return false
	}
	for i := range n.ir.inputs {
		n.aligned.data[i] = n.accumulated[i].data
		n.aligned.time[i] = n.accumulated[i].time
		n.accumulated[i].consumed = true
	}
	return true
}

// InputTime returns the timestamp series for the input at the given parameter
// index.
func (n *State) InputTime(paramIndex int) telem.Series {
	return n.aligned.time[paramIndex]
}

// InitInput initializes an input's source output with dummy values.
func (n *State) InitInput(paramIndex int, data, time telem.Series) {
	if paramIndex >= 0 && paramIndex < len(n.ir.inputs) {
		sourceHandle := n.ir.inputs[paramIndex].Source
		if v, ok := n.nodeOutputs[sourceHandle]; ok {
			v.data = data
			v.time = time
		}
	}
}

// Input returns the data series for the input at the given parameter index.
func (n *State) Input(paramIndex int) telem.Series {
	return n.aligned.data[paramIndex]
}

// Output returns a mutable pointer to the data series for the output at the
// given parameter index.
func (n *State) Output(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].data
}

// OutputTime returns a mutable pointer to the timestamp series for the output
// at the given parameter index.
func (n *State) OutputTime(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].time
}

// IsOutputTruthy checks if the output at the given param name is truthy.
func (n *State) IsOutputTruthy(paramName string) bool {
	for i, h := range n.ir.outputs {
		if h.Param == paramName {
			series := &n.outputCache[i].data
			return isSeriesTruthy(*series)
		}
	}
	return false
}

func isSeriesTruthy(s telem.Series) bool {
	if s.Len() == 0 {
		return false
	}
	dt := s.DataType
	switch dt {
	case telem.Float64T:
		return telem.ValueAt[float64](s, -1) != 0
	case telem.Float32T:
		return telem.ValueAt[float32](s, -1) != 0
	case telem.Int64T:
		return telem.ValueAt[int64](s, -1) != 0
	case telem.Int32T:
		return telem.ValueAt[int32](s, -1) != 0
	case telem.Int16T:
		return telem.ValueAt[int16](s, -1) != 0
	case telem.Int8T:
		return telem.ValueAt[int8](s, -1) != 0
	case telem.Uint64T:
		return telem.ValueAt[uint64](s, -1) != 0
	case telem.Uint32T:
		return telem.ValueAt[uint32](s, -1) != 0
	case telem.Uint16T:
		return telem.ValueAt[uint16](s, -1) != 0
	case telem.Uint8T:
		return telem.ValueAt[uint8](s, -1) != 0
	case telem.TimeStampT:
		return telem.ValueAt[telem.TimeStamp](s, -1) != 0
	default:
		return false
	}
}

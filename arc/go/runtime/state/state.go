// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package state manages runtime data flow and temporal alignment for arc programs.
//
// The state package provides the data infrastructure for reactive node execution:
//   - Node input/output data storage with temporal metadata
//   - Channel read/write buffering for external I/O
//   - Temporal alignment of inputs across multiple sources
//   - Watermark-based data consumption tracking
//
// Temporal alignment ensures that nodes process time-aligned data from multiple
// inputs. The RefreshInputs algorithm selects the input with the earliest new
// timestamp as the "trigger" and aligns other inputs to that temporal point.
package state

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	channelstate "github.com/synnaxlabs/arc/stl/channel/state"
	controlstate "github.com/synnaxlabs/arc/stl/control/state"
	seriesstate "github.com/synnaxlabs/arc/stl/series/state"
	stringsstate "github.com/synnaxlabs/arc/stl/strings/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type value struct {
	data telem.Series
	time telem.Series
}

// Config provides dependencies for creating a State instance.
type Config struct {
	IR             ir.IR
	ChannelDigests []channelstate.Digest
}

// State manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type State struct {
	Channel *channelstate.State
	Series  *seriesstate.State
	Strings *stringsstate.State
	Auth    *controlstate.State
	outputs map[ir.Handle]*value
	cfg     Config
}

// New creates a state manager from the given configuration.
// It initializes output storage for all node outputs and maps channel keys
// to their indexes.
func New(cfg Config) *State {
	s := &State{
		cfg:     cfg,
		outputs: make(map[ir.Handle]*value),
		Channel: channelstate.New(cfg.ChannelDigests),
		Series:  seriesstate.New(),
		Strings: stringsstate.New(),
		Auth:    &controlstate.State{},
	}
	for _, node := range cfg.IR.Nodes {
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
func (s *State) Node(key string) *Node {
	var (
		n            = s.cfg.IR.Nodes.Get(key)
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
		edge, found := s.cfg.IR.Edges.FindByTarget(
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

	return &Node{
		inputs: inputs,
		outputs: lo.Map(n.Outputs, func(item types.Param, _ int) ir.Handle {
			return ir.Handle{Node: key, Param: item.Name}
		}),
		channel:      s.Channel,
		nodeOutputs:  s.outputs,
		accumulated:  accumulated,
		alignedData:  alignedData,
		alignedTime:  alignedTime,
		inputSources: inputSources,
		outputCache:  outputCache,
	}
}

type inputEntry struct {
	data          telem.Series
	time          telem.Series
	lastTimestamp telem.TimeStamp
	consumed      bool
}

// Node provides node-specific access to state, handling input alignment and
// output storage.
type Node struct {
	inputs       []ir.Edge
	outputs      []ir.Handle
	channel      *channelstate.State
	nodeOutputs  map[ir.Handle]*value
	accumulated  []inputEntry
	alignedData  []telem.Series
	alignedTime  []telem.Series
	inputSources []*value
	outputCache  []*value
}

// Reset is called by the scheduler when the stage containing this node is
// activated.
func (n *Node) Reset() {}

// RefreshInputs performs temporal alignment of node inputs and returns whether
// the node should execute.
func (n *Node) RefreshInputs() (recalculate bool) {
	if len(n.inputs) == 0 {
		return true
	}
	hasUnconsumed := false
	for i := range n.inputs {
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
	for i := range n.inputs {
		n.alignedData[i] = n.accumulated[i].data
		n.alignedTime[i] = n.accumulated[i].time
		n.accumulated[i].consumed = true
	}
	return true
}

// InputTime returns the timestamp series for the input at the given parameter
// index.
func (n *Node) InputTime(paramIndex int) telem.Series {
	return n.alignedTime[paramIndex]
}

// InitInput initializes an input's source output with dummy values.
func (n *Node) InitInput(paramIndex int, data, time telem.Series) {
	if paramIndex >= 0 && paramIndex < len(n.inputs) {
		sourceHandle := n.inputs[paramIndex].Source
		if v, ok := n.nodeOutputs[sourceHandle]; ok {
			v.data = data
			v.time = time
		}
	}
}

// Input returns the data series for the input at the given parameter index.
func (n *Node) Input(paramIndex int) telem.Series {
	return n.alignedData[paramIndex]
}

// Output returns a mutable pointer to the data series for the output at the
// given parameter index.
func (n *Node) Output(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].data
}

// OutputTime returns a mutable pointer to the timestamp series for the output
// at the given parameter index.
func (n *Node) OutputTime(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].time
}

// ReadSeries reads buffered data and time series from a channel.
func (n *Node) ReadSeries(
	key uint32,
) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	return n.channel.ReadSeries(key)
}

// WriteSeries buffers data and time series for writing to a channel.
func (n *Node) WriteSeries(key uint32, value, time telem.Series) {
	n.channel.WriteSeries(key, value, time)
}

// IsOutputTruthy checks if the output at the given param name is truthy.
func (n *Node) IsOutputTruthy(paramName string) bool {
	for i, h := range n.outputs {
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

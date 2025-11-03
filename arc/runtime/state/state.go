// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type value struct {
	data telem.Series
	time telem.Series
}

// State manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type State struct {
	cfg     Config
	outputs map[ir.Handle]*value
	indexes map[uint32]uint32
	channel struct {
		reads  map[uint32]telem.MultiSeries
		writes map[uint32]telem.Series
	}
}

// ChannelDigest provides metadata about a channel for state initialization.
type ChannelDigest struct {
	Key      uint32
	DataType telem.DataType
	Index    uint32
}

// Config provides dependencies for creating a State instance.
type Config struct {
	ChannelDigests []ChannelDigest
	Edges          ir.Edges
	Nodes          ir.Nodes
}

// New creates a state manager from the given configuration.
// It initializes output storage for all node outputs and maps channel keys to their indexes.
func New(cfg Config) *State {
	s := &State{
		cfg:     cfg,
		outputs: make(map[ir.Handle]*value),
		indexes: make(map[uint32]uint32),
	}
	s.channel.reads = make(map[uint32]telem.MultiSeries)
	s.channel.writes = make(map[uint32]telem.Series)
	for _, d := range cfg.ChannelDigests {
		s.indexes[d.Key] = d.Index
	}
	for _, node := range cfg.Nodes {
		for p, ot := range node.Outputs.Iter() {
			s.outputs[ir.Handle{Node: node.Key, Param: p}] = &value{
				data: telem.Series{DataType: types.ToTelem(ot)},
				time: telem.Series{DataType: telem.TimeStampT},
			}
		}
	}
	return s
}

// Ingest adds external channel data to the read buffer.
// This is called when new data arrives from external sources (e.g., Synnax channels).
func (s *State) Ingest(fr telem.Frame[uint32]) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		s.channel.reads[key] = s.channel.reads[key].Append(fr.RawSeriesAt(rawI))
	}
}

// FlushWrites extracts buffered channel writes into a frame and clears the write buffer.
// Returns the updated frame and true if any writes were flushed, or the original frame and false otherwise.
func (s *State) FlushWrites(fr telem.Frame[uint32]) (telem.Frame[uint32], bool) {
	if len(s.channel.writes) == 0 {
		return fr, false
	}
	for key, data := range s.channel.writes {
		fr = fr.Append(key, data.DeepCopy())
	}
	clear(s.channel.writes)
	return fr, true
}

func (s *State) readChannel(key uint32) (telem.MultiSeries, bool) {
	series, ok := s.channel.reads[key]
	return series, ok
}

func (s *State) writeChannel(key uint32, data, time telem.Series) {
	s.channel.writes[key] = data
	idx := s.indexes[key]
	if idx != 0 {
		s.channel.writes[idx] = time
	}
}

func (s *State) ClearReads() {
	for key, ser := range s.channel.reads {
		ser.Series = ser.Series[:0]
		s.channel.reads[key] = ser
	}
}

// Node creates a node-specific state accessor for the given node key.
// It initializes alignment buffers and watermark tracking for the node's inputs.
func (s *State) Node(key string) *Node {
	n := s.cfg.Nodes.Get(key)
	inputs := lo.FilterMap(n.Inputs.Keys, func(item string, _ int) (ir.Edge, bool) {
		return s.cfg.Edges.FindByTarget(ir.Handle{Node: key, Param: item})
	})
	alignedData := make([]telem.Series, len(inputs))
	for i, input := range inputs {
		alignedData[i] = telem.Series{DataType: s.outputs[input.Source].data.DataType}
	}
	alignedTime := make([]telem.Series, len(alignedData))
	for i := range alignedData {
		alignedTime[i] = telem.Series{DataType: telem.TimeStampT}
	}
	return &Node{
		inputs: inputs,
		outputs: lo.Map(n.Outputs.Keys, func(item string, _ int) ir.Handle {
			return ir.Handle{Node: key, Param: item}
		}),
		state:       s,
		accumulated: make([]inputEntry, len(inputs)),
		alignedData: alignedData,
		alignedTime: make([]telem.Series, len(inputs)),
	}
}

type inputEntry struct {
	data      telem.MultiSeries
	time      telem.MultiSeries
	watermark telem.TimeStamp
}

// Node provides node-specific access to state, handling input alignment and output storage.
type Node struct {
	inputs      []ir.Edge
	outputs     []ir.Handle
	state       *State
	accumulated []inputEntry
	alignedData []telem.Series
	alignedTime []telem.Series
}

// RefreshInputs performs temporal alignment of node inputs and returns whether the node should execute.
//
// The algorithm:
//  1. Accumulates new data from source outputs that exceed the current watermark
//  2. Selects the input with the earliest new timestamp as the "trigger"
//  3. Aligns all inputs to the trigger's temporal point
//  4. Updates watermarks: trigger input is consumed, other inputs are reused
//  5. Prunes consumed data from accumulated buffers
//
// Returns true if the node has aligned inputs ready for execution, false otherwise.
func (n *Node) RefreshInputs() (recalculate bool) {
	// If node has no inputs, always allow execution
	if len(n.inputs) == 0 {
		return true
	}

	for i, edge := range n.inputs {
		sourceOutput, exists := n.state.outputs[edge.Source]
		if !exists || sourceOutput.data.Len() == 0 || sourceOutput.time.Len() == 0 {
			continue
		}
		lastTimestamp := telem.ValueAt[telem.TimeStamp](sourceOutput.time, -1)
		if lastTimestamp <= n.accumulated[i].watermark {
			continue
		}
		n.accumulated[i].data.Series = append(n.accumulated[i].data.Series, sourceOutput.data)
		n.accumulated[i].time.Series = append(n.accumulated[i].time.Series, sourceOutput.time)
	}
	for i := range n.inputs {
		if len(n.accumulated[i].data.Series) == 0 {
			return false
		}
	}
	var (
		triggerInputIdx  = -1
		triggerTimestamp telem.TimeStamp
		triggerSeriesIdx int
	)
	for i := range n.inputs {
		for j, timeSeries := range n.accumulated[i].time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > n.accumulated[i].watermark {
				if triggerInputIdx == -1 || ts < triggerTimestamp {
					triggerInputIdx = i
					triggerTimestamp = ts
					triggerSeriesIdx = j
				}
			}
		}
	}
	if triggerInputIdx == -1 {
		return false
	}
	for i := range n.inputs {
		if i == triggerInputIdx {
			n.alignedData[i] = n.accumulated[i].data.Series[triggerSeriesIdx]
			n.alignedTime[i] = n.accumulated[i].time.Series[triggerSeriesIdx]
			// For trigger input, set watermark to its aligned data's last timestamp
			if n.alignedTime[i].Len() > 0 {
				n.accumulated[i].watermark = telem.ValueAt[telem.TimeStamp](n.alignedTime[i], -1)
			} else {
				n.accumulated[i].watermark = triggerTimestamp
			}
		} else {
			latestIdx := len(n.accumulated[i].data.Series) - 1
			n.alignedData[i] = n.accumulated[i].data.Series[latestIdx]
			n.alignedTime[i] = n.accumulated[i].time.Series[latestIdx]
			// For catch-up inputs, set watermark to trigger timestamp (they're reused, not consumed)
			n.accumulated[i].watermark = triggerTimestamp
		}
	}
	for i := range n.inputs {
		var (
			newData []telem.Series
			newTime []telem.Series
		)
		for j, timeSeries := range n.accumulated[i].time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > n.accumulated[i].watermark {
				newData = append(newData, n.accumulated[i].data.Series[j])
				newTime = append(newTime, timeSeries)
			}
		}
		if len(newData) == 0 && len(n.accumulated[i].data.Series) > 0 {
			lastIdx := len(n.accumulated[i].data.Series) - 1
			newData = []telem.Series{n.accumulated[i].data.Series[lastIdx]}
			newTime = []telem.Series{n.accumulated[i].time.Series[lastIdx]}
		}
		n.accumulated[i].data.Series = newData
		n.accumulated[i].time.Series = newTime
	}
	return true
}

func (n *Node) output(paramIndex int) *value {
	handle := n.outputs[paramIndex]
	v, ok := n.state.outputs[handle]
	if !ok {
		v = &value{}
		n.state.outputs[handle] = v
	}
	return v
}

// InputTime returns the timestamp series for the input at the given parameter index.
// This is the aligned temporal metadata corresponding to Input(paramIndex).
func (n *Node) InputTime(paramIndex int) telem.Series {
	return n.alignedTime[paramIndex]
}

// Input returns the data series for the input at the given parameter index.
// This is the aligned data prepared by RefreshInputs.
func (n *Node) Input(paramIndex int) telem.Series {
	return n.alignedData[paramIndex]
}

// InitInput initializes an input's source output with dummy values.
// This is used for optional inputs to prevent alignment blocking when no real data has arrived yet.
// The timestamp should be > 0 to ensure it's above the initial watermark.
func (n *Node) InitInput(paramIndex int, data, time telem.Series) {
	if paramIndex >= 0 && paramIndex < len(n.inputs) {
		sourceHandle := n.inputs[paramIndex].Source
		if v, ok := n.state.outputs[sourceHandle]; ok {
			v.data = data
			v.time = time
		}
	}
}

// Output returns a mutable pointer to the data series for the output at the given parameter index.
// Nodes write their computed results to this series.
func (n *Node) Output(paramIndex int) *telem.Series {
	d := n.output(paramIndex)
	return &d.data
}

// OutputTime returns a mutable pointer to the timestamp series for the output at the given parameter index.
// Nodes write temporal metadata for their outputs to this series.
func (n *Node) OutputTime(paramIndex int) *telem.Series {
	d := n.output(paramIndex)
	return &d.time
}

// ReadChan reads buffered data and time series from a channel.
// If the channel has an index, both data and time are returned.
// Returns ok=false if the channel has no buffered data.
func (n *Node) ReadChan(key uint32) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	data, ok = n.state.readChannel(key)
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	indexKey := n.state.indexes[key]
	if indexKey == 0 {
		return data, telem.MultiSeries{}, true
	}
	time, ok = n.state.readChannel(indexKey)
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	return data, time, len(time.Series) > 0 && len(data.Series) > 0
}

// WriteChan buffers data and time series for writing to a channel.
// If the channel has an index, the time series is automatically written to the index channel.
func (n *Node) WriteChan(key uint32, value, time telem.Series) {
	n.state.writeChannel(key, value, time)
}

// ReadChannelValue reads a single value from a channel (for WASM runtime bindings).
func (s *State) ReadChannelValue(key uint32) (telem.Series, bool) {
	series, ok := s.channel.reads[key]
	if !ok {
		return telem.Series{}, false
	}
	return series.Series[series.Len()-1], ok
}

// WriteChannelValue writes a single value to a channel (for WASM runtime bindings).
// For channels with an index, you should also write the timestamp.
func (s *State) WriteChannelValue(key uint32, value telem.Series) {
	s.channel.writes[key] = value
}

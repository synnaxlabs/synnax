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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

type value struct {
	data telem.Series
	time telem.Series
}

// AuthorityChange represents a buffered authority change request.
// It is produced by set_authority nodes during reactive execution and consumed
// by the runtime task when flushing state.
type AuthorityChange struct {
	// Channel is the specific channel to change authority for.
	// If nil, the change applies to all write channels.
	Channel *uint32
	// Authority is the new authority value (0-255).
	Authority uint8
}

// State manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type State struct {
	channel struct {
		reads  map[uint32]telem.MultiSeries
		writes map[uint32]telem.Series
	}
	outputs          map[ir.Handle]*value
	indexes          map[uint32]uint32
	cfg              Config
	authorityChanges []AuthorityChange

	series              map[uint32]telem.Series
	seriesHandleCounter uint32
	strings             map[uint32]string
	stringHandleCounter uint32
}

// ChannelDigest provides metadata about a channel for state initialization.
type ChannelDigest struct {
	DataType telem.DataType
	Key      uint32
	Index    uint32
}

// Config provides dependencies for creating a State instance.
type Config struct {
	IR             ir.IR
	ChannelDigests []ChannelDigest
}

// New creates a state manager from the given configuration.
// It initializes output storage for all node outputs and maps channel keys to their indexes.
func New(cfg Config) *State {
	s := &State{
		cfg:                 cfg,
		outputs:             make(map[ir.Handle]*value),
		indexes:             make(map[uint32]uint32),
		series:              make(map[uint32]telem.Series),
		seriesHandleCounter: 1,
		strings:             make(map[uint32]string),
		stringHandleCounter: 1,
	}
	s.channel.reads = make(map[uint32]telem.MultiSeries)
	s.channel.writes = make(map[uint32]telem.Series)
	for _, d := range cfg.ChannelDigests {
		s.indexes[d.Key] = d.Index
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

// Flush extracts buffered channel writes into a frame and clears the write buffer.
// Returns the updated frame and true if any writes were flushed, or the original frame and false otherwise.
func (s *State) Flush(fr telem.Frame[uint32]) (telem.Frame[uint32], bool) {
	clear(s.series)
	s.seriesHandleCounter = 1
	clear(s.strings)
	s.stringHandleCounter = 1

	if len(s.channel.writes) == 0 {
		return fr, false
	}
	for key, data := range s.channel.writes {
		fr = fr.Append(key, data.DeepCopy())
	}
	clear(s.channel.writes)
	return fr, true
}

// SetAuthority buffers an authority change request for later flushing.
// If channelKey is nil, the change applies to all write channels.
func (s *State) SetAuthority(channelKey *uint32, authority uint8) {
	s.authorityChanges = append(s.authorityChanges, AuthorityChange{
		Channel:   channelKey,
		Authority: authority,
	})
}

// FlushAuthorityChanges returns and clears all buffered authority changes.
func (s *State) FlushAuthorityChanges() []AuthorityChange {
	if len(s.authorityChanges) == 0 {
		return nil
	}
	changes := s.authorityChanges
	s.authorityChanges = nil
	return changes
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

// ClearReads clears accumulated channel read buffers while preserving the latest series for each channel.
// This allows channel_read_* calls to return the most recent value even if new frames for that
// channel haven't arrived yet in the current cycle.
func (s *State) ClearReads() {
	for key, ser := range s.channel.reads {
		if len(ser.Series) <= 1 {
			continue
		}
		// Keep only the last series to preserve the latest value for each channel.
		last := ser.Series[len(ser.Series)-1]
		ser.Series = ser.Series[:1]
		ser.Series[0] = last
		s.channel.reads[key] = ser
	}
}

// Node creates a node-specific state accessor for the given node key.
// It initializes alignment buffers and watermark tracking for the node's inputs.
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
		edge, found := s.cfg.IR.Edges.FindByTarget(ir.Handle{Node: key, Param: p.Name})
		if found {
			inputs[i] = edge
			alignedData[i] = telem.Series{DataType: s.outputs[edge.Source].data.DataType}
			inputSources[i] = s.outputs[edge.Source] // Cache the pointer
		} else {
			// Unconnected input - create synthetic edge pointing to a synthetic source
			syntheticSource := ir.Handle{Node: "__default_" + key + "_" + p.Name, Param: ir.DefaultOutputParam}
			inputs[i] = ir.Edge{Source: syntheticSource, Target: ir.Handle{Node: key, Param: p.Name}}
			data := telem.NewSeriesFromAny(p.Value, types.ToTelem(p.Type))
			time := telem.NewSeriesV[telem.TimeStamp](0)
			alignedData[i] = data
			alignedTime[i] = time
			// Initialize with timestamp 0 and NOT consumed so defaults can trigger
			accumulated[i] = inputEntry{
				data:          data,
				time:          time,
				lastTimestamp: 0,
				consumed:      false,
			}
			if _, exists := s.outputs[syntheticSource]; !exists {
				s.outputs[syntheticSource] = &value{data: data, time: time}
			}
			inputSources[i] = s.outputs[syntheticSource] // Cache the synthetic source pointer
		}
	}

	// Pre-cache output value pointers to avoid map lookups in hot paths
	outputCache := make([]*value, len(n.Outputs))
	for i, p := range n.Outputs {
		handle := ir.Handle{Node: key, Param: p.Name}
		outputCache[i] = s.outputs[handle] // These were created in State.New()
	}

	return &Node{
		inputs: inputs,
		outputs: lo.Map(n.Outputs, func(item types.Param, _ int) ir.Handle {
			return ir.Handle{Node: key, Param: item.Name}
		}),
		state:        s,
		accumulated:  accumulated,
		alignedData:  alignedData,
		alignedTime:  alignedTime,
		inputSources: inputSources,
		outputCache:  outputCache,
	}
}

type inputEntry struct {
	data          telem.Series    // Latest data snapshot
	time          telem.Series    // Latest time snapshot
	lastTimestamp telem.TimeStamp // Last timestamp we've seen (for detecting new data)
	consumed      bool            // Whether this data has triggered execution
}

// Node provides node-specific access to state, handling input alignment and output storage.
type Node struct {
	inputs       []ir.Edge
	outputs      []ir.Handle
	state        *State
	accumulated  []inputEntry
	alignedData  []telem.Series
	alignedTime  []telem.Series
	inputSources []*value // Cached pointers to input sources (avoids map lookups)
	outputCache  []*value // Cached pointers to output values (avoids map lookups)
}

// Reset is called by the scheduler when the stage containing this node is activated.
// Nodes that embed *state.Node inherit this behavior and can override to add
// custom reset logic (calling the embedded Reset() first).
// One-shot edge tracking is handled at the scheduler level, not per-node.
func (n *Node) Reset() {}

// RefreshInputs performs temporal alignment of node inputs and returns whether the node should execute.
//
// Algorithm:
//  1. Snapshot latest series from each input source (series may contain multiple samples)
//  2. Check if any input has new data (timestamp changed)
//  3. If new data exists, align all inputs and provide their series for execution
//  4. Mark all inputs as consumed (series are processed entirely by the node)
//
// Returns true if there's new data ready for execution.
func (n *Node) RefreshInputs() (recalculate bool) {
	if len(n.inputs) == 0 {
		return true
	}

	// Single-pass: snapshot new data, validate all inputs have data, detect unconsumed data
	hasUnconsumed := false
	for i := range n.inputs {
		src := n.inputSources[i]

		// Update snapshot if source has new data (timestamp advanced)
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

		// Early exit if any input has no data
		if n.accumulated[i].data.Len() == 0 {
			return false
		}

		// Track if any input has unconsumed data
		if !n.accumulated[i].consumed {
			hasUnconsumed = true
		}
	}

	// No unconsumed data - all inputs already processed
	if !hasUnconsumed {
		return false
	}

	// Align all inputs and mark as consumed
	for i := range n.inputs {
		n.alignedData[i] = n.accumulated[i].data
		n.alignedTime[i] = n.accumulated[i].time
		n.accumulated[i].consumed = true
	}

	return true
}

// InputTime returns the timestamp series for the input at the given parameter index.
// This is the aligned temporal metadata corresponding to Input(paramIndex).
func (n *Node) InputTime(paramIndex int) telem.Series {
	return n.alignedTime[paramIndex]
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

// Input returns the data series for the input at the given parameter index.
// This is the aligned data prepared by RefreshInputs.
func (n *Node) Input(paramIndex int) telem.Series {
	return n.alignedData[paramIndex]
}

// Output returns a mutable pointer to the data series for the output at the given parameter index.
// Nodes write their computed results to this series.
func (n *Node) Output(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].data
}

// OutputTime returns a mutable pointer to the timestamp series for the output at the given parameter index.
// Nodes write temporal metadata for their outputs to this series.
func (n *Node) OutputTime(paramIndex int) *telem.Series {
	return &n.outputCache[paramIndex].time
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
		return data, telem.MultiSeries{}, len(data.Series) > 0
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

// IsOutputTruthy checks if the output at the given param name is truthy.
// Returns false if the param doesn't exist, if the output is empty,
// or if the last element is zero. Returns true otherwise.
// Used by the scheduler to evaluate one-shot edges - edges only fire
// when the source output is truthy.
func (n *Node) IsOutputTruthy(paramName string) bool {
	for i, h := range n.outputs {
		if h.Param == paramName {
			series := &n.outputCache[i].data
			return isSeriesTruthy(*series)
		}
	}
	return false
}

// isSeriesTruthy checks if a series is truthy by examining its last element.
// Empty series are falsy. A series with a last element of zero is falsy.
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
		// StringT and other types default to falsy
		return false
	}
}

// ReadChannelValue reads a single value from a channel (for WASM runtime bindings).
func (s *State) ReadChannelValue(key uint32) (telem.Series, bool) {
	ms, ok := s.channel.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// WriteChannelValue writes a single value to a channel (for WASM runtime bindings).
// For channels with an index, it auto-generates a timestamp using telem.Now() and writes
// to both the data channel and its index channel, matching the behavior of writeChannel.
func (s *State) WriteChannelValue(key uint32, value telem.Series) {
	s.writeChannel(key, value, telem.NewSeriesV(telem.Now()))
}

// SeriesStore stores a series and returns a handle for later retrieval.
// Handles are transient and are cleared on each Flush call.
func (s *State) SeriesStore(series telem.Series) uint32 {
	handle := s.seriesHandleCounter
	s.seriesHandleCounter++
	s.series[handle] = series
	return handle
}

// SeriesGet retrieves a series by its handle.
// Returns the series and true if found, or an empty series and false if not found.
func (s *State) SeriesGet(handle uint32) (telem.Series, bool) {
	series, ok := s.series[handle]
	return series, ok
}

// StringCreate stores a string and returns a handle for later retrieval.
// Handles are transient and are cleared on each Flush call.
func (s *State) StringCreate(str string) uint32 {
	handle := s.stringHandleCounter
	s.stringHandleCounter++
	s.strings[handle] = str
	return handle
}

// StringGet retrieves a string by its handle.
// Returns the string and true if found, or an empty string and false if not found.
func (s *State) StringGet(handle uint32) (string, bool) {
	str, ok := s.strings[handle]
	return str, ok
}

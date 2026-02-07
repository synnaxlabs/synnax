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

// SeriesHandleStore manages transient series handles.
// Handles are short-lived references used within a single execution cycle
// and cleared on each flush.
type SeriesHandleStore struct {
	series  map[uint32]telem.Series
	counter uint32
}

// NewSeriesHandleStore creates a new SeriesHandleStore.
func NewSeriesHandleStore() *SeriesHandleStore {
	return &SeriesHandleStore{
		series:  make(map[uint32]telem.Series),
		counter: 1,
	}
}

// Store stores a series and returns a handle for later retrieval.
func (s *SeriesHandleStore) Store(series telem.Series) uint32 {
	handle := s.counter
	s.counter++
	s.series[handle] = series
	return handle
}

// Get retrieves a series by its handle.
func (s *SeriesHandleStore) Get(handle uint32) (telem.Series, bool) {
	series, ok := s.series[handle]
	return series, ok
}

// Clear removes all stored series and resets the counter.
func (s *SeriesHandleStore) Clear() {
	clear(s.series)
	s.counter = 1
}

// StringHandleStore manages transient string handles.
// Handles are short-lived references used within a single execution cycle
// and cleared on each flush.
type StringHandleStore struct {
	strings map[uint32]string
	counter uint32
}

// NewStringHandleStore creates a new StringHandleStore.
func NewStringHandleStore() *StringHandleStore {
	return &StringHandleStore{
		strings: make(map[uint32]string),
		counter: 1,
	}
}

// Create stores a string and returns a handle for later retrieval.
func (s *StringHandleStore) Create(str string) uint32 {
	handle := s.counter
	s.counter++
	s.strings[handle] = str
	return handle
}

// Get retrieves a string by its handle.
func (s *StringHandleStore) Get(handle uint32) (string, bool) {
	str, ok := s.strings[handle]
	return str, ok
}

// Clear removes all stored strings and resets the counter.
func (s *StringHandleStore) Clear() {
	clear(s.strings)
	s.counter = 1
}

// AuthorityBuffer buffers authority change requests produced during
// reactive execution for later flushing.
type AuthorityBuffer struct {
	changes []AuthorityChange
}

// NewAuthorityBuffer creates a new AuthorityBuffer.
func NewAuthorityBuffer() *AuthorityBuffer {
	return &AuthorityBuffer{}
}

// Set buffers an authority change request.
// If channelKey is nil, the change applies to all write channels.
func (b *AuthorityBuffer) Set(channelKey *uint32, authority uint8) {
	b.changes = append(b.changes, AuthorityChange{
		Channel:   channelKey,
		Authority: authority,
	})
}

// Flush returns and clears all buffered authority changes.
func (b *AuthorityBuffer) Flush() []AuthorityChange {
	if len(b.changes) == 0 {
		return nil
	}
	changes := b.changes
	b.changes = nil
	return changes
}

// ChannelState manages channel I/O buffers and index mapping.
type ChannelState struct {
	reads   map[uint32]telem.MultiSeries
	writes  map[uint32]telem.Series
	indexes map[uint32]uint32
}

// NewChannelState creates a new ChannelState from channel digests.
func NewChannelState(digests []ChannelDigest) *ChannelState {
	cs := &ChannelState{
		reads:   make(map[uint32]telem.MultiSeries),
		writes:  make(map[uint32]telem.Series),
		indexes: make(map[uint32]uint32),
	}
	for _, d := range digests {
		cs.indexes[d.Key] = d.Index
	}
	return cs
}

// Ingest adds external channel data to the read buffer.
func (cs *ChannelState) Ingest(fr telem.Frame[uint32]) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		cs.reads[key] = cs.reads[key].Append(fr.RawSeriesAt(rawI))
	}
}

// Flush extracts buffered channel writes into a frame and clears the write buffer.
func (cs *ChannelState) Flush(fr telem.Frame[uint32]) (telem.Frame[uint32], bool) {
	if len(cs.writes) == 0 {
		return fr, false
	}
	for key, data := range cs.writes {
		fr = fr.Append(key, data.DeepCopy())
	}
	clear(cs.writes)
	return fr, true
}

// ClearReads clears accumulated channel read buffers while preserving the latest
// series for each channel.
func (cs *ChannelState) ClearReads() {
	for key, ser := range cs.reads {
		if len(ser.Series) <= 1 {
			continue
		}
		last := ser.Series[len(ser.Series)-1]
		ser.Series = ser.Series[:1]
		ser.Series[0] = last
		cs.reads[key] = ser
	}
}

// ReadValue reads a single value from a channel (for WASM runtime bindings).
func (cs *ChannelState) ReadValue(key uint32) (telem.Series, bool) {
	ms, ok := cs.reads[key]
	if !ok || len(ms.Series) == 0 {
		return telem.Series{}, false
	}
	return ms.Series[len(ms.Series)-1], ok
}

// WriteValue writes a single value to a channel (for WASM runtime bindings).
// For channels with an index, it auto-generates a timestamp using telem.Now()
// and writes to both the data channel and its index channel.
func (cs *ChannelState) WriteValue(key uint32, value telem.Series) {
	cs.writeChannel(key, value, telem.NewSeriesV(telem.Now()))
}

// ReadChan reads buffered data and time series from a channel.
func (cs *ChannelState) ReadChan(key uint32) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	data, ok = cs.reads[key]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	indexKey := cs.indexes[key]
	if indexKey == 0 {
		return data, telem.MultiSeries{}, len(data.Series) > 0
	}
	time, ok = cs.reads[indexKey]
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	return data, time, len(time.Series) > 0 && len(data.Series) > 0
}

// WriteChan buffers data and time series for writing to a channel.
func (cs *ChannelState) WriteChan(key uint32, value, time telem.Series) {
	cs.writeChannel(key, value, time)
}

func (cs *ChannelState) writeChannel(key uint32, data, time telem.Series) {
	cs.writes[key] = data
	idx := cs.indexes[key]
	if idx != 0 {
		cs.writes[idx] = time
	}
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

// State manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type State struct {
	Channel *ChannelState
	Series  *SeriesHandleStore
	Strings *StringHandleStore
	Auth    *AuthorityBuffer
	outputs map[ir.Handle]*value
	cfg     Config
}

// New creates a state manager from the given configuration.
// It initializes output storage for all node outputs and maps channel keys to their indexes.
func New(cfg Config) *State {
	s := &State{
		cfg:     cfg,
		outputs: make(map[ir.Handle]*value),
		Channel: NewChannelState(cfg.ChannelDigests),
		Series:  NewSeriesHandleStore(),
		Strings: NewStringHandleStore(),
		Auth:    NewAuthorityBuffer(),
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
	data          telem.Series    // Latest data snapshot
	time          telem.Series    // Latest time snapshot
	lastTimestamp telem.TimeStamp // Last timestamp we've seen (for detecting new data)
	consumed      bool            // Whether this data has triggered execution
}

// Node provides node-specific access to state, handling input alignment and output storage.
type Node struct {
	inputs       []ir.Edge
	outputs      []ir.Handle
	channel      *ChannelState
	nodeOutputs  map[ir.Handle]*value
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
		if v, ok := n.nodeOutputs[sourceHandle]; ok {
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
	return n.channel.ReadChan(key)
}

// WriteChan buffers data and time series for writing to a channel.
// If the channel has an index, the time series is automatically written to the index channel.
func (n *Node) WriteChan(key uint32, value, time telem.Series) {
	n.channel.WriteChan(key, value, time)
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

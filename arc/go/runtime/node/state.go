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
	"github.com/synnaxlabs/x/errors"
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

// New creates a state manager for the given program IR.
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
		isReference  = make([]bool, len(n.Inputs))
	)
	for i := range alignedData {
		alignedTime[i] = telem.Series{DataType: telem.TimeStampT}
	}
	hasEdgeFed := false
	for i, p := range n.Inputs {
		// A channel input is a reference resolved by key in the host functions, not
		// a value stream. It carries no data series and never gates execution.
		if p.Type.Kind == types.KindChan {
			isReference[i] = true
			if edge, found := s.ir.Edges.FindByTarget(
				ir.Handle{Node: key, Param: p.Name},
			); found {
				inputs[i] = edge
				inputSources[i] = s.outputs[edge.Source]
			}
			continue
		}
		// A var input names its variable's node in Type.Name. It binds that
		// node's output slot directly: no edge, never gates or wakes.
		if p.Type.Kind == types.KindVarRef {
			isReference[i] = true
			src := ir.Handle{Node: p.Type.Name, Param: ir.DefaultOutputParam}
			inputs[i] = ir.Edge{
				Source: src,
				Target: ir.Handle{Node: key, Param: p.Name},
			}
			inputSources[i] = s.outputs[src]
			continue
		}
		edge, found := s.ir.Edges.FindByTarget(
			ir.Handle{Node: key, Param: p.Name},
		)
		if found {
			hasEdgeFed = true
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

	// A node with no edge-fed data input never re-arms on its own; its trigger
	// edges register as gating-only entries so each fire re-runs it exactly once.
	// SY-4495: registering unconditionally would make multi-trigger nodes await
	// fresh values on every trigger before running.
	if !hasEdgeFed {
		for _, e := range s.ir.Edges {
			if e.Target.Node != key {
				continue
			}
			if _, ok := n.Inputs.Get(e.Target.Param); ok {
				continue
			}
			inputs = append(inputs, e)
			alignedData = append(alignedData, telem.Series{})
			alignedTime = append(alignedTime, telem.Series{DataType: telem.TimeStampT})
			accumulated = append(accumulated, inputEntry{})
			inputSources = append(inputSources, s.outputs[e.Source])
			isReference = append(isReference, false)
		}
	}

	// Register reads re-arm on fresh values; deref reads on post-entry values;
	// self-write feeders on Reset only.
	rearm := make([]rearmRule, len(inputs))
	for i := range inputs {
		srcNode, found := s.ir.Nodes.Find(inputs[i].Source.Node)
		if !found ||
			(srcNode.Type != "variable" && srcNode.Type != "stateful_variable") {
			continue
		}
		rearm[i] = rearmOnFresh
		if len(srcNode.Inputs) > 0 && srcNode.Inputs[0].Value == nil {
			rearm[i] = rearmOnArrival
		}
		for _, e := range s.ir.Edges {
			if e.Target.Node == srcNode.Key && e.Source.Node == key {
				rearm[i] = rearmOnReset
				break
			}
		}
	}

	// A node that feeds a register it reads by var ref is an entry one-shot:
	// its trigger entries latch on Reset, matching the edge-fed read latch.
	selfWrite := false
	for _, p := range n.Inputs {
		if p.Type.Kind != types.KindVarRef {
			continue
		}
		for _, e := range s.ir.Edges {
			if e.Source.Node == key && e.Target.Node == p.Type.Name {
				selfWrite = true
				break
			}
		}
	}
	if selfWrite {
		for i := len(n.Inputs); i < len(inputs); i++ {
			rearm[i] = rearmOnReset
		}
	}

	outputCache := make([]*value, len(n.Outputs))
	for i, p := range n.Outputs {
		handle := ir.Handle{Node: key, Param: p.Name}
		outputCache[i] = s.outputs[handle]
	}

	inputIndex := make(map[string]int, len(n.Inputs))
	for i, p := range n.Inputs {
		inputIndex[p.Name] = i
	}

	nd := &State{}
	nd.ir.inputs = inputs
	nd.rearm = rearm
	nd.inputIndex = inputIndex
	nd.params = n.Inputs
	nd.ir.outputs = lo.Map(n.Outputs, func(item types.Param, _ int) ir.Handle {
		return ir.Handle{Node: key, Param: item.Name}
	})
	nd.aligned.data = alignedData
	nd.aligned.time = alignedTime
	nd.nodeOutputs = s.outputs
	nd.accumulated = accumulated
	nd.inputSources = inputSources
	nd.outputCache = outputCache
	nd.isReference = isReference
	return nd
}

type inputEntry struct {
	data          telem.Series
	time          telem.Series
	lastTimestamp telem.TimeStamp
	consumed      bool
}

// rearmRule selects when a consumed input re-arms and fires again.
type rearmRule uint8

const (
	// rearmAlways re-arms on scope Reset and on fresh data.
	rearmAlways rearmRule = iota
	// rearmOnFresh re-arms only on fresh data: reads fire on unseen values.
	rearmOnFresh
	// rearmOnReset re-arms only on scope Reset: self-writes fire once per entry.
	rearmOnReset
	// rearmOnArrival absorbs pending data on Reset: only post-entry values fire.
	rearmOnArrival
)

// State provides node-specific access to state, handling input alignment and
// output storage.
type State struct {
	ir struct {
		inputs  []ir.Edge
		outputs []ir.Handle
	}
	// inputIndex maps an input's parameter name to its position.
	inputIndex map[string]int
	// params holds the node's input params with their configured values.
	params types.Params
	// isReference marks inputs that are channel references rather than value
	// streams. Reference inputs carry no data series and never gate execution.
	isReference []bool
	// rearm[i] selects when a consumed input i fires again.
	rearm       []rearmRule
	accumulated []inputEntry
	aligned     struct {
		data []telem.Series
		time []telem.Series
	}
	nodeOutputs  map[ir.Handle]*value
	inputSources []*value
	outputCache  []*value
}

// Reset re-arms every input when the node's stage is (re)activated, so a node
// whose gating inputs are all literal-valued re-runs instead of staying consumed.
func (n *State) Reset() {
	for i := range n.accumulated {
		switch n.rearm[i] {
		case rearmOnFresh:
		case rearmOnArrival:
			n.absorbInput(i)
		case rearmAlways, rearmOnReset:
			n.accumulated[i].consumed = false
			n.accumulated[i].lastTimestamp = 0
		}
	}
}

// RefreshInputs performs temporal alignment of node inputs and returns whether
// the node should execute.
func (n *State) RefreshInputs() (recalculate bool) {
	hasDataInput, hasUnconsumed := false, false
	for i := range n.ir.inputs {
		if n.isReference[i] {
			continue
		}
		hasDataInput = true
		src := n.inputSources[i]
		if src != nil && src.time.Len() > 0 {
			ts := telem.ValueAt[telem.TimeStamp](src.time, -1)
			if ts > n.accumulated[i].lastTimestamp {
				consumed := false
				if n.rearm[i] == rearmOnReset {
					consumed = n.accumulated[i].consumed
				}
				n.accumulated[i] = inputEntry{
					data:          src.data,
					time:          src.time,
					lastTimestamp: ts,
					consumed:      consumed,
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
	if !hasDataInput {
		return true
	}
	if !hasUnconsumed {
		return false
	}
	for i := range n.ir.inputs {
		if n.isReference[i] {
			continue
		}
		n.aligned.data[i] = n.accumulated[i].data
		n.aligned.time[i] = n.accumulated[i].time
		n.accumulated[i].consumed = true
	}
	return true
}

// RefSourced reports whether the reference input at paramIndex is edge-fed.
func (n *State) RefSourced(paramIndex int) bool {
	return paramIndex >= 0 && paramIndex < len(n.inputSources) &&
		n.isReference[paramIndex] && n.inputSources[paramIndex] != nil
}

// RefInput returns the current data of an edge-fed reference input, or an
// empty series when the input is unedged.
func (n *State) RefInput(paramIndex int) telem.Series {
	if paramIndex >= 0 && paramIndex < len(n.inputSources) && n.isReference[paramIndex] {
		if src := n.inputSources[paramIndex]; src != nil {
			return src.data
		}
	}
	return telem.Series{}
}

// StringInput returns the named input's current value: the referenced
// variable's value when var-bound (its declared initial until first written),
// else the configured value.
func (n *State) StringInput(name string) string {
	i, err := n.ResolveInput(name)
	if err != nil {
		return ""
	}
	if s := n.RefInput(i); s.Len() > 0 {
		return string(s.At(-1))
	}
	if v, ok := n.params[i].Value.(string); ok {
		return v
	}
	return ""
}

// NumericInput returns the named input's current value: the referenced
// variable's value when var-bound (its declared initial until first written),
// else the configured value.
func NumericInput[T telem.NumericSample](n *State, name string) T {
	i, err := n.ResolveInput(name)
	if err != nil {
		return 0
	}
	if s := n.RefInput(i); s.Len() > 0 {
		return telem.ValueAt[T](s, -1)
	}
	if v := n.params[i].Value; v != nil {
		return telem.CastNumeric[T](v)
	}
	return 0
}

// AbsorbInputs marks every data input consumed at its current source timestamp,
// so only writes after this call re-fire the node.
func (n *State) AbsorbInputs() {
	for i := range n.ir.inputs {
		n.absorbInput(i)
	}
}

// absorbInput marks input i consumed at its current source timestamp.
func (n *State) absorbInput(i int) {
	if n.isReference[i] {
		return
	}
	src := n.inputSources[i]
	if src == nil {
		return
	}
	var ts telem.TimeStamp
	if src.time.Len() > 0 {
		ts = telem.ValueAt[telem.TimeStamp](src.time, -1)
	}
	n.accumulated[i] = inputEntry{
		data:          src.data,
		time:          src.time,
		lastTimestamp: ts,
		consumed:      true,
	}
}

// ConsumeInput returns input i's unconsumed data, marking it consumed. ok is
// false when input i is a reference or has no new data.
func (n *State) ConsumeInput(i int) (telem.Series, bool) {
	if i < 0 || i >= len(n.ir.inputs) || n.isReference[i] {
		return telem.Series{}, false
	}
	src := n.inputSources[i]
	if src == nil || src.data.Len() == 0 {
		return telem.Series{}, false
	}
	var ts telem.TimeStamp
	if src.time.Len() > 0 {
		ts = telem.ValueAt[telem.TimeStamp](src.time, -1)
	}
	if ts <= n.accumulated[i].lastTimestamp && n.accumulated[i].consumed {
		return telem.Series{}, false
	}
	n.accumulated[i] = inputEntry{
		data:          src.data,
		time:          src.time,
		lastTimestamp: ts,
		consumed:      true,
	}
	return src.data, true
}

// InputFresh reports whether input i has unconsumed data, without consuming it.
func (n *State) InputFresh(i int) bool {
	if i < 0 || i >= len(n.ir.inputs) || n.isReference[i] {
		return false
	}
	src := n.inputSources[i]
	if src == nil || src.data.Len() == 0 {
		return false
	}
	var ts telem.TimeStamp
	if src.time.Len() > 0 {
		ts = telem.ValueAt[telem.TimeStamp](src.time, -1)
	}
	return ts > n.accumulated[i].lastTimestamp || !n.accumulated[i].consumed
}

// LastChanged returns the series of the most-recently-changed input, marking it
// consumed for last-write-wins. ok is false when no input has new data.
func (n *State) LastChanged() (telem.Series, bool) {
	best, bestTS, found := -1, telem.TimeStamp(0), false
	for i := range n.ir.inputs {
		if n.isReference[i] {
			continue
		}
		src := n.inputSources[i]
		if src == nil || src.data.Len() == 0 {
			continue
		}
		var ts telem.TimeStamp
		if src.time.Len() > 0 {
			ts = telem.ValueAt[telem.TimeStamp](src.time, -1)
		}
		if ts <= n.accumulated[i].lastTimestamp && n.accumulated[i].consumed {
			continue
		}
		if !found || ts > bestTS {
			best, bestTS, found = i, ts, true
		}
	}
	if !found {
		return telem.Series{}, false
	}
	src := n.inputSources[best]
	n.accumulated[best] = inputEntry{
		data:          src.data,
		time:          src.time,
		lastTimestamp: bestTS,
		consumed:      true,
	}
	return src.data, true
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

// ErrInputNotFound is returned by ResolveInput when a node has no input param
// matching the requested name.
var ErrInputNotFound = errors.New("input not found")

// ResolveInput returns the position of the named input, or ErrInputNotFound if
// the node has no such param. Resolve at construction so wiring mistakes fail at
// load.
func (n *State) ResolveInput(name string) (int, error) {
	idx, ok := n.inputIndex[name]
	if !ok {
		return 0, errors.Wrapf(ErrInputNotFound, "node has no input named %q", name)
	}
	return idx, nil
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

// IsOutputTruthy reports whether the output at the given 0-based ordinal
// is truthy. Out-of-range ordinals report false.
func (n *State) IsOutputTruthy(outputIdx int) bool {
	if outputIdx < 0 || outputIdx >= len(n.outputCache) {
		return false
	}
	return isSeriesTruthy(n.outputCache[outputIdx].data)
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
	case telem.StringT:
		return len(s.At(-1)) > 0
	default:
		return false
	}
}

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
	// rev is the program-wide revision at which this value was last written.
	// Zero means never written. Consumers compare it against what they last
	// consumed; ordering across values is the program's true write order.
	rev uint64
}

// ProgramState manages runtime data for an arc program.
// It stores node outputs, channel I/O buffers, and index relationships.
type ProgramState struct {
	ir      ir.IR
	outputs map[ir.Handle]*value
	// rev counts writes across every output in the program. Every cycle stamps
	// one timestamp, so the stamp cannot tell a consumer that a value is new;
	// this counter does.
	rev uint64
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
				data: telem.Series{DataType: p.Type.ToTelem()},
				time: telem.Series{DataType: telem.TimestampT},
			}
		}
	}
	return s
}

// Node creates a node-specific state accessor for the given node key.
// It initializes alignment buffers and watermark tracking for the node's
// inputs.
func (s *ProgramState) Node(key string) *State {
	n := s.ir.Nodes.Get(key)
	nd := &State{params: n.Inputs, nodeOutputs: s.outputs, progRev: &s.rev}
	if !s.bindInputs(nd, key, n.Inputs) {
		s.appendTriggerInputs(nd, key, n.Inputs)
	}
	nd.rearm = s.rearmRules(key, n.Inputs, nd.ir.inputs)
	nd.stale = make([]bool, len(nd.ir.inputs))
	nd.outputCache = make([]*value, len(n.Outputs))
	for i, p := range n.Outputs {
		nd.outputCache[i] = s.outputs[ir.Handle{Node: key, Param: p.Name}]
	}
	nd.inputIndex = make(map[string]int, len(n.Inputs))
	for i, p := range n.Inputs {
		nd.inputIndex[p.Name] = i
	}
	nd.ir.outputs = lo.Map(n.Outputs, func(item types.Param, _ int) ir.Handle {
		return ir.Handle{Node: key, Param: item.Name}
	})
	for i := range nd.literal {
		if !nd.literal[i] && !nd.isReference[i] {
			nd.edgeFed = true
			break
		}
	}
	return nd
}

// bindInputs fills one input slot per parameter and reports whether an edge feeds
// any data input.
func (s *ProgramState) bindInputs(nd *State, key string, params types.Params) bool {
	count := len(params)
	nd.ir.inputs = make([]ir.Edge, count)
	nd.aligned.data = make([]telem.Series, count)
	nd.aligned.time = make([]telem.Series, count)
	nd.accumulated = make([]inputEntry, count)
	nd.literal = make([]bool, count)
	nd.inputSources = make([]*value, count)
	nd.isReference = make([]bool, count)
	for i := range nd.aligned.time {
		nd.aligned.time[i] = telem.Series{DataType: telem.TimestampT}
	}
	hasEdgeFed := false
	for i, p := range params {
		target := ir.Handle{Node: key, Param: p.Name}
		// A channel input is a reference resolved by key in the host functions, not
		// a value stream. It carries no data series and never gates execution.
		if p.Type.Kind == types.KindChan {
			nd.isReference[i] = true
			if edge, found := s.ir.Edges.FindByTarget(target); found {
				nd.ir.inputs[i] = edge
				nd.inputSources[i] = s.outputs[edge.Source]
			}
			continue
		}
		// A var input names its variable's node in Type.Name. It binds that
		// node's output slot directly: no edge, never gates or wakes.
		if p.Type.Kind == types.KindVarRef {
			nd.isReference[i] = true
			src := ir.Handle{Node: p.Type.Name, Param: ir.DefaultOutputParam}
			nd.ir.inputs[i] = ir.Edge{Source: src, Target: target}
			nd.inputSources[i] = s.outputs[src]
			continue
		}
		edge, found := s.ir.Edges.FindByTarget(target)
		if !found {
			s.bindLiteralInput(nd, i, target, p)
			continue
		}
		hasEdgeFed = true
		nd.ir.inputs[i] = edge
		nd.aligned.data[i] = telem.Series{
			DataType: s.outputs[edge.Source].data.DataType,
		}
		nd.inputSources[i] = s.outputs[edge.Source]
	}
	return hasEdgeFed
}

// bindLiteralInput feeds an input with no edge from a synthetic source that holds
// its default value.
func (s *ProgramState) bindLiteralInput(
	nd *State,
	i int,
	target ir.Handle,
	p types.Param,
) {
	syntheticSource := ir.Handle{
		Node:  "__default_" + target.Node + "_" + p.Name,
		Param: ir.DefaultOutputParam,
	}
	nd.ir.inputs[i] = ir.Edge{Source: syntheticSource, Target: target}
	data := telem.NewSeriesFromAny(types.DecodeValue(p.Value, p.Type), p.Type.ToTelem())
	time := telem.NewSeriesV[telem.TimeStamp](0)
	nd.aligned.data[i] = data
	nd.aligned.time[i] = time
	nd.accumulated[i] = inputEntry{data: data, time: time}
	nd.literal[i] = true
	if _, exists := s.outputs[syntheticSource]; !exists {
		s.outputs[syntheticSource] = &value{data: data, time: time}
	}
	nd.inputSources[i] = s.outputs[syntheticSource]
}

// appendTriggerInputs registers a node's trigger edges as gating-only entries. A
// node with no edge-fed data input never re-arms on its own, so each fire re-runs
// it exactly once. SY-4495: registering unconditionally would make multi-trigger
// nodes await fresh values on every trigger before running.
func (s *ProgramState) appendTriggerInputs(
	nd *State,
	key string,
	params types.Params,
) {
	for _, e := range s.ir.Edges {
		if e.Target.Node != key {
			continue
		}
		if _, ok := params.Get(e.Target.Param); ok {
			continue
		}
		nd.ir.inputs = append(nd.ir.inputs, e)
		nd.aligned.data = append(nd.aligned.data, telem.Series{})
		nd.aligned.time = append(
			nd.aligned.time,
			telem.Series{DataType: telem.TimestampT},
		)
		nd.accumulated = append(nd.accumulated, inputEntry{})
		nd.inputSources = append(nd.inputSources, s.outputs[e.Source])
		nd.isReference = append(nd.isReference, false)
		nd.literal = append(nd.literal, false)
	}
}

// rearmRules picks each input's re-arm rule. Register reads re-arm on fresh
// values; deref reads on post-entry values; self-write feeders on Reset only.
func (s *ProgramState) rearmRules(
	key string,
	params types.Params,
	inputs []ir.Edge,
) []rearmRule {
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
		if s.feeds(key, srcNode.Key) {
			rearm[i] = rearmOnReset
		}
	}
	// A node that feeds a register it reads by var ref is an entry one-shot: its
	// trigger entries latch on Reset, matching the edge-fed read latch.
	for _, p := range params {
		if p.Type.Kind != types.KindVarRef || !s.feeds(key, p.Type.Name) {
			continue
		}
		for i := len(params); i < len(inputs); i++ {
			rearm[i] = rearmOnReset
		}
		break
	}
	return rearm
}

// feeds reports whether an edge runs from the source node to the target node.
func (s *ProgramState) feeds(source, target string) bool {
	for _, e := range s.ir.Edges {
		if e.Source.Node == source && e.Target.Node == target {
			return true
		}
	}
	return false
}

type inputEntry struct {
	data telem.Series
	time telem.Series
	// lastRev is the source value's revision when this entry was taken.
	lastRev  uint64
	consumed bool
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
	// literal marks inputs fed by a configured value rather than an edge. A
	// configured value has no time of its own.
	literal []bool
	// edgeFed is true when an edge feeds at least one data input. Reset leaves the
	// literals of such a node consumed, so only fresh edge data re-runs it.
	edgeFed bool
	// rearm[i] selects when a consumed input i fires again.
	rearm []rearmRule
	// stale[i] marks an input whose series an earlier run consumed. It holds only
	// its last sample until new data arrives.
	stale       []bool
	accumulated []inputEntry
	aligned     struct {
		data []telem.Series
		time []telem.Series
	}
	nodeOutputs  map[ir.Handle]*value
	inputSources []*value
	outputCache  []*value
	// progRev points at the owning ProgramState's write counter.
	progRev *uint64
}

// Emit publishes the value the node just wrote to the output at paramIndex:
// downstream readers see it as unconsumed, and the scheduler runs them. Every
// producer calls it once per write.
func (s *State) Emit(ctx Context, paramIndex int) {
	s.MarkFresh(paramIndex)
	ctx.MarkChanged(paramIndex)
}

// MarkFresh makes the output at paramIndex unconsumed for downstream readers
// without waking them. A cycle stamps one timestamp on everything it produces,
// so a reader cannot tell a new value from the one it already consumed; the
// revision this records is what tells it. Producers writing during Next call
// Emit instead; this is for a write on Reset, which has no running node for the
// scheduler to propagate from.
func (s *State) MarkFresh(paramIndex int) {
	*s.progRev++
	s.outputCache[paramIndex].rev = *s.progRev
}

// Reset re-arms the node's inputs when its stage is (re)activated, so a node
// whose inputs are all literal-valued re-runs instead of staying consumed. An
// edge-fed input keeps what it consumed: re-arming one makes the node re-emit a
// value it already emitted, which duplicates writes downstream.
func (s *State) Reset(Context) {
	for i := range s.accumulated {
		switch s.rearm[i] {
		case rearmOnFresh:
		case rearmOnArrival:
			s.absorbInput(i)
		case rearmAlways:
			if !s.literal[i] || s.edgeFed {
				continue
			}
			s.accumulated[i].consumed = false
			s.accumulated[i].lastRev = 0
		case rearmOnReset:
			s.accumulated[i].consumed = false
			s.accumulated[i].lastRev = 0
		}
	}
}

// RefreshInputs performs temporal alignment of node inputs and returns whether
// the node should execute.
func (s *State) RefreshInputs() (recalculate bool) {
	hasDataInput, hasUnconsumed := false, false
	for i := range s.ir.inputs {
		if s.isReference[i] {
			continue
		}
		hasDataInput = true
		src := s.inputSources[i]
		refreshed := src != nil && src.rev > s.accumulated[i].lastRev
		s.stale[i] = !refreshed && s.accumulated[i].consumed
		if refreshed {
			consumed := false
			if s.rearm[i] == rearmOnReset {
				consumed = s.accumulated[i].consumed
			}
			s.accumulated[i] = inputEntry{
				data:     src.data,
				time:     src.time,
				lastRev:  src.rev,
				consumed: consumed,
			}
		}
		if s.accumulated[i].data.Len() == 0 {
			return false
		}
		if !s.accumulated[i].consumed {
			hasUnconsumed = true
		}
	}
	if !hasDataInput {
		return true
	}
	if !hasUnconsumed {
		return false
	}
	for i := range s.ir.inputs {
		if s.isReference[i] {
			continue
		}
		data, time := s.accumulated[i].data, s.accumulated[i].time
		if s.stale[i] {
			if data.Len() > 1 {
				data = data.Last()
			}
			if time.Len() > 1 {
				time = time.Last()
			}
		}
		s.aligned.data[i] = data
		s.aligned.time[i] = time
		s.accumulated[i].consumed = true
	}
	return true
}

// InputStale reports whether the input at paramIndex was consumed by an earlier run.
func (s *State) InputStale(paramIndex int) bool {
	return paramIndex >= 0 && paramIndex < len(s.stale) && s.stale[paramIndex]
}

// RefSourced reports whether the reference input at paramIndex is edge-fed.
func (s *State) RefSourced(paramIndex int) bool {
	return paramIndex >= 0 && paramIndex < len(s.inputSources) &&
		s.isReference[paramIndex] && s.inputSources[paramIndex] != nil
}

// RefInput returns the current data of an edge-fed reference input, or an
// empty series when the input is unedged.
func (s *State) RefInput(paramIndex int) telem.Series {
	if paramIndex >= 0 && paramIndex < len(s.inputSources) &&
		s.isReference[paramIndex] {
		if src := s.inputSources[paramIndex]; src != nil {
			return src.data
		}
	}
	return telem.Series{}
}

// StringInput returns the named input's current value: the referenced
// variable's value when var-bound (its declared initial until first written),
// else the configured value.
func (s *State) StringInput(name string) string {
	i, err := s.ResolveInput(name)
	if err != nil {
		return ""
	}
	if ref := s.RefInput(i); ref.Len() > 0 {
		return string(ref.At(-1))
	}
	if v, ok := s.params[i].Value.(string); ok {
		return v
	}
	return ""
}

// NumericInput returns the named input's current value: the referenced
// variable's value when var-bound (its declared initial until first written),
// else the configured value.
func (s *State) NumericInput[T telem.NumericSample](name string) T {
	i, err := s.ResolveInput(name)
	if err != nil {
		return 0
	}
	if ref := s.RefInput(i); ref.Len() > 0 {
		return ref.ValueAt[T](-1)
	}
	if p := s.params[i]; p.Value != nil {
		return telem.CastNumeric[T](types.DecodeValue(p.Value, p.Type))
	}
	return 0
}

// AbsorbInputs marks every data input consumed at its current source timestamp,
// so only writes after this call re-fire the node.
func (s *State) AbsorbInputs() {
	for i := range s.ir.inputs {
		s.absorbInput(i)
	}
}

// absorbInput marks input i consumed at its current source revision.
func (s *State) absorbInput(i int) {
	if s.isReference[i] {
		return
	}
	src := s.inputSources[i]
	if src == nil {
		return
	}
	s.accumulated[i] = inputEntry{
		data:     src.data,
		time:     src.time,
		lastRev:  src.rev,
		consumed: true,
	}
}

// ConsumeInput returns input i's unconsumed data, marking it consumed. ok is
// false when input i is a reference or has no new data.
func (s *State) ConsumeInput(i int) (telem.Series, bool) {
	if i < 0 || i >= len(s.ir.inputs) || s.isReference[i] {
		return telem.Series{}, false
	}
	src := s.inputSources[i]
	if src == nil || src.data.Len() == 0 {
		return telem.Series{}, false
	}
	if src.rev <= s.accumulated[i].lastRev && s.accumulated[i].consumed {
		return telem.Series{}, false
	}
	s.accumulated[i] = inputEntry{
		data:     src.data,
		time:     src.time,
		lastRev:  src.rev,
		consumed: true,
	}
	return src.data, true
}

// InputFresh reports whether input i has unconsumed data, without consuming it.
func (s *State) InputFresh(i int) bool {
	if i < 0 || i >= len(s.ir.inputs) || s.isReference[i] {
		return false
	}
	src := s.inputSources[i]
	if src == nil || src.data.Len() == 0 {
		return false
	}
	return src.rev > s.accumulated[i].lastRev || !s.accumulated[i].consumed
}

// LastChanged returns the series of the most-recently-changed input, marking it
// consumed for last-write-wins. ok is false when no input has new data.
func (s *State) LastChanged() (telem.Series, bool) {
	best, bestRev, found := -1, uint64(0), false
	for i := range s.ir.inputs {
		if s.isReference[i] {
			continue
		}
		src := s.inputSources[i]
		if src == nil || src.data.Len() == 0 {
			continue
		}
		if src.rev <= s.accumulated[i].lastRev && s.accumulated[i].consumed {
			continue
		}
		if !found || src.rev > bestRev {
			best, bestRev, found = i, src.rev, true
		}
	}
	if !found {
		return telem.Series{}, false
	}
	src := s.inputSources[best]
	s.accumulated[best] = inputEntry{
		data:     src.data,
		time:     src.time,
		lastRev:  bestRev,
		consumed: true,
	}
	return src.data, true
}

// TimeSourceIdx returns the index of the input a node copies its output timestamps
// from, or -1 when no input has time and the node stamps the cycle instead. Among the
// inputs that have time it picks the longest, matching how nodes broadcast a shorter
// input up to a longer one.
func (s *State) TimeSourceIdx() int {
	best, bestLen := -1, int64(0)
	for i := range s.ir.inputs {
		if !s.HasTime(i) {
			continue
		}
		if l := s.aligned.data[i].Len(); l > bestLen {
			best, bestLen = i, l
		}
	}
	return best
}

// HasTime reports whether the input at paramIndex holds upstream timestamps a node
// can copy. Literal and reference inputs never do: a configured value has no time.
func (s *State) HasTime(paramIndex int) bool {
	return !s.isReference[paramIndex] && !s.literal[paramIndex] &&
		s.aligned.time[paramIndex].Len() > 0
}

// StampCycle overwrites the output's time series with a single sample of the
// cycle stamp, reusing its buffer. Nodes with no input time to copy use it.
func (s *State) StampCycle(ctx Context, outputIdx int) {
	t := s.OutputTime(outputIdx)
	t.Resize(1)
	t.SetValueAt(0, ctx.Now)
}

// InputTime returns the timestamp series for the input at the given parameter
// index.
func (s *State) InputTime(paramIndex int) telem.Series {
	return s.aligned.time[paramIndex]
}

// InitInput initializes an input's source output with dummy values.
func (s *State) InitInput(paramIndex int, data, time telem.Series) {
	if paramIndex >= 0 && paramIndex < len(s.ir.inputs) {
		sourceHandle := s.ir.inputs[paramIndex].Source
		if v, ok := s.nodeOutputs[sourceHandle]; ok {
			v.data = data
			v.time = time
			*s.progRev++
			v.rev = *s.progRev
		}
	}
}

// Input returns the data series for the input at the given parameter index.
func (s *State) Input(paramIndex int) telem.Series {
	return s.aligned.data[paramIndex]
}

// ErrInputNotFound is returned by ResolveInput when a node has no input param
// matching the requested name.
var ErrInputNotFound = errors.New("input not found")

// ResolveInput returns the position of the named input, or ErrInputNotFound if
// the node has no such param. Resolve at construction so wiring mistakes fail at
// load.
func (s *State) ResolveInput(name string) (int, error) {
	idx, ok := s.inputIndex[name]
	if !ok {
		return 0, errors.Wrapf(ErrInputNotFound, "node has no input named %q", name)
	}
	return idx, nil
}

// Output returns a mutable pointer to the data series for the output at the
// given parameter index.
func (s *State) Output(paramIndex int) *telem.Series {
	return &s.outputCache[paramIndex].data
}

// OutputTime returns a mutable pointer to the timestamp series for the output
// at the given parameter index.
func (s *State) OutputTime(paramIndex int) *telem.Series {
	return &s.outputCache[paramIndex].time
}

// IsOutputTruthy reports whether the output at the given 0-based ordinal
// is truthy. Out-of-range ordinals report false.
func (s *State) IsOutputTruthy(outputIdx int) bool {
	if outputIdx < 0 || outputIdx >= len(s.outputCache) {
		return false
	}
	return isSeriesTruthy(s.outputCache[outputIdx].data)
}

func isSeriesTruthy(s telem.Series) bool {
	if s.Len() == 0 {
		return false
	}
	dt := s.DataType
	switch dt {
	case telem.Float64T:
		return s.ValueAt[float64](-1) != 0
	case telem.Float32T:
		return s.ValueAt[float32](-1) != 0
	case telem.Int64T:
		return s.ValueAt[int64](-1) != 0
	case telem.Int32T:
		return s.ValueAt[int32](-1) != 0
	case telem.Int16T:
		return s.ValueAt[int16](-1) != 0
	case telem.Int8T:
		return s.ValueAt[int8](-1) != 0
	case telem.Uint64T:
		return s.ValueAt[uint64](-1) != 0
	case telem.Uint32T:
		return s.ValueAt[uint32](-1) != 0
	case telem.Uint16T:
		return s.ValueAt[uint16](-1) != 0
	case telem.Uint8T:
		return s.ValueAt[uint8](-1) != 0
	case telem.BooleanT:
		return s.ValueAt[bool](-1)
	case telem.TimestampT:
		return s.ValueAt[telem.TimeStamp](-1) != 0
	case telem.StringT:
		return len(s.At(-1)) > 0
	default:
		return false
	}
}

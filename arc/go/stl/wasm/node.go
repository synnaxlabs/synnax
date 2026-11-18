// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

var _ node.Node = (*nodeImpl)(nil)

// NodeKeySetter is implemented by modules that need to know which node is
// currently executing (e.g., stateful variable scoping). The runtime calls
// SetNodeKey before each WASM invocation. This follows the same optional
// interface pattern as MemorySetter.
type NodeKeySetter interface {
	SetNodeKey(key string)
	ClearNode(key string)
}

type result struct {
	Value   uint64
	Changed bool
}

type nodeImpl struct {
	*node.State
	ir            ir.Node
	fn            api.Function
	mem           api.Memory
	fnOutputs     types.Params
	memOffsets    []uint32
	outputValues  []result
	memBase       uint32
	params        []uint64
	stack         []uint64
	offsets       []int
	selIdx        int
	nodeKeySetter NodeKeySetter
	stringInputs  []bool
	chanInputs    []bool
	varInputs     []bool
	stringOutputs []bool
	strings       *stlstrings.ProgramState
	// batch is nil when the node's signature does not admit vectorization.
	batch *batchCall
}

// trapCause returns the error that wazero wrapped in its stack trace, or err if none.
func trapCause(err error) error {
	var wrapper interface{ Unwrap() error }
	if errors.As(err, &wrapper) {
		return wrapper.Unwrap()
	}
	return err
}

func (n *nodeImpl) call(ctx context.Context) ([]result, error) {
	for i := range n.outputValues {
		n.outputValues[i].Changed = false
	}
	copy(n.stack, n.params)
	if err := n.fn.CallWithStack(ctx, n.stack); err != nil {
		return nil, trapCause(err)
	}
	if n.memBase == 0 {
		if len(n.outputValues) > 0 {
			n.outputValues[0] = result{Value: n.stack[0], Changed: true}
		}
		return n.outputValues, nil
	}
	dirtyFlags := lo.Must(n.mem.ReadUint64Le(n.memBase))
	for i := range n.fnOutputs {
		if (dirtyFlags & (1 << i)) != 0 {
			n.outputValues[i] = result{
				Value:   lo.Must(n.mem.ReadUint64Le(n.memOffsets[i])),
				Changed: true,
			}
		}
	}
	return n.outputValues, nil
}

func (n *nodeImpl) Init(node.Context) {}

// dataFresh reports whether any input other than $sel has unconsumed data.
func (n *nodeImpl) dataFresh() bool {
	for i := range n.ir.Inputs {
		if i == n.selIdx || n.ir.Inputs[i].Value != nil || n.chanInputs[i] ||
			n.varInputs[i] {
			continue
		}
		if n.InputFresh(i) {
			return true
		}
	}
	return false
}

func (n *nodeImpl) Next(ctx node.Context) {
	defer func() {
		if r := recover(); r != nil {
			ctx.ReportError(errors.Newf("WASM trap in node %s: %v", n.ir.Key, r))
		}
	}()

	// A $sel-only change re-points without emitting; the value fires on the next input.
	if n.selIdx >= 0 && !n.dataFresh() {
		n.RefreshInputs()
		return
	}

	if !n.RefreshInputs() || !n.readRefInputs() {
		return
	}

	maxLength, longestInputIdx := n.longestInput()
	if maxLength <= 0 {
		return
	}
	for j := range n.offsets {
		n.offsets[j] = 0
	}
	stringResults := n.resizeOutputs(maxLength)
	n.copyInputBounds()
	// Dispatcher drivers alternate; no input's time is honest, so stamp the clock.
	stamps := sampleStamps{clock: longestInputIdx < 0 || n.selIdx >= 0}
	if longestInputIdx >= 0 {
		stamps.input = n.InputTime(longestInputIdx)
	}
	if stamps.clock {
		stamps.clockStart = ctx.ReserveStamps(int(maxLength))
	}
	if n.nodeKeySetter != nil {
		n.nodeKeySetter.SetNodeKey(n.ir.Key)
	}
	batched := n.batch != nil && maxLength > 1 && n.runBatch(ctx, maxLength, stamps)
	trapped := !batched && n.runSamples(ctx, maxLength, stamps, stringResults)
	n.emitOutputs(ctx, stringResults)
	if !batched && !trapped {
		ctx.MarkDone()
	}
}

// readRefInputs re-reads the latest value of each channel and variable reference
// input, reporting false when one has no data yet. A KindChan param holds the key of
// the channel the body targets, and the key is edge-fed and can rebind at runtime.
func (n *nodeImpl) readRefInputs() bool {
	for i := range n.ir.Inputs {
		if !n.chanInputs[i] || n.ir.Inputs[i].Value != nil {
			continue
		}
		t := n.RefInput(i)
		if t.Len() == 0 {
			return false
		}
		n.params[i] = uint64(t.ValueAt[uint32](-1))
	}
	for i := range n.ir.Inputs {
		if !n.varInputs[i] {
			continue
		}
		t := n.RefInput(i)
		if t.Len() == 0 {
			return false
		}
		if n.stringInputs[i] {
			n.params[i] = uint64(n.strings.Create(string(t.At(-1))))
		} else {
			n.params[i] = valueAt(t, int(t.Len()-1))
		}
	}
	return true
}

func (n *nodeImpl) streamed(i int) bool {
	return n.ir.Inputs[i].Value == nil && !n.chanInputs[i] && !n.varInputs[i]
}

// longestInput returns the number of samples to compute and the index of the streamed
// input that sets it, or -1 when no input is streamed.
func (n *nodeImpl) longestInput() (int64, int) {
	maxLength := int64(0)
	longestInputIdx := -1
	for i := range n.ir.Inputs {
		if !n.streamed(i) {
			continue
		}
		dataLen := n.Input(i).Len()
		// A fresh input wins a tie, so stamps come from new data.
		if dataLen > maxLength ||
			(dataLen == maxLength && !n.InputStale(i) && n.InputStale(longestInputIdx)) {
			maxLength = dataLen
			longestInputIdx = i
		}
	}
	// With no edge-fed inputs, the node executes once over its literal inputs.
	if longestInputIdx == -1 {
		maxLength = 1
	}
	return maxLength, longestInputIdx
}

// resizeOutputs pre-sizes numeric outputs so setValueAt can do fixed-stride writes per
// sample. String outputs are variable-density, so it returns a buffer to accumulate
// them in instead.
func (n *nodeImpl) resizeOutputs(maxLength int64) [][]string {
	var stringResults [][]string
	for i := range n.ir.Outputs {
		if n.stringOutputs[i] {
			if stringResults == nil {
				stringResults = make([][]string, len(n.ir.Outputs))
			}
			stringResults[i] = make([]string, 0, maxLength)
		} else {
			out := n.Output(i)
			*out = out.Resized(maxLength)
		}
		outTime := n.OutputTime(i)
		*outTime = outTime.Resized(maxLength)
	}
	return stringResults
}

// copyInputBounds copies alignment and time range from the inputs to the outputs.
// Alignments are summed to guarantee uniqueness across different input sources.
func (n *nodeImpl) copyInputBounds() {
	var alignmentSum telem.Alignment
	var timeRange telem.TimeRange
	for i := range n.ir.Inputs {
		if !n.streamed(i) {
			continue
		}
		input := n.Input(i)
		alignmentSum += input.Alignment
		if timeRange.Start.IsZero() || input.TimeRange.Start < timeRange.Start {
			timeRange.Start = input.TimeRange.Start
		}
		if input.TimeRange.End > timeRange.End {
			timeRange.End = input.TimeRange.End
		}
	}
	for i := range n.ir.Outputs {
		n.Output(i).Alignment = alignmentSum
		n.Output(i).TimeRange = timeRange
		n.OutputTime(i).Alignment = alignmentSum
		n.OutputTime(i).TimeRange = timeRange
	}
}

// sampleStamps picks the timestamp of each output sample: the longest input's, or a
// run reserved from the clock.
type sampleStamps struct {
	input      telem.Series
	clockStart telem.TimeStamp
	clock      bool
}

func (s sampleStamps) at(i int64) uint64 {
	if s.clock {
		return uint64(s.clockStart) + uint64(i)
	}
	return valueAt(s.input, int(i))
}

// runSamples calls the guest once per sample, reporting whether a trap ended the
// cycle.
func (n *nodeImpl) runSamples(
	ctx node.Context,
	maxLength int64,
	stamps sampleStamps,
	stringResults [][]string,
) bool {
	for i := range maxLength {
		n.loadSampleParams(i)
		res, err := n.call(ctx.Context)
		if err != nil {
			// A trap ends the cycle. Emitting the samples that already succeeded
			// would make the output depend on where in the series the trap landed.
			ctx.ReportError(errors.Wrapf(
				err,
				"WASM execution failed in node %s at sample %d/%d",
				n.ir.Key,
				i,
				maxLength,
			))
			for j := range n.offsets {
				n.offsets[j] = 0
			}
			for j := range stringResults {
				stringResults[j] = stringResults[j][:0]
			}
			return true
		}
		n.appendResults(res, i, maxLength, stamps.at(i), stringResults)
	}
	return false
}

func (n *nodeImpl) loadSampleParams(i int64) {
	for j := range n.ir.Inputs {
		if !n.streamed(j) {
			continue
		}
		inputLen := n.Input(j).Len()
		idx := int(i % inputLen)
		if !n.stringInputs[j] {
			n.params[j] = valueAt(n.Input(j), idx)
		} else {
			// String channels are variable-length but WASM expects
			// i32 handles. Convert inline — string channels are
			// virtual (length 1), so At(idx) is always O(1).
			data := n.Input(j).At(idx)
			n.params[j] = uint64(n.strings.Create(string(data)))
		}
	}
}

func (n *nodeImpl) appendResults(
	res []result,
	i, maxLength int64,
	ts uint64,
	stringResults [][]string,
) {
	for j, value := range res {
		if !value.Changed {
			continue
		}
		if n.stringOutputs[j] {
			// WASM returned an i32 string handle; materialize it to its actual string
			// value, mirroring loadSampleParams.
			s, ok := n.strings.Get(uint32(value.Value))
			if !ok {
				// An unregistered handle is an Arc compiler/runtime
				// defect, not anything a .arc program can provoke.
				zap.S().DPanicf(
					"node %s output %d returned unregistered string handle %d at sample %d/%d",
					n.ir.Key,
					j,
					value.Value,
					i,
					maxLength,
				)
				continue
			}
			stringResults[j] = append(stringResults[j], s)
		} else {
			setValueAt(*n.Output(j), n.offsets[j], value.Value)
		}
		setValueAt(*n.OutputTime(j), n.offsets[j], ts)
		n.offsets[j]++
	}
}

func (n *nodeImpl) emitOutputs(ctx node.Context, stringResults [][]string) {
	for j := range n.ir.Outputs {
		if n.stringOutputs[j] {
			out := n.Output(j)
			out.Data = telem.NewSeriesV(stringResults[j]...).Data
		} else {
			out := n.Output(j)
			*out = out.Resized(int64(n.offsets[j]))
		}
		outTime := n.OutputTime(j)
		*outTime = outTime.Resized(int64(n.offsets[j]))
		if n.offsets[j] > 0 {
			n.Emit(ctx, j)
		}
	}
}

func (n *nodeImpl) Reset(ctx node.Context) {
	n.State.Reset(ctx)
	if n.nodeKeySetter != nil {
		n.nodeKeySetter.ClearNode(n.ir.Key)
	}
}

func setValueAt(s telem.Series, i int, v uint64) {
	density := s.DataType.Density()
	offset := i * int(density)
	switch density {
	case telem.Bit8:
		s.Data[offset] = byte(v)
	case telem.Bit16:
		telem.ByteOrder.PutUint16(s.Data[offset:offset+2], uint16(v))
	case telem.Bit32:
		telem.ByteOrder.PutUint32(s.Data[offset:offset+4], uint32(v))
	case telem.Bit64:
		telem.ByteOrder.PutUint64(s.Data[offset:offset+8], v)
	case telem.UnknownDensity, telem.Bit128:
		panic("unsupported data density for WASM conversion")
	}
}

// valueAt marshals one sample for the guest call stack. Narrow signed integers
// sign-extend: the guest operates on them as signed i32.
func valueAt(s telem.Series, i int) uint64 {
	data := s.At(i)
	density := s.DataType.Density()
	switch density {
	case telem.Bit8:
		if s.DataType == telem.Int8T {
			return uint64(int8(data[0]))
		}
		return uint64(data[0])
	case telem.Bit16:
		if s.DataType == telem.Int16T {
			return uint64(int16(telem.ByteOrder.Uint16(data)))
		}
		return uint64(telem.ByteOrder.Uint16(data))
	case telem.Bit32:
		return uint64(telem.ByteOrder.Uint32(data))
	case telem.Bit64:
		return telem.ByteOrder.Uint64(data)
	case telem.UnknownDensity, telem.Bit128:
	}
	panic("unsupported data density for WASM conversion")
}

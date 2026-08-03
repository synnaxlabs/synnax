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
	offsets       []int
	initialized   bool
	isEntryNode   bool
	selIdx        int
	clock         telem.MonoClock
	nodeKeySetter NodeKeySetter
	stringInputs  []bool
	chanInputs    []bool
	varInputs     []bool
	stringOutputs []bool
	strings       *stlstrings.ProgramState
}

func (n *nodeImpl) call(ctx context.Context, params ...uint64) ([]result, error) {
	for i := range n.outputValues {
		n.outputValues[i].Changed = false
	}
	results, err := n.fn.Call(ctx, params...)
	if err != nil {
		return nil, err
	}
	if n.memBase == 0 {
		if len(n.outputValues) > 0 {
			n.outputValues[0] = result{Value: results[0], Changed: true}
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

func (*nodeImpl) Init(node.Context) {}

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

	if n.isEntryNode {
		if n.initialized {
			return
		}
		n.initialized = true
	}

	// A $sel-only change re-points without emitting; the value fires on the next input.
	if n.selIdx >= 0 && !n.dataFresh() {
		n.RefreshInputs()
		return
	}

	if !n.RefreshInputs() {
		return
	}

	// A KindChan param holds the key of the channel the body targets. The key
	// is edge-fed and can rebind at runtime, so re-read the latest each pass.
	for i := range n.ir.Inputs {
		if !n.chanInputs[i] || n.ir.Inputs[i].Value != nil {
			continue
		}
		t := n.RefInput(i)
		if t.Len() == 0 {
			return
		}
		n.params[i] = uint64(telem.ValueAt[uint32](t, -1))
	}

	// A var input references a variable's node; re-read the latest each pass.
	for i := range n.ir.Inputs {
		if !n.varInputs[i] {
			continue
		}
		t := n.RefInput(i)
		if t.Len() == 0 {
			return
		}
		if n.stringInputs[i] {
			n.params[i] = uint64(n.strings.Create(string(t.At(-1))))
		} else {
			n.params[i] = valueAt(t, int(t.Len()-1))
		}
	}

	maxLength := int64(0)
	longestInputIdx := -1
	for i := range n.ir.Inputs {
		if n.ir.Inputs[i].Value != nil || n.chanInputs[i] || n.varInputs[i] {
			continue
		}
		dataLen := n.Input(i).Len()
		if dataLen > maxLength {
			maxLength = dataLen
			longestInputIdx = i
		}
	}
	// With no edge-fed inputs, the node executes once over its literal inputs.
	if longestInputIdx == -1 {
		maxLength = 1
	}
	if maxLength <= 0 {
		return
	}
	for j := range n.offsets {
		n.offsets[j] = 0
	}
	// String outputs are variable-density and cannot be resized . Their
	// Data buffer is built once at the end of the loop from accumulated
	// strings. Numeric outputs are pre-sized here so setValueAt can do
	// fixed-stride writes per sample.
	var stringResults [][]string
	for i := range n.ir.Outputs {
		if n.stringOutputs[i] {
			if stringResults == nil {
				stringResults = make([][]string, len(n.ir.Outputs))
			}
			stringResults[i] = make([]string, 0, maxLength)
		} else {
			n.Output(i).Resize(maxLength)
		}
		n.OutputTime(i).Resize(maxLength)
	}
	// Copy alignment and time range from inputs to outputs.
	// Alignments are summed to guarantee uniqueness across different input sources.
	var alignmentSum telem.Alignment
	var timeRange telem.TimeRange
	for i := range n.ir.Inputs {
		if n.ir.Inputs[i].Value != nil || n.chanInputs[i] || n.varInputs[i] {
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
	var longestInputTime telem.Series
	if longestInputIdx >= 0 {
		longestInputTime = n.InputTime(longestInputIdx)
	}
	// Dispatcher drivers alternate; no input's time is honest, so stamp the clock.
	clockStamp := longestInputIdx < 0 || n.selIdx >= 0
	if n.nodeKeySetter != nil {
		n.nodeKeySetter.SetNodeKey(n.ir.Key)
	}
	for i := int64(0); i < maxLength; i++ {
		for j := range n.ir.Inputs {
			if n.ir.Inputs[j].Value != nil || n.chanInputs[j] || n.varInputs[j] {
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
		res, err := n.call(ctx.Context, n.params...)
		if err != nil {
			ctx.ReportError(errors.Wrapf(
				err,
				"WASM execution failed in node %s at sample %d/%d",
				n.ir.Key,
				i,
				maxLength,
			))
			continue
		}
		var ts uint64
		if clockStamp {
			ts = uint64(n.clock.Now())
		} else {
			ts = valueAt(longestInputTime, int(i))
		}
		for j, value := range res {
			if value.Changed {
				if n.stringOutputs[j] {
					// WASM returned an i32 string handle; materialize it
					// to its actual string value, mirroring the input-side
					// conversion above.
					s, ok := n.strings.Get(uint32(value.Value))
					if !ok {
						// An unregistered handle is an Arc compiler/runtime
						// bug, not anything a .arc program can provoke.
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
	}
	for j := range n.ir.Outputs {
		if n.stringOutputs[j] {
			out := n.Output(j)
			out.Data = telem.NewSeriesV[string](stringResults[j]...).Data
		} else {
			n.Output(j).Resize(int64(n.offsets[j]))
		}
		n.OutputTime(j).Resize(int64(n.offsets[j]))
		if n.offsets[j] > 0 {
			ctx.MarkChanged(j)
		}
	}
}

func (n *nodeImpl) Reset() {
	n.State.Reset()
	n.initialized = false
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
	}
}

func valueAt(s telem.Series, i int) uint64 {
	data := s.At(i)
	density := s.DataType.Density()
	switch density {
	case telem.Bit8:
		return uint64(data[0])
	case telem.Bit16:
		return uint64(telem.ByteOrder.Uint16(data))
	case telem.Bit32:
		return uint64(telem.ByteOrder.Uint32(data))
	case telem.Bit64:
		return telem.ByteOrder.Uint64(data)
	default:
		panic("unsupported data density for WASM conversion")
	}
}

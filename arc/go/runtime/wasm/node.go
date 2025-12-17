// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type nodeImpl struct {
	*state.Node
	ir      ir.Node
	wasm    *Function
	inputs  []uint64
	offsets []int
}

func (n *nodeImpl) Init(node.Context) {}

func (n *nodeImpl) Next(ctx node.Context) {
	defer func() {
		if r := recover(); r != nil {
			ctx.ReportError(errors.Newf("WASM trap in node %s: %v", n.ir.Key, r))
		}
	}()

	if !n.RefreshInputs() {
		return
	}

	maxLength := int64(0)
	longestInputIdx := 0
	for i := range n.ir.Inputs {
		dataLen := n.Input(i).Len()
		if dataLen > maxLength {
			maxLength = dataLen
			longestInputIdx = i
		}
	}
	// If no inputs, execute once
	if len(n.ir.Inputs) == 0 {
		maxLength = 1
	}
	if maxLength <= 0 {
		return
	}
	for j := range n.offsets {
		n.offsets[j] = 0
	}
	for i := range n.ir.Outputs {
		n.Output(i).Resize(maxLength)
		n.OutputTime(i).Resize(maxLength)
	}
	// Copy alignment and time range from inputs to outputs.
	// Alignments are summed to guarantee uniqueness across different input sources.
	var alignmentSum telem.Alignment
	var timeRange telem.TimeRange
	for i := range n.ir.Inputs {
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
	if len(n.ir.Inputs) > 0 {
		longestInputTime = n.InputTime(longestInputIdx)
	}
	for i := int64(0); i < maxLength; i++ {
		for j := range n.ir.Inputs {
			inputLen := n.Input(j).Len()
			n.inputs[j] = valueAt(n.Input(j), int(i%inputLen))
		}
		res, err := n.wasm.Call(ctx, n.inputs...)
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
		if len(n.ir.Inputs) > 0 {
			ts = valueAt(longestInputTime, int(i))
		} else {
			ts = uint64(telem.Now())
		}
		for j, value := range res {
			if value.changed {
				setValueAt(*n.Output(j), n.offsets[j], value.value)
				setValueAt(*n.OutputTime(j), n.offsets[j], ts)
				n.offsets[j]++
			}
		}
	}
	for j := range n.ir.Outputs {
		n.Output(j).Resize(int64(n.offsets[j]))
		n.OutputTime(j).Resize(int64(n.offsets[j]))
		if n.offsets[j] > 0 {
			ctx.MarkChanged(n.ir.Outputs[j].Name)
		}
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

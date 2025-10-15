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
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
)

type node struct {
	ir    ir.Node
	wasm  *Function
	edges struct {
		input, output []ir.Edge
	}
	state          *state.State
	inputs         []uint64
	changedOutputs []string
}

func (n *node) Init(context.Context, func(output string)) {}

func (n *node) Next(ctx context.Context, markChanged func(output string)) {
	var maxLength int64
	var maxLengthInput ir.Edge
	for _, o := range n.edges.input {
		if oLen := n.state.Outputs[o.Source].Data.Len(); oLen > maxLength {
			maxLength = oLen
			maxLengthInput = o
		}
	}
	// Get time series from the longest input
	var outputTime telem.Series
	if maxLength > 0 {
		outputTime = n.state.Outputs[maxLengthInput.Source].Time
	}

	for i := range maxLength {
		for inputIdx, o := range n.edges.input {
			n.inputs[inputIdx] = valueAt(n.state.Outputs[o.Source].Data, int(i))
		}
		res, err := n.wasm.Call(ctx, n.inputs...)
		if err != nil {
			panic(err)
		}
		for param, value := range res {
			outputHandle := ir.Handle{Param: param, Node: n.ir.Key}
			outputState := n.state.Outputs[outputHandle]
			setValueAt(outputState.Data, int(i), value)
			// Set time from the longest input
			if outputState.Time.Len() == 0 && maxLength > 0 {
				outputState.Time = outputTime
			}
			n.state.Outputs[outputHandle] = outputState
			markChanged(param)
		}
	}
}

func valueAt(s telem.Series, i int) uint64 {
	if s.DataType.IsVariable() {
		panic("variable density series not supported for WASM conversion")
	}
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

func setValueAt(s telem.Series, i int, v uint64) {
	if s.DataType.IsVariable() {
		panic("variable density series not supported for WASM conversion")
	}
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
	default:
		panic("unsupported data density for WASM conversion")
	}
}

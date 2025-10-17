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
	ir      ir.Node
	state   *state.Node
	wasm    *Function
	inputs  []uint64
	offsets []int
}

func (n *node) Init(context.Context, func(output string)) {}

func (n *node) Next(ctx context.Context, markChanged func(output string)) {
	if !n.state.RefreshInputs() {
		return
	}
	minLength := int64(-1)
	for i := range n.ir.Inputs.Count() {
		dataLen := n.state.Input(i).Len()
		if minLength == -1 || dataLen < minLength {
			minLength = dataLen
		}
	}
	if minLength <= 0 {
		return
	}
	for i := range n.ir.Outputs.Count() {
		n.state.Output(i).Resize(minLength)
	}
	for i := int64(0); i < minLength; i++ {
		for j := range n.ir.Inputs.Count() {
			n.inputs[j] = valueAt(n.state.Input(j), int(i))
		}
		res, err := n.wasm.Call(ctx, n.inputs...)
		if err != nil {
			panic(err)
		}
		for j, value := range res {
			if value.changed {
				setValueAt(*n.state.Output(j), n.offsets[j], value.value)
				n.offsets[j]++
			}
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

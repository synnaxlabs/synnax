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
	"github.com/synnaxlabs/arc/runtime"
	"github.com/synnaxlabs/x/telem"
)

type Node struct {
	Node    ir.Node
	Wasm    *Function
	Inputs  []ir.Edge
	Outputs []ir.Edge
	State   *runtime.State
	Params  []uint64
}

func (n *Node) Next(ctx context.Context, onOutput func(param string)) {
	var maxLength int64
	for _, o := range n.Inputs {
		if oLen := n.State.Outputs[o.Source].Len(); oLen > maxLength {
			maxLength = oLen
		}
	}
	for i := range maxLength {
		for inputIdx, o := range n.Inputs {
			n.Params[inputIdx] = ValueAt(n.State.Outputs[o.Source], int(i))
		}
		res, err := n.Wasm.Call(ctx, n.Params...)
		if err != nil {
		}
		for param, value := range res {
			onOutput(param)
			SetValueAt(n.State.Outputs[ir.Handle{Param: param, Node: n.Node.Key}], int(i), value)
		}
	}
	return
}

// ValueAt reads the value at index i from the series in its native data type and
// converts it to a WASM compatible uint64.
func ValueAt(s telem.Series, i int) uint64 {
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

// SetValueAt takes the Wasm encoded uint64 of the same data type as the series and
// sets it at the index i.
func SetValueAt(s telem.Series, i int, v uint64) {
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

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
	"github.com/synnaxlabs/arc/runtime/align"
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
	aligner        *align.Aligner
	inputs         []uint64
	changedOutputs []string
}

func (n *node) Init(context.Context, func(output string)) {}

func (n *node) Next(ctx context.Context, markChanged func(output string)) {
	// Add all input data to the aligner
	for _, edge := range n.edges.input {
		inputData := n.state.Outputs[edge.Source]
		if inputData.Data.Len() == 0 {
			continue
		}
		if err := n.aligner.Add(edge.Target.Param, inputData.Data, inputData.Time); err != nil {
			panic(err)
		}
	}

	// Get the next aligned operation
	op, ok := n.aligner.Next()
	if !ok {
		return
	}

	// Find the minimum length across all aligned inputs
	var minLength int64 = -1
	for _, alignedInput := range op.Inputs {
		dataLen := alignedInput.Data.Len()
		if minLength == -1 || dataLen < minLength {
			minLength = dataLen
		}
	}

	if minLength <= 0 {
		return
	}

	// Process each sample in the aligned data
	for i := int64(0); i < minLength; i++ {
		// Extract input values for this sample
		for inputIdx, edge := range n.edges.input {
			alignedInput := op.Inputs[edge.Target.Param]
			n.inputs[inputIdx] = valueAt(alignedInput.Data, int(i))
		}

		// Call the WASM function
		res, err := n.wasm.Call(ctx, n.inputs...)
		if err != nil {
			panic(err)
		}

		// Store the results in output series
		for param, value := range res {
			outputHandle := ir.Handle{Param: param, Node: n.ir.Key}
			outputState := n.state.Outputs[outputHandle]

			// Ensure output data is allocated with sufficient capacity
			requiredSize := (int(i) + 1) * int(outputState.Data.DataType.Density())
			if len(outputState.Data.Data) < requiredSize {
				newData := make([]byte, requiredSize)
				copy(newData, outputState.Data.Data)
				outputState.Data.Data = newData
			}

			setValueAt(outputState.Data, int(i), value)

			// Set time series from one of the aligned inputs (use first available)
			if outputState.Time.Len() == 0 {
				for _, alignedInput := range op.Inputs {
					if alignedInput.Time.Len() > 0 {
						outputState.Time = alignedInput.Time
						break
					}
				}
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

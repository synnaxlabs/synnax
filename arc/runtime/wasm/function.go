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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/types"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

type result struct {
	value   uint64
	changed bool
}

type Function struct {
	fn           api.Function
	mem          api.Memory
	outputs      types.Params
	base         uint32
	offsets      []uint32
	outputValues []result
}

func (f *Function) Call(ctx context.Context, params ...uint64) ([]result, error) {
	for i := range f.outputValues {
		f.outputValues[i].changed = false
	}
	results, err := f.fn.Call(ctx, params...)
	if err != nil {
		return nil, err
	}
	if f.base == 0 {
		f.outputValues[0] = result{value: results[0], changed: true}
		return f.outputValues, nil
	}
	dirtyFlags := lo.Must(f.mem.ReadUint64Le(f.base))
	for i := range f.outputs.Keys {
		if (dirtyFlags & (1 << i)) != 0 {
			f.outputValues[i] = result{
				value:   lo.Must(f.mem.ReadUint64Le(f.offsets[i])),
				changed: true,
			}
		}
	}
	return f.outputValues, nil
}

func WrapFunction(
	fn api.Function,
	mem api.Memory,
	outputs types.Params,
	base uint32,
) *Function {
	offsets := make([]uint32, outputs.Count())
	offset := base + 8
	for i, t := range outputs.Values {
		offsets[i] = offset
		offset += sizeOf(t)
	}
	return &Function{
		fn:           fn,
		mem:          mem,
		outputs:      outputs,
		base:         base,
		offsets:      offsets,
		outputValues: make([]result, outputs.Count()),
	}
}

func sizeOf(t types.Type) uint32 {
	switch t.Kind {
	case types.KindU8, types.KindI8:
		return 1
	case types.KindU16, types.KindI16:
		return 2
	case types.KindU32, types.KindI32, types.KindF32:
		return 4
	case types.KindU64, types.KindI64, types.KindF64, types.KindTimeStamp, types.KindTimeSpan:
		return 8
	default:
		zap.S().DPanic("unknown type", zap.String("type", t.String()))
		return 8
	}
}

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/types"
)

// needsConversion checks if type conversion is needed between expression and target type
func needsConversion(from, to types.Type) bool {
	// I64 to I32 conversion
	if _, fromI64 := from.(types.I64); fromI64 {
		if _, toI32 := to.(types.I32); toI32 {
			return true
		}
		if _, toI16 := to.(types.I16); toI16 {
			return true
		}
		if _, toI8 := to.(types.I8); toI8 {
			return true
		}
		if _, toU32 := to.(types.U32); toU32 {
			return true
		}
		if _, toU16 := to.(types.U16); toU16 {
			return true
		}
		if _, toU8 := to.(types.U8); toU8 {
			return true
		}
	}
	// F64 to F32 conversion
	if _, fromF64 := from.(types.F64); fromF64 {
		if _, toF32 := to.(types.F32); toF32 {
			return true
		}
	}
	return false
}

// emitTypeConversion emits the appropriate type conversion instruction
func emitTypeConversion(enc *wasm.Encoder, from, to types.Type) {
	// I64 to smaller integer types
	if _, fromI64 := from.(types.I64); fromI64 {
		switch to.(type) {
		case types.I32, types.I16, types.I8, types.U32, types.U16, types.U8:
			enc.WriteOpcode(wasm.OpI32WrapI64)
		}
	}
	// F64 to F32
	if _, fromF64 := from.(types.F64); fromF64 {
		if _, toF32 := to.(types.F32); toF32 {
			enc.WriteOpcode(wasm.OpF32DemoteF64)
		}
	}
}
// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

func (e *Compiler) compileTypeCast(cast parser.ITypeCastContext) (types.Type, error) {
	targetType := extractType(cast.Type_())
	if targetType == nil {
		return nil, errors.New("unknown cast target type")
	}

	sourceType, err := e.Compile(cast.Expression())
	if err != nil {
		return nil, err
	}

	if err := e.emitCast(sourceType, targetType); err != nil {
		return nil, err
	}

	return targetType, nil
}

func extractType(typeCtx parser.ITypeContext) types.Type {
	if prim := typeCtx.PrimitiveType(); prim != nil {
		if num := prim.NumericType(); num != nil {
			if intType := num.IntegerType(); intType != nil {
				if intType.I8() != nil {
					return types.I8{}
				} else if intType.I16() != nil {
					return types.I16{}
				} else if intType.I32() != nil {
					return types.I32{}
				} else if intType.I64() != nil {
					return types.I64{}
				} else if intType.U8() != nil {
					return types.U8{}
				} else if intType.U16() != nil {
					return types.U16{}
				} else if intType.U32() != nil {
					return types.U32{}
				} else if intType.U64() != nil {
					return types.U64{}
				}
			} else if floatType := num.FloatType(); floatType != nil {
				if floatType.F32() != nil {
					return types.F32{}
				} else if floatType.F64() != nil {
					return types.F64{}
				}
			}
		}
	}
	return nil
}

func (e *Compiler) emitCast(from, to types.Type) error {
	fromWasm := MapType(from)
	toWasm := MapType(to)

	if fromWasm == toWasm {
		return nil
	}

	switch fromWasm {
	case wasm.I32:
		switch toWasm {
		case wasm.I64:
			if types.IsSignedInteger(from) {
				e.encoder.WriteOpcode(wasm.OpI64ExtendI32S)
			} else {
				e.encoder.WriteOpcode(wasm.OpI64ExtendI32U)
			}
		case wasm.F32:
			if types.IsSignedInteger(from) {
				e.encoder.WriteOpcode(wasm.OpF32ConvertI32S)
			} else {
				e.encoder.WriteOpcode(wasm.OpF32ConvertI32U)
			}
		case wasm.F64:
			if types.IsSignedInteger(from) {
				e.encoder.WriteOpcode(wasm.OpF64ConvertI32S)
			} else {
				e.encoder.WriteOpcode(wasm.OpF64ConvertI32U)
			}
		}

	case wasm.I64:
		switch toWasm {
		case wasm.I32:
			e.encoder.WriteOpcode(wasm.OpI32WrapI64)
		case wasm.F32:
			if types.IsSignedInteger(from) {
				e.encoder.WriteOpcode(wasm.OpF32ConvertI64S)
			} else {
				e.encoder.WriteOpcode(wasm.OpF32ConvertI64U)
			}
		case wasm.F64:
			if types.IsSignedInteger(from) {
				e.encoder.WriteOpcode(wasm.OpF64ConvertI64S)
			} else {
				e.encoder.WriteOpcode(wasm.OpF64ConvertI64U)
			}
		}

	case wasm.F32:
		switch toWasm {
		case wasm.I32:
			if types.IsSignedInteger(to) {
				e.encoder.WriteOpcode(wasm.OpI32TruncF32S)
			} else {
				e.encoder.WriteOpcode(wasm.OpI32TruncF32U)
			}
		case wasm.I64:
			if types.IsSignedInteger(to) {
				e.encoder.WriteOpcode(wasm.OpI64TruncF32S)
			} else {
				e.encoder.WriteOpcode(wasm.OpI64TruncF32U)
			}
		case wasm.F64:
			e.encoder.WriteOpcode(wasm.OpF64PromoteF32)
		}

	case wasm.F64:
		switch toWasm {
		case wasm.I32:
			if types.IsSignedInteger(to) {
				e.encoder.WriteOpcode(wasm.OpI32TruncF64S)
			} else {
				e.encoder.WriteOpcode(wasm.OpI32TruncF64U)
			}
		case wasm.I64:
			if types.IsSignedInteger(to) {
				e.encoder.WriteOpcode(wasm.OpI64TruncF64S)
			} else {
				e.encoder.WriteOpcode(wasm.OpI64TruncF64U)
			}
		case wasm.F32:
			e.encoder.WriteOpcode(wasm.OpF32DemoteF64)
		}
	}

	return nil
}

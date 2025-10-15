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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileTypeCast(
	ctx context.Context[parser.ITypeCastContext],
) (types.Type, error) {
	targetType := extractType(ctx.AST.Type_())
	if targetType == nil {
		return nil, errors.New("unknown cast target type")
	}

	sourceType, err := Compile(context.Child(ctx, ctx.AST.Expression()))
	if err != nil {
		return nil, err
	}

	if err := EmitCast(ctx, sourceType, targetType); err != nil {
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
					return types.F64()
				}
			}
		}
	}
	return nil
}

func EmitCast[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from, to types.Type,
) error {
	fromWasm := wasm.ConvertType(from)
	toWasm := wasm.ConvertType(to)

	if fromWasm == toWasm {
		return nil
	}

	switch fromWasm {
	case wasm.I32:
		switch toWasm {
		case wasm.I64:
			if ir.IsSignedInteger(from) {
				ctx.Writer.WriteOpcode(wasm.OpI64ExtendI32S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpI64ExtendI32U)
			}
		case wasm.F32:
			if ir.IsSignedInteger(from) {
				ctx.Writer.WriteOpcode(wasm.OpF32ConvertI32S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpF32ConvertI32U)
			}
		case wasm.F64:
			if ir.IsSignedInteger(from) {
				ctx.Writer.WriteOpcode(wasm.OpF64ConvertI32S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpF64ConvertI32U)
			}
		}

	case wasm.I64:
		switch toWasm {
		case wasm.I32:
			ctx.Writer.WriteOpcode(wasm.OpI32WrapI64)
		case wasm.F32:
			if ir.IsSignedInteger(from) {
				ctx.Writer.WriteOpcode(wasm.OpF32ConvertI64S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpF32ConvertI64U)
			}
		case wasm.F64:
			if ir.IsSignedInteger(from) {
				ctx.Writer.WriteOpcode(wasm.OpF64ConvertI64S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpF64ConvertI64U)
			}
		}

	case wasm.F32:
		switch toWasm {
		case wasm.I32:
			if ir.IsSignedInteger(to) {
				ctx.Writer.WriteOpcode(wasm.OpI32TruncF32S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpI32TruncF32U)
			}
		case wasm.I64:
			if ir.IsSignedInteger(to) {
				ctx.Writer.WriteOpcode(wasm.OpI64TruncF32S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpI64TruncF32U)
			}
		case wasm.F64:
			ctx.Writer.WriteOpcode(wasm.OpF64PromoteF32)
		}

	case wasm.F64:
		switch toWasm {
		case wasm.I32:
			if ir.IsSignedInteger(to) {
				ctx.Writer.WriteOpcode(wasm.OpI32TruncF64S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpI32TruncF64U)
			}
		case wasm.I64:
			if ir.IsSignedInteger(to) {
				ctx.Writer.WriteOpcode(wasm.OpI64TruncF64S)
			} else {
				ctx.Writer.WriteOpcode(wasm.OpI64TruncF64U)
			}
		case wasm.F32:
			ctx.Writer.WriteOpcode(wasm.OpF32DemoteF64)
		}
	}

	return nil
}

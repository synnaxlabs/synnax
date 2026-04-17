// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileTypeCast(
	ctx context.Context[parser.ITypeCastContext],
) (types.Type, error) {
	targetType := extractType(ctx.AST.Type_())
	if !targetType.IsValid() {
		return types.Type{}, errors.New("unknown cast target type")
	}
	// Pass the target type as a hint so literals can be emitted with the correct
	// type directly. Bool is an exception: a literal is never natively bool, so
	// letting the literal take its natural numeric type and then emitting a
	// numeric-to-bool normalization in EmitCast is simpler and keeps the cast
	// matrix centralized.
	hint := targetType
	if targetType.Kind == types.KindBool {
		hint = types.Type{}
	}
	sourceType, err := Compile(context.Child(ctx, ctx.AST.Expression()).WithHint(hint))
	if err != nil {
		return types.Type{}, err
	}
	if err = EmitCast(ctx, sourceType, targetType); err != nil {
		return types.Type{}, err
	}
	return targetType, nil
}

func extractType(typeCtx parser.ITypeContext) types.Type {
	if prim := typeCtx.PrimitiveType(); prim != nil {
		if prim.BOOL() != nil {
			return types.Bool()
		}
		if num := prim.NumericType(); num != nil {
			if intType := num.IntegerType(); intType != nil {
				if intType.I8() != nil {
					return types.I8()
				} else if intType.I16() != nil {
					return types.I16()
				} else if intType.I32() != nil {
					return types.I32()
				} else if intType.I64() != nil {
					return types.I64()
				} else if intType.U8() != nil {
					return types.U8()
				} else if intType.U16() != nil {
					return types.U16()
				} else if intType.U32() != nil {
					return types.U32()
				} else if intType.U64() != nil {
					return types.U64()
				}
			} else if floatType := num.FloatType(); floatType != nil {
				if floatType.F32() != nil {
					return types.F32()
				} else if floatType.F64() != nil {
					return types.F64()
				}
			}
		}
	}
	return types.Type{}
}

func EmitCast[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from, to types.Type,
) error {
	if from.Kind == to.Kind {
		return nil
	}
	if to.Kind == types.KindBool {
		return emitCastToBool(ctx, from)
	}
	if from.Kind == types.KindBool {
		emitCastFromBool(ctx, to)
		return nil
	}
	var (
		fromWasm = wasm.ConvertType(from)
		toWasm   = wasm.ConvertType(to)
	)
	if fromWasm == toWasm {
		return nil
	}
	var (
		opCode       wasm.Opcode
		fromIsSigned = from.IsSignedInteger()
		toIsSigned   = to.IsSignedInteger()
	)
	switch fromWasm {
	case wasm.I32:
		switch toWasm {
		case wasm.I64:
			opCode = lo.Ternary(fromIsSigned, wasm.OpI64ExtendI32S, wasm.OpI64ExtendI32U)
		case wasm.F32:
			opCode = lo.Ternary(fromIsSigned, wasm.OpF32ConvertI32S, wasm.OpF32ConvertI32U)
		case wasm.F64:
			opCode = lo.Ternary(fromIsSigned, wasm.OpF64ConvertI32S, wasm.OpF64ConvertI32U)
		}
	case wasm.I64:
		switch toWasm {
		case wasm.I32:
			opCode = wasm.OpI32WrapI64
		case wasm.F32:
			opCode = lo.Ternary(fromIsSigned, wasm.OpF32ConvertI64S, wasm.OpF32ConvertI64U)
		case wasm.F64:
			opCode = lo.Ternary(fromIsSigned, wasm.OpF64ConvertI64S, wasm.OpF64ConvertI64U)
		}
	case wasm.F32:
		switch toWasm {
		case wasm.I32:
			opCode = lo.Ternary(toIsSigned, wasm.OpI32TruncF32S, wasm.OpI32TruncF32U)
		case wasm.I64:
			opCode = lo.Ternary(toIsSigned, wasm.OpI64TruncF32S, wasm.OpI64TruncF32U)
		case wasm.F64:
			opCode = wasm.OpF64PromoteF32
		}
	case wasm.F64:
		switch toWasm {
		case wasm.I32:
			opCode = lo.Ternary(toIsSigned, wasm.OpI32TruncF64S, wasm.OpI32TruncF64U)
		case wasm.I64:
			opCode = lo.Ternary(toIsSigned, wasm.OpI64TruncF64S, wasm.OpI64TruncF64U)
		case wasm.F32:
			opCode = wasm.OpF32DemoteF64
		}
	}
	ctx.Writer.WriteOpcode(opCode)
	return nil
}

// emitCastToBool normalizes any numeric value on the stack to canonical bool:
// nonzero maps to 1, zero maps to 0. The result is always an i32.
func emitCastToBool[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from types.Type,
) error {
	switch wasm.ConvertType(from) {
	case wasm.I32:
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz)
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz)
	case wasm.I64:
		ctx.Writer.WriteOpcode(wasm.OpI64Eqz)
		ctx.Writer.WriteOpcode(wasm.OpI32Eqz)
	case wasm.F32:
		ctx.Writer.WriteF32Const(0)
		ctx.Writer.WriteOpcode(wasm.OpF32Ne)
	case wasm.F64:
		ctx.Writer.WriteF64Const(0)
		ctx.Writer.WriteOpcode(wasm.OpF64Ne)
	default:
		return errors.Newf("cannot cast %s to bool", from)
	}
	return nil
}

// emitCastFromBool widens a canonical bool (i32 0 or 1 on the stack) to a
// numeric type. Since bool bytes are already canonical, no normalization is
// needed, only any width/representation conversion.
func emitCastFromBool[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	to types.Type,
) {
	switch wasm.ConvertType(to) {
	case wasm.I32:
	case wasm.I64:
		ctx.Writer.WriteOpcode(wasm.OpI64ExtendI32U)
	case wasm.F32:
		ctx.Writer.WriteOpcode(wasm.OpF32ConvertI32U)
	case wasm.F64:
		ctx.Writer.WriteOpcode(wasm.OpF64ConvertI32U)
	}
}

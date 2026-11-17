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
	"math"

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
	sourceType, err := Compile(
		ctx.Child(ctx.AST.Expression()).WithHint(targetType),
	)
	if err != nil {
		return types.Type{}, err
	}
	if err = EmitCast(ctx, sourceType, targetType); err != nil {
		return types.Type{}, err
	}
	return targetType, nil
}

func extractType(typeCtx parser.ITypeContext) types.Type {
	prim := typeCtx.PrimitiveType()
	if prim == nil {
		return types.Type{}
	}
	if prim.STR() != nil {
		return types.String()
	}
	num := prim.NumericType()
	if num == nil {
		return types.Type{}
	}
	if intType := num.IntegerType(); intType != nil {
		return extractIntegerType(intType)
	}
	if floatType := num.FloatType(); floatType != nil {
		return extractFloatType(floatType)
	}
	return types.Type{}
}

func extractIntegerType(intType parser.IIntegerTypeContext) types.Type {
	switch {
	case intType.I8() != nil:
		return types.I8()
	case intType.I16() != nil:
		return types.I16()
	case intType.I32() != nil:
		return types.I32()
	case intType.I64() != nil:
		return types.I64()
	case intType.U8() != nil:
		return types.U8()
	case intType.U16() != nil:
		return types.U16()
	case intType.U32() != nil:
		return types.U32()
	case intType.U64() != nil:
		return types.U64()
	default:
		return types.Type{}
	}
}

func extractFloatType(floatType parser.IFloatTypeContext) types.Type {
	switch {
	case floatType.F32() != nil:
		return types.F32()
	case floatType.F64() != nil:
		return types.F64()
	default:
		return types.Type{}
	}
}

func EmitCast[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from, to types.Type,
) error {
	if to.Kind == types.KindString {
		return emitStringCast(ctx, from)
	}
	var (
		fromWasm = wasm.ConvertType(from)
		toWasm   = wasm.ConvertType(to)
	)
	if fromWasm == toWasm {
		if !fitsIn(from, to) {
			ctx.Writer.WriteNarrow(to)
		}
		return nil
	}
	opCode := convertOpcode(
		fromWasm, toWasm, from.IsSignedInteger(), to.IsSignedInteger(),
	)
	if lower, upper, ok := narrowBounds(to); ok && from.IsFloat() {
		emitClamp(ctx, fromWasm, lower, upper)
		ctx.Writer.WriteOpcode(opCode)
		return nil
	}
	ctx.Writer.WriteOpcode(opCode)
	ctx.Writer.WriteNarrow(to)
	return nil
}

func emitStringCast[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from types.Type,
) error {
	if from.Kind == types.KindString {
		return nil
	}
	if from.Kind == types.KindBool {
		return ctx.Resolver.EmitFixedImportCall(
			ctx,
			ctx.Writer,
			ctx.WriterID,
			ctx.Scope,
			"strings",
			"from_bool",
		)
	}
	return ctx.Resolver.EmitNumericToString(
		ctx,
		ctx.Writer,
		ctx.WriterID,
		ctx.Scope,
		from,
	)
}

// convertOpcode returns the opcode that converts between two different WASM value
// types.
func convertOpcode(from, to wasm.ValueType, fromSigned, toSigned bool) wasm.Opcode {
	switch from {
	case wasm.I32:
		switch to {
		case wasm.I64:
			return lo.Ternary(fromSigned, wasm.OpI64ExtendI32S, wasm.OpI64ExtendI32U)
		case wasm.F32:
			return lo.Ternary(fromSigned, wasm.OpF32ConvertI32S, wasm.OpF32ConvertI32U)
		case wasm.F64:
			return lo.Ternary(fromSigned, wasm.OpF64ConvertI32S, wasm.OpF64ConvertI32U)
		}
	case wasm.I64:
		switch to {
		case wasm.I32:
			return wasm.OpI32WrapI64
		case wasm.F32:
			return lo.Ternary(fromSigned, wasm.OpF32ConvertI64S, wasm.OpF32ConvertI64U)
		case wasm.F64:
			return lo.Ternary(fromSigned, wasm.OpF64ConvertI64S, wasm.OpF64ConvertI64U)
		}
	case wasm.F32:
		switch to {
		case wasm.I32:
			return lo.Ternary(toSigned, wasm.OpI32TruncF32S, wasm.OpI32TruncF32U)
		case wasm.I64:
			return lo.Ternary(toSigned, wasm.OpI64TruncF32S, wasm.OpI64TruncF32U)
		case wasm.F64:
			return wasm.OpF64PromoteF32
		}
	case wasm.F64:
		switch to {
		case wasm.I32:
			return lo.Ternary(toSigned, wasm.OpI32TruncF64S, wasm.OpI32TruncF64U)
		case wasm.I64:
			return lo.Ternary(toSigned, wasm.OpI64TruncF64S, wasm.OpI64TruncF64U)
		case wasm.F32:
			return wasm.OpF32DemoteF64
		}
	}
	return 0
}

// emitClamp clamps the float on the stack to [lower, upper].
func emitClamp[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	fromWasm wasm.ValueType,
	lower, upper float64,
) {
	if fromWasm == wasm.F32 {
		ctx.Writer.WriteF32Const(float32(upper))
		ctx.Writer.WriteOpcode(wasm.OpF32Min)
		ctx.Writer.WriteF32Const(float32(lower))
		ctx.Writer.WriteOpcode(wasm.OpF32Max)
		return
	}
	ctx.Writer.WriteF64Const(upper)
	ctx.Writer.WriteOpcode(wasm.OpF64Min)
	ctx.Writer.WriteF64Const(lower)
	ctx.Writer.WriteOpcode(wasm.OpF64Max)
}

// narrowBounds returns the range of an 8-bit or 16-bit integer type.
func narrowBounds(t types.Type) (lower, upper float64, ok bool) {
	switch t.Kind {
	case types.KindU8:
		return 0, math.MaxUint8, true
	case types.KindU16:
		return 0, math.MaxUint16, true
	case types.KindI8:
		return math.MinInt8, math.MaxInt8, true
	case types.KindI16:
		return math.MinInt16, math.MaxInt16, true
	default:
		return 0, 0, false
	}
}

// fitsIn reports whether to holds every value of from.
func fitsIn(from, to types.Type) bool {
	switch from.Kind {
	case to.Kind, types.KindBool:
		return true
	case types.KindU8:
		return to.Kind == types.KindU16 || to.Kind == types.KindI16
	case types.KindI8:
		return to.Kind == types.KindI16
	default:
		return false
	}
}

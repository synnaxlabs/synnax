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
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileTypeCast(
	ctx context.Context[parser.ITypeCastContext],
) (types.Type, error) {
	targetType, err := atypes.InferFromTypeContext(ctx.AST.Type_())
	if err != nil || !targetType.IsValid() {
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

// EmitCast converts the value of type from on the stack to the type and unit of to.
func EmitCast[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	from, to types.Type,
) error {
	if !types.UnitsConvertible(from.Unit, to.Unit) {
		return emitKindCast(ctx, from, to)
	}
	// The wider of the two types holds the value while it converts.
	wide := to
	if width(from) > width(to) {
		wide = from
	}
	wide.Unit = from.Unit
	if err := emitKindCast(ctx, from, wide); err != nil {
		return err
	}
	if err := emitUnitScale(ctx, wide, to.Unit); err != nil {
		return err
	}
	return emitKindCast(ctx, wide, to)
}

// width orders the numeric types by the range of values that they hold.
func width(t types.Type) int {
	switch {
	case t.Kind == types.KindF64:
		return 3
	case t.Kind == types.KindF32:
		return 2
	case t.Is64Bit():
		return 1
	default:
		return 0
	}
}

// emitUnitScale converts the value or series of type t on the stack to the unit to.
// An integer truncates toward zero.
func emitUnitScale[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	t types.Type,
	to *types.Unit,
) error {
	elem := t.Unwrap()
	factor, divide, exact := types.UnitScale(elem, to)
	op := "*"
	if divide {
		op = "/"
	}
	if t.Kind == types.KindSeries {
		if !exact {
			return errors.Newf("cannot convert a series of %s to %s", elem, to.Name)
		}
		writeConst(ctx.Writer, elem, factor)
		return ctx.Resolver.EmitSeriesArithmetic(
			ctx.Writer, ctx.WriterID, op, elem, true,
		)
	}
	if exact {
		writeConst(ctx.Writer, elem, factor)
		return ctx.Writer.WriteBinaryOpInferred(op, elem)
	}
	if err := emitKindCast(ctx, elem, types.F64()); err != nil {
		return err
	}
	ctx.Writer.WriteF64Const(factor)
	ctx.Writer.WriteBinaryOp(wasm.OpF64Mul)
	return emitKindCast(ctx, types.F64(), elem)
}

// writeConst pushes v as a constant of the numeric type t.
func writeConst(w *wasm.Writer, t types.Type, v float64) {
	switch {
	case t.Kind == types.KindF64:
		w.WriteF64Const(v)
	case t.Kind == types.KindF32:
		w.WriteF32Const(float32(v))
	case t.Is64Bit():
		w.WriteI64Const(int64(v))
	default:
		w.WriteI32Const(int32(v))
	}
}

func emitKindCast[ASTNode antlr.ParserRuleContext](
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
	if from.IsFloat() && (toWasm == wasm.I32 || toWasm == wasm.I64) {
		emitFloatToInt(ctx, fromWasm, toWasm, to)
		return nil
	}
	ctx.Writer.WriteOpcode(convertOpcode(fromWasm, toWasm, from.IsSignedInteger()))
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
// types, except for float to integer conversions.
func convertOpcode(from, to wasm.ValueType, fromSigned bool) wasm.Opcode {
	switch from {
	case wasm.I32:
		switch to {
		case wasm.I64:
			return lo.Ternary(fromSigned, wasm.OpI64ExtendI32S, wasm.OpI64ExtendI32U)
		case wasm.F32:
			return lo.Ternary(fromSigned, wasm.OpF32ConvertI32S, wasm.OpF32ConvertI32U)
		case wasm.F64:
			return lo.Ternary(fromSigned, wasm.OpF64ConvertI32S, wasm.OpF64ConvertI32U)
		case wasm.I32:
			panic("cast between equal wasm types")
		}
	case wasm.I64:
		switch to {
		case wasm.I32:
			return wasm.OpI32WrapI64
		case wasm.F32:
			return lo.Ternary(fromSigned, wasm.OpF32ConvertI64S, wasm.OpF32ConvertI64U)
		case wasm.F64:
			return lo.Ternary(fromSigned, wasm.OpF64ConvertI64S, wasm.OpF64ConvertI64U)
		case wasm.I64:
			panic("cast between equal wasm types")
		}
	case wasm.F32:
		switch to {
		case wasm.F64:
			return wasm.OpF64PromoteF32
		case wasm.I32, wasm.I64:
			panic("float to integer casts saturate through emitFloatToInt")
		case wasm.F32:
			panic("cast between equal wasm types")
		}
	case wasm.F64:
		switch to {
		case wasm.F32:
			return wasm.OpF32DemoteF64
		case wasm.I32, wasm.I64:
			panic("float to integer casts saturate through emitFloatToInt")
		case wasm.F64:
			panic("cast between equal wasm types")
		}
	}
	return 0
}

// emitFloatToInt truncates the float on the stack toward zero and saturates it at the
// bounds of to.
func emitFloatToInt[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	fromWasm, toWasm wasm.ValueType,
	to types.Type,
) {
	if lower, upper, ok := narrowBounds(to); ok {
		emitClamp(ctx, fromWasm, lower, upper)
	}
	var op wasm.TruncSatOpcode
	switch {
	case toWasm == wasm.I32 && fromWasm == wasm.F32:
		op = lo.Ternary(
			to.IsSignedInteger(),
			wasm.OpI32TruncSatF32S,
			wasm.OpI32TruncSatF32U,
		)
	case toWasm == wasm.I32:
		op = lo.Ternary(
			to.IsSignedInteger(),
			wasm.OpI32TruncSatF64S,
			wasm.OpI32TruncSatF64U,
		)
	case fromWasm == wasm.F32:
		op = lo.Ternary(
			to.IsSignedInteger(),
			wasm.OpI64TruncSatF32S,
			wasm.OpI64TruncSatF32U,
		)
	default:
		op = lo.Ternary(
			to.IsSignedInteger(),
			wasm.OpI64TruncSatF64S,
			wasm.OpI64TruncSatF64U,
		)
	}
	ctx.Writer.WriteTruncSat(op)
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
	case types.KindInvalid, types.KindU32, types.KindU64, types.KindI32, types.KindI64,
		types.KindF32, types.KindF64, types.KindString, types.KindChan,
		types.KindSeries, types.KindVariable, types.KindNumericConstant,
		types.KindIntegerConstant, types.KindFloatConstant,
		types.KindExactIntegerFloatConstant, types.KindFunction, types.KindSequence,
		types.KindStage, types.KindVarRef, types.KindBool:
	}
	return 0, 0, false
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
	case types.KindInvalid, types.KindU16, types.KindU32, types.KindU64, types.KindI16,
		types.KindI32, types.KindI64, types.KindF32, types.KindF64, types.KindString,
		types.KindChan, types.KindSeries, types.KindVariable, types.KindNumericConstant,
		types.KindIntegerConstant, types.KindFloatConstant,
		types.KindExactIntegerFloatConstant, types.KindFunction, types.KindSequence,
		types.KindStage, types.KindVarRef:
	}
	return false
}

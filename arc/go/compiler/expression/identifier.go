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
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileIdentifier[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	head, tail string,
) (types.Type, error) {
	scope, err := ctx.Scope.Resolve(ctx, head, symbol.IncludeInternal)
	if err != nil {
		return types.Type{}, err
	}
	name := head
	if tail != "" {
		scope, err = scope.Resolve(ctx, tail)
		if err != nil {
			return types.Type{}, err
		}
		name = head + "." + tail
	}
	chanRef := ctx.Hint.Kind == types.KindChan
	switch scope.Kind {
	case symbol.KindVariable, symbol.KindInput, symbol.KindLoopVariable:
		// Inherited channel read/write (cpu := chan): no local here, read by key.
		if scope.Type.Kind == types.KindChan && !sameFunction(ctx.Scope, scope) {
			ctx.Writer.WriteI32Const(int32(channelKeyOf(scope)))
			if chanRef {
				return scope.Type, nil
			}
			emitChannelRead(ctx, scope.Type)
			return scope.Type.Unwrap(), nil
		}
		// A flow-level variable has no local in this unit: fold its seed.
		if scope.Kind == symbol.KindVariable && !sameFunction(ctx.Scope, scope) {
			if !scope.Reassigned && scope.DefaultValue != nil {
				return emitSeedConst(ctx, scope)
			}
			return types.Type{}, errors.Newf(
				"cannot read reassigned variable '%s' inside an expression yet", name,
			)
		}
		ctx.Writer.WriteLocalGet(scope.ID)
		if scope.Type.Kind == types.KindChan {
			if chanRef {
				return scope.Type, nil
			}
			emitChannelRead(ctx, scope.Type)
			return scope.Type.Unwrap(), nil
		}
		return scope.Type, nil
	case symbol.KindStatefulVariable:
		if !scope.Reassigned && scope.DefaultValue != nil {
			return emitSeedConst(ctx, scope)
		}
		emitStatefulLoad(ctx, scope.ID, scope.Type)
		return scope.Type, nil
	case symbol.KindChannel:
		ctx.Writer.WriteI32Const(int32(scope.ID))
		if chanRef {
			return scope.Type, nil
		}
		emitChannelRead(ctx, scope.Type)
		return scope.Type.Unwrap(), nil
	default:
		return types.Type{}, errors.Newf("unsupported symbol kind: %v for '%s'", scope.Kind, name)
	}
}

// channelKeyOf returns the channel key sym refers to: its SourceID when sym is
// an alias bound to another channel, otherwise its own ID.
func channelKeyOf(sym *symbol.Symbol) int {
	if sym.SourceID != nil {
		return *sym.SourceID
	}
	return sym.ID
}

// sameFunction reports whether a and b compile into the same unit (same enclosing function, or both module-level).
func sameFunction(a, b *symbol.Symbol) bool {
	af, _ := a.ClosestAncestorOfKind(symbol.KindFunction)
	bf, _ := b.ClosestAncestorOfKind(symbol.KindFunction)
	return af == bf
}

// emitSeedConst compiles a read of a never-reassigned seeded variable as its
// compile-time seed value.
func emitSeedConst[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	sym *symbol.Symbol,
) (types.Type, error) {
	t := sym.Type
	v := sym.DefaultValue
	switch t.Kind {
	case types.KindString:
		s, ok := v.(string)
		if !ok {
			return types.Type{}, errors.Newf("seed for '%s' is not a string: %T", sym.Name, v)
		}
		emitLiteralSegment(ctx, s)
	case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16,
		types.KindU32:
		n, ok := seedInt(v)
		if !ok {
			return types.Type{}, errors.Newf("seed for '%s' is not an integer: %T", sym.Name, v)
		}
		ctx.Writer.WriteI32Const(int32(n))
	case types.KindI64, types.KindU64:
		n, ok := seedInt(v)
		if !ok {
			return types.Type{}, errors.Newf("seed for '%s' is not an integer: %T", sym.Name, v)
		}
		ctx.Writer.WriteI64Const(n)
	case types.KindF32:
		f, ok := seedFloat(v)
		if !ok {
			return types.Type{}, errors.Newf("seed for '%s' is not numeric: %T", sym.Name, v)
		}
		ctx.Writer.WriteF32Const(float32(f))
	case types.KindF64:
		f, ok := seedFloat(v)
		if !ok {
			return types.Type{}, errors.Newf("seed for '%s' is not numeric: %T", sym.Name, v)
		}
		ctx.Writer.WriteF64Const(f)
	default:
		return types.Type{}, errors.Newf("cannot fold seed of type %s for '%s'", t, sym.Name)
	}
	return t, nil
}

// seedInt widens any integer seed value to int64.
func seedInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	}
	return 0, false
}

// seedFloat widens any numeric seed value to float64.
func seedFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	if n, ok := seedInt(v); ok {
		return float64(n), true
	}
	return 0, false
}

func emitStatefulLoad[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	idx int,
	t types.Type,
) {
	ctx.Writer.WriteI32Const(int32(idx))
	emitZeroValue(ctx, t)
	if t.Kind == types.KindSeries {
		ctx.Resolver.EmitStateLoadSeries(ctx.Writer, ctx.WriterID, *t.Elem)
	} else {
		ctx.Resolver.EmitStateLoad(ctx.Writer, ctx.WriterID, t.Unwrap())
	}
}

func emitZeroValue[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	t types.Type,
) {
	switch t.Kind {
	case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16, types.KindU32:
		ctx.Writer.WriteI32Const(0)
	case types.KindI64, types.KindU64:
		ctx.Writer.WriteI64Const(0)
	case types.KindF32:
		ctx.Writer.WriteF32Const(0.0)
	case types.KindF64:
		ctx.Writer.WriteF64Const(0.0)
	case types.KindString:
		ctx.Writer.WriteI32Const(0)
	default:
		ctx.Writer.WriteI32Const(0)
	}
}

func emitChannelRead[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	t types.Type,
) {
	ctx.Resolver.EmitChannelRead(ctx.Writer, ctx.WriterID, t)
}

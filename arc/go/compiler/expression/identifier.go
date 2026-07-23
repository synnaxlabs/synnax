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
	"github.com/synnaxlabs/x/query"
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
		// A never-reassigned flow variable collapses to its initial value;
		// reassigned reads are lifted into params before compilation gets here.
		if scope.Kind == symbol.KindVariable && !sameFunction(ctx.Scope, scope) {
			if scope.DefaultValue != nil {
				return castAndEmitConst(ctx, scope)
			}
			// SY-4474: Enable const-expression initializers for variables
			return types.Type{}, errors.Newf(
				"cannot read '%s': its initializer is not a compile-time constant", name,
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
		// Reassigned is only tracked at flow level; a never-written $= there is
		// its initial value. Func-local statefuls always load their cell.
		if _, fnErr := scope.ClosestAncestorOfKind(symbol.KindFunction); errors.Is(fnErr, query.ErrNotFound) &&
			!scope.Reassigned && scope.DefaultValue != nil {
			return castAndEmitConst(ctx, scope)
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

// castAndEmitConst compiles a read of a never-reassigned variable by casting
// its initial value to the variable's type and emitting it as a constant.
func castAndEmitConst[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	sym *symbol.Symbol,
) (types.Type, error) {
	t, v := sym.Type, sym.DefaultValue
	var cast any
	switch t.Kind {
	case types.KindString:
		if s, ok := v.(string); ok {
			cast = s
		}
	case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16,
		types.KindU32:
		if n, ok := asInt64(v); ok {
			cast = int32(n)
		}
	case types.KindI64:
		if n, ok := asInt64(v); ok {
			cast = n
		}
	case types.KindU64:
		if n, ok := asInt64(v); ok {
			cast = uint64(n)
		}
	case types.KindF32:
		if f, ok := asFloat64(v); ok {
			cast = float32(f)
		}
	case types.KindF64:
		if f, ok := asFloat64(v); ok {
			cast = f
		}
	}
	if cast == nil {
		return types.Type{}, errors.Newf(
			"cannot fold a %s constant from %T for '%s'", t, v, sym.Name,
		)
	}
	if err := emitLiteralValue(ctx, t, cast); err != nil {
		return types.Type{}, err
	}
	return t, nil
}

// asInt64 widens any integer value to int64.
func asInt64(v any) (int64, bool) {
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

// asFloat64 widens any numeric value to float64.
func asFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	if n, ok := asInt64(v); ok {
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

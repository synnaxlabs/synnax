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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

func compileIdentifier[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	n parser.Name,
) (types.Type, error) {
	name := n.String()
	var scope *symbol.Symbol
	// A quoted channel name never binds a variable, a lifted input included.
	if !n.Channel {
		scope = liftedInput(ctx.Scope, name)
	}
	if scope == nil {
		var err error
		scope, err = ctx.Scope.ResolveName(ctx, n, symbol.IncludeInternal)
		if err != nil {
			return types.Type{}, err
		}
	}
	switch scope.Kind {
	case symbol.KindVariable, symbol.KindInput, symbol.KindLoopVariable:
		return compileVariableRead(ctx, scope, name)
	case symbol.KindStatefulVariable:
		// Reassigned is only tracked at flow level; a never-written $= there is
		// its initial value. Func-local statefuls always load their cell.
		if _, fnErr := scope.ClosestAncestorOfKind(
			symbol.KindFunction,
		); errors.Is(fnErr, query.ErrNotFound) &&
			!scope.Reassigned &&
			scope.DefaultValue != nil {
			return castAndEmitConst(ctx, scope)
		}
		emitStatefulLoad(ctx, scope.ID, scope.Type)
		return scope.Type, nil
	case symbol.KindChannel:
		return compileChannelRead(ctx, scope), nil
	case symbol.KindConstant:
		return compileFloatConstant(ctx, scope)
	case symbol.KindFunction, symbol.KindBlock, symbol.KindOutput, symbol.KindSequence,
		symbol.KindStage, symbol.KindLoop, symbol.KindModule, symbol.KindModuleAlias,
		symbol.KindAmbient:
	}
	return types.Type{}, errors.Newf(
		"unsupported symbol kind: %v for '%s'",
		scope.Kind,
		name,
	)
}

func compileVariableRead[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	scope *symbol.Symbol,
	name string,
) (types.Type, error) {
	if scope.Type.Kind == types.KindChan {
		return compileChannelRead(ctx, scope), nil
	}
	// A never-reassigned literal flow variable collapses to its initial value;
	// reassigned and computed reads are lifted into params before compilation.
	if scope.Kind == symbol.KindVariable && !SameFunction(ctx.Scope, scope) {
		if scope.DefaultValue != nil {
			return castAndEmitConst(ctx, scope)
		}
		return types.Type{}, errors.Newf(
			"cannot read '%s': its initializer is not a compile-time constant",
			name,
		)
	}
	ctx.Writer.WriteLocalGet(scope.ID)
	return scope.Type, nil
}

// compileChannelRead emits the channel key of scope, and its latest value unless the
// hint asks for the channel itself.
func compileChannelRead[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	scope *symbol.Symbol,
) types.Type {
	EmitChannelKey(ctx, scope)
	if ctx.Hint.Kind == types.KindChan {
		return scope.Type
	}
	emitChannelRead(ctx, scope.Type)
	return scope.Type.Unwrap()
}

// liftedInput returns the input of the enclosing function that carries a read of the
// module member name, nil for a bare name or a read that was not lifted.
func liftedInput(scope *symbol.Symbol, name string) *symbol.Symbol {
	if !strings.Contains(name, ".") {
		return nil
	}
	fn, err := scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		return nil
	}
	if in := fn.FindChild(name); in != nil && in.Kind == symbol.KindInput {
		return in
	}
	return nil
}

// EmitChannelKey pushes the key of the channel that sym names. A param or variable of
// the function being compiled holds the key in a local. A channel, or an alias declared
// outside the function, has a constant key.
func EmitChannelKey[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	sym *symbol.Symbol,
) {
	if sym.Kind != symbol.KindChannel && SameFunction(ctx.Scope, sym) {
		ctx.Writer.WriteLocalGet(sym.ID)
		return
	}
	ctx.Writer.WriteI32Const(int32(channelKeyOf(sym)))
}

// channelKeyOf returns the channel key sym refers to: its SourceID when sym is
// an alias bound to another channel, otherwise its own ID.
func channelKeyOf(sym *symbol.Symbol) int {
	if sym.SourceID != nil {
		return *sym.SourceID
	}
	return sym.ID
}

// SameFunction reports whether a and b compile into the same unit (same enclosing
// function, or both module-level).
func SameFunction(a, b *symbol.Symbol) bool {
	af, aErr := a.ClosestAncestorOfKind(symbol.KindFunction)
	bf, bErr := b.ClosestAncestorOfKind(symbol.KindFunction)
	// A module-level symbol has no enclosing function.
	if aErr != nil || bErr != nil {
		return aErr != nil && bErr != nil
	}
	return af == bf
}

// castAndEmitConst compiles a read of a never-reassigned variable by casting
// its initial value to the variable's type and emitting it as a constant.
func castAndEmitConst[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	sym *symbol.Symbol,
) (types.Type, error) {
	t, v := sym.Type, types.DecodeValue(sym.DefaultValue, sym.Type)
	var cast any
	switch t.Kind {
	case types.KindString:
		if s, ok := v.(string); ok {
			cast = s
		}
	case types.KindBool:
		if b, ok := v.(bool); ok {
			cast = b
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
	case types.KindInvalid, types.KindChan, types.KindSeries, types.KindVariable,
		types.KindNumericConstant, types.KindIntegerConstant, types.KindFloatConstant,
		types.KindExactIntegerFloatConstant, types.KindFunction, types.KindSequence,
		types.KindStage, types.KindVarRef:
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
	case telem.TimeSpan:
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
	case types.KindI8,
		types.KindI16,
		types.KindI32,
		types.KindU8,
		types.KindU16,
		types.KindU32:
		ctx.Writer.WriteI32Const(0)
	case types.KindI64, types.KindU64:
		ctx.Writer.WriteI64Const(0)
	case types.KindF32:
		ctx.Writer.WriteF32Const(0.0)
	case types.KindF64:
		ctx.Writer.WriteF64Const(0.0)
	case types.KindInvalid, types.KindString, types.KindChan, types.KindSeries,
		types.KindVariable, types.KindNumericConstant, types.KindIntegerConstant,
		types.KindFloatConstant, types.KindExactIntegerFloatConstant,
		types.KindFunction, types.KindSequence, types.KindStage, types.KindVarRef,
		types.KindBool:
		ctx.Writer.WriteI32Const(0)
	default:
		panic("unknown type kind " + t.Kind.String())
	}
}

func emitChannelRead[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	t types.Type,
) {
	ctx.Resolver.EmitChannelRead(ctx.Writer, ctx.WriterID, t)
}

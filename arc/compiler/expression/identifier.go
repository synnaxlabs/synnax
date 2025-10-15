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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileIdentifier[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	name string,
) (types.Type, error) {
	scope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return types.Type{}, errors.Wrapf(err, "identifier '%s' not found", name)
	}
	switch scope.Kind {
	case symbol.KindVariable, symbol.KindInput, symbol.KindConfig:
		ctx.Writer.WriteLocalGet(scope.ID)
		return scope.Type, nil
	case symbol.KindStatefulVariable:
		if err = emitStatefulLoad(ctx, scope.ID, scope.Type); err != nil {
			return types.Type{}, err
		}
		return scope.Type, nil
	case symbol.KindChannel:
		ctx.Writer.WriteI32Const(int32(scope.ID))
		if err = emitChannelRead(ctx, scope.Type); err != nil {
			return types.Type{}, err
		}
		return *scope.Type.ValueType, nil
	default:
		return types.Type{}, errors.Newf("unsupported symbol kind: %v for '%s'", scope.Kind, name)
	}
}

func emitStatefulLoad[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	idx int,
	t types.Type,
) error {
	// TODO: This needs to be implemented
	ctx.Writer.WriteI32Const(0)
	ctx.Writer.WriteI32Const(int32(idx))
	importIdx, err := ctx.Imports.GetStateLoad(t)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

func emitChannelRead[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	t types.Type,
) error {
	importIdx, err := ctx.Imports.GetChannelRead(t)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

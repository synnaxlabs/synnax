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
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// compileIdentifier compiles variable references
func compileIdentifier(ctx *core.Context, name string) (types.Type, error) {
	// First, look up the symbol in the symbol table to get its type
	scope, err := ctx.Scope.Resolve(name)
	if err != nil {
		return nil, errors.Wrapf(err, "identifier '%s' not found", name)
	}

	switch scope.Kind {
	case symbol.KindVariable, symbol.KindParam, symbol.KindConfigParam:
		ctx.Writer.WriteLocalGet(scope.ID)
		return scope.Type, nil
	case symbol.KindStatefulVariable:
		if err = emitStatefulLoad(ctx, scope.ID, scope.Type); err != nil {
			return nil, err
		}
		return scope.Type, nil
	case symbol.KindChannel:
		ctx.Writer.WriteLocalGet(scope.ID)
		if err = emitChannelRead(ctx, scope.Type); err != nil {
			return nil, err
		}
		return scope.Type, nil
	default:
		return nil, errors.Newf("unsupported symbol kind: %v for '%s'", scope.Kind, name)
	}
}

// emitStatefulLoad emits code to load a stateful variable
func emitStatefulLoad(ctx *core.Context, idx int, t types.Type) error {
	// Push task ID (0 for now - would be provided at runtime)
	ctx.Writer.WriteI32Const(0)
	// Push variable key
	ctx.Writer.WriteI32Const(int32(idx))
	// Call appropriate state load function based on type
	importIdx, err := ctx.Imports.GetStateLoad(t)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

// emitChannelRead emits code for non-blocking channel read
func emitChannelRead(ctx *core.Context, t types.Type) error {
	// Stack has channel ID
	// Call appropriate channel read function based on type
	importIdx, err := ctx.Imports.GetChannelRead(t)
	if err != nil {
		return err
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

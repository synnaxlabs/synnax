// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package constant implements semantic analysis for Arc global constant declarations.
//
// Top-level variable declarations (using :=) are compile-time constants. Values are
// inlined directly into WASM bytecode at each reference site with no runtime overhead.
// Only literal values are allowed - no expressions or runtime evaluation.
//
// Example:
//
//	MAX_PRESSURE f64 := 500.0   // explicit type
//	SAMPLE_RATE := 1000         // inferred i64
//	TIMEOUT := 100ms            // unit literals supported
package constant

import (
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// CollectDeclarations registers all global constant declarations in the symbol table.
// This follows the same two-pass pattern as function.CollectDeclarations and
// sequence.CollectDeclarations.
func CollectDeclarations(ctx acontext.Context[parser.IProgramContext]) {
	for _, item := range ctx.AST.AllTopLevelItem() {
		if globalConst := item.GlobalConstant(); globalConst != nil {
			collectConstant(ctx, globalConst)
		}
	}
}

func collectConstant(
	ctx acontext.Context[parser.IProgramContext],
	globalConst parser.IGlobalConstantContext,
) {
	name := globalConst.IDENTIFIER().GetText()
	lit := globalConst.Literal()

	var targetType types.Type
	if typeCtx := globalConst.Type_(); typeCtx != nil {
		var err error
		targetType, err = atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, typeCtx))
			return
		}
	}

	parsed, err := literal.Parse(lit, targetType)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, lit))
		return
	}

	if !targetType.IsValid() {
		targetType = parsed.Type
	}

	if _, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name:         name,
		Kind:         symbol.KindGlobalConstant,
		Type:         targetType,
		DefaultValue: parsed.Value,
		AST:          globalConst,
	}); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, globalConst))
	}
}

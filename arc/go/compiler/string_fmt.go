// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"github.com/antlr4-go/antlr/v4"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// compileStringFmtSynthetic emits a zero-param WASM body returning the
// formatted string handle for an analyzer-synthesized backtick Function.
func compileStringFmtSynthetic(
	rootCtx ccontext.Context[antlr.ParserRuleContext],
	fn ir.Function,
) (compiledFunction, error) {
	segments, err := fmtstring.Parse(fn.Body.Raw)
	if err != nil {
		return compiledFunction{}, errors.Wrapf(err, "parse synthetic format for %q", fn.Key)
	}
	ctx := rootCtx.WithNewWriter()
	funcT := wasm.FunctionType{
		Results: []wasm.ValueType{wasm.ConvertType(types.String())},
	}
	typeIdx := ctx.Module.AddType(funcT)
	if _, err := expression.EmitFmtSegments(ctx, segments); err != nil {
		return compiledFunction{}, errors.Wrapf(err, "emit synthetic format for %q", fn.Key)
	}
	return compiledFunction{
		scopeName: fn.Key,
		typeIdx:   typeIdx,
		writer:    ctx.Writer,
	}, nil
}

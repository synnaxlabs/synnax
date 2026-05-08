// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow

import (
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/parser"
)

// stringFmtFormatConfigParam mirrors formatConfigParam in
// arc/go/stl/strings/string.go.
const stringFmtFormatConfigParam = "format"

// analyzeStringFmtFlowFormat analyzes placeholders in the `format` config of
// a flow-form `string.fmt{...}` invocation, registering references against
// the flow scope. No-op if `format` is absent from configBlock.
func analyzeStringFmtFlowFormat(
	ctx context.Context[parser.IFunctionContext],
	configBlock parser.IConfigValuesContext,
) {
	formatExpr := stringFmtFormatExpression(configBlock)
	if formatExpr == nil {
		return
	}
	expression.AnalyzeStringFmtSegments(ctx, formatExpr)
}

// stringFmtFormatExpression returns the IExpressionContext bound to the
// `format` config slot, or nil if not present. Both named (`format = "..."`)
// and anonymous (`{ "..." }`, single positional) forms are handled.
func stringFmtFormatExpression(
	configBlock parser.IConfigValuesContext,
) parser.IExpressionContext {
	if configBlock == nil {
		return nil
	}
	if named := configBlock.NamedConfigValues(); named != nil {
		for _, configVal := range named.AllNamedConfigValue() {
			if configVal.IDENTIFIER().GetText() != stringFmtFormatConfigParam {
				continue
			}
			return configVal.Expression()
		}
		return nil
	}
	if anon := configBlock.AnonymousConfigValues(); anon != nil {
		exprs := anon.AllExpression()
		if len(exprs) == 0 {
			return nil
		}
		return exprs[0]
	}
	return nil
}

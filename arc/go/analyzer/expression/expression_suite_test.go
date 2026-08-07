// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Analyzer Expression Suite")
}

func buildExpressionRoot(extras []symbol.Symbol) *symbol.Symbol {
	root := symbol.NewRoot(nil, stl.NewSymbols())
	for i := range extras {
		s := extras[i]
		root.Parent.AddChild(&s)
	}
	return root
}

func expectSuccess(specCtx context.Context, code string, extras []symbol.Symbol) {
	ast := MustSucceed(parser.Parse(code))
	ctx := acontext.NewRoot(specCtx, ast, buildExpressionRoot(extras))
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
}

func expectFailure(
	specCtx context.Context,
	code string,
	extras []symbol.Symbol,
	expectedMsg string,
) {
	ast := MustSucceed(parser.Parse(code))
	ctx := acontext.NewRoot(specCtx, ast, buildExpressionRoot(extras))
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeFalse())
	Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(expectedMsg))
}

var _ = ShouldNotLeakGoroutinesPerSpec()

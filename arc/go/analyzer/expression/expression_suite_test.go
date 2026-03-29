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
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

func TestExpression(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expression Analyzer Suite")
}

func expectSuccess(specCtx context.Context, code string, resolver symbol.Resolver) {
	ast := MustSucceed(parser.Parse(code))
	ctx := acontext.CreateRoot(specCtx, ast, resolver)
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
}

func expectFailure(specCtx context.Context, code string, resolver symbol.Resolver, expectedMsg string) {
	ast := MustSucceed(parser.Parse(code))
	ctx := acontext.CreateRoot(specCtx, ast, resolver)
	analyzer.AnalyzeProgram(ctx)
	Expect(ctx.Diagnostics.Ok()).To(BeFalse())
	Expect((*ctx.Diagnostics)[0].Message).To(ContainSubstring(expectedMsg))
}

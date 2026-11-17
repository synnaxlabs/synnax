// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Diagnostic kinds", func() {
	DescribeTable("Should report the code of the kind",
		func(ctx SpecContext, source string, kind diagnostics.Kind) {
			aCtx := context.NewRoot(
				ctx,
				MustSucceed(parser.Parse(source)),
				NewRoot(nil),
			)
			analyzer.AnalyzeProgram(aCtx)
			reported := lo.Map(
				*aCtx.Diagnostics,
				func(d diagnostics.Diagnostic, _ int) diagnostics.ErrorCode { return d.Code },
			)
			Expect(reported).To(ContainElement(kind.Code), aCtx.Diagnostics.String())
		},
		Entry("a value of the wrong type",
			"func f() {\n x i32 := 1\n y str := x\n}", codes.TypeMismatch),
		Entry("an operator on a type it does not support",
			"func f() {\n x := \"a\" - \"b\"\n}", codes.InvalidOperand),
		Entry("a cast that Arc does not convert",
			"func f() {\n x := i32(\"a\")\n}", codes.InvalidCast),
		Entry("a literal with no exact value",
			"func f(x i32 = 1.5) {\n}", codes.InvalidLiteral),
		Entry("a literal that does not fit its type",
			"func f(x i8 = 300) {\n}", codes.ConstantOverflow),
		Entry("units with different dimensions",
			"func f() {\n x := 5s + 3m\n}", codes.DimensionMismatch),
	)
})

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
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Diagnostic kinds", func() {
	DescribeTable("Should report the code of the kind",
		func(ctx SpecContext, source string, kind diagnostics.Kind) {
			aCtx := context.NewRoot(
				ctx,
				MustSucceed(parser.Parse(source)),
				NewRoot(nil, symbol.Symbol{
					Name: "sensor",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   1,
				}),
			)
			analyzer.AnalyzeProgram(aCtx)
			reported := lo.Map(
				*aCtx.Diagnostics,
				func(d diagnostics.Diagnostic, _ int) diagnostics.ErrorCode { return d.Code },
			)
			Expect(reported).To(ContainElement(kind.Code), aCtx.Diagnostics.String())
			for _, d := range *aCtx.Diagnostics {
				if d.Code == kind.Code {
					Expect(d.Message).To(Or(
						Equal(kind.Message),
						HavePrefix(kind.Message+": "),
					), "a message starts with the headline of its kind")
				}
			}
		},
		Entry("a value of the wrong type",
			"func f() {\n x i32 := 1\n y str := x\n}", codes.TypeMismatch),
		Entry("an operator on a type it does not support",
			"func f() {\n x := \"a\" - \"b\"\n}", codes.InvalidOperand),
		Entry("a cast that Arc does not convert",
			"func f() {\n x := i32(\"a\")\n}", codes.InvalidCast),
		Entry("a function that reads a variable with a computed initializer",
			"k := 1 + 2\nfunc f() i64 {\n return k\n}", codes.NotConstant),
		Entry(
			"two routing tables in one flow",
			"func split() {v f64} (a f64, b f64) {\n a = v\n b = v\n}\n"+
				"{v: sensor} -> split() -> {a: sensor, b: sensor}",
			codes.RoutingTable,
		),
		Entry("a flow call of a variable",
			"x := 1\nsensor -> x()", codes.NotCallable),
		Entry(
			"a call of a function with more than one named output",
			"func split(v f64) (a f64, b f64) {\n a = v\n b = v\n}\n"+
				"func f() {\n x := split(1.0)\n}",
			codes.NotCallable,
		),
		Entry("an operator other than += on a string",
			"func f() {\n b := \"a\"\n b -= \"c\"\n}", codes.InvalidOperand),
		Entry("an operator other than += on a string series",
			"func f() {\n s series str := [\"a\"]\n s -= [\"b\"]\n}",
			codes.InvalidOperand),
		Entry("an operator other than += on a string series element",
			"func f() {\n s series str := [\"a\"]\n s[0] -= \"b\"\n}",
			codes.InvalidOperand),
		Entry("a return with no value in a function that returns one",
			"func f() i64 {\n return\n}", codes.ReturnValue),
		Entry("a literal with no exact value",
			"func f(x i32 = 1.5) {\n}", codes.InvalidLiteral),
		Entry("a literal that does not fit its type",
			"func f(x i8 = 300) {\n}", codes.ConstantOverflow),
		Entry("units with different dimensions",
			"func f() {\n x := 5s + 3m\n}", codes.DimensionMismatch),
		Entry("a call of a variable",
			"func f() {\n x := 1\n y := x()\n}", codes.NotCallable),
		Entry(
			"a call with no argument for a parameter",
			"func g(a i32) i32 {\n return a\n}\nfunc f() {\n y := g()\n}",
			codes.FuncArgMissing,
		),
		Entry(
			"a call with too many arguments",
			"func g(a i32) i32 {\n return a\n}\nfunc f() {\n y := g(1, 2)\n}",
			codes.FuncArgExtra,
		),
		Entry("a required parameter after an optional one",
			"func f(a i32 = 1, b i32) {\n}", codes.ParamOrder),
		Entry("a value returned from a function with no return type",
			"func f() {\n return 1\n}", codes.ReturnValue),
		Entry("a break outside a loop",
			"func f() {\n break\n}", codes.BreakOutsideLoop),
		Entry("a stateful variable at the top level",
			"s $= 1", codes.StatefulVariable),
		Entry("an assignment to a loop variable",
			"func f() {\n for i := range(3) {\n i = 1\n }\n}", codes.AssignmentTarget),
		Entry("two functions that call each other",
			"func a() {\n b()\n}\nfunc b() {\n a()\n}", codes.CircularCall),
		Entry("a named output that no path sets",
			"func f() (out i32) {\n}", codes.OutputNeverAssigned),
		Entry("an import of a module that does not exist",
			"import nope", codes.ImportFailed),
		Entry("an import after a declaration",
			"func f() {\n}\nimport time", codes.ImportOrder),
		Entry("two imports of one module",
			"import math\nimport math", codes.DuplicateImport),
		Entry("an export in a program",
			"export func f() {\n}", codes.ExportOutsideModule),
		Entry("a name that no scope declares",
			"func f() {\n x := y\n}", codes.SymbolUndefined),
		Entry("a name declared two times",
			"func f() {\n x := 1\n x := 2\n}", codes.SymbolRedefined),
	)
})

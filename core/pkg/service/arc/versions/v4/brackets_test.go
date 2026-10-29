// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"math"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("SwapBrackets", func() {
	// Headers and bare calls are not whole programs, so only the text is checked.
	DescribeTable("Should rewrite the RFC rows byte for byte",
		func(old, want string) {
			Expect(v4.SwapBrackets(old)).To(Equal(want))
		},
		Entry("inputs and one trigger",
			"func scale{factor f64} (value f64) f64",
			"func scale(factor f64) {value f64} f64"),
		Entry("triggers only",
			"func add(x f64, y f64) f64",
			"func add() {x f64, y f64} f64"),
		Entry("trigger and multiple outputs",
			"func split(value f64) (high f64, low f64)",
			"func split() {value f64} (high f64, low f64)"),
		Entry("inputs, trigger and multiple outputs",
			"func band{limit f64} (value f64) (high f64, low f64)",
			"func band(limit f64) {value f64} (high f64, low f64)"),
		Entry("inputs and no trigger",
			"func log{ch chan f64} ()",
			"func log(ch chan f64) {}"),
		Entry("no inputs and no trigger", "func tick()", "func tick() {}"),
		Entry("named call in a flow",
			"sensor -> scale{factor=2.0} -> out",
			"sensor -> scale(factor=2.0) -> out"),
		Entry("empty call before a routing table",
			"sensor -> split{} -> {high: 1 -> hi, low: 1 -> lo}",
			"sensor -> split() -> {high: 1 -> hi, low: 1 -> lo}"),
		Entry("dotted call", "time.wait{2s}", "time.wait(2s)"),
	)

	DescribeTable("Should rewrite a program that then parses",
		func(old, want string) {
			got := v4.SwapBrackets(old)
			Expect(got).To(Equal(want))
			MustSucceed(parser.Parse(got))
		},
		Entry("declaration with no spacing",
			"func add(x i32,y i32)i32{return x+y}",
			"func add() {x i32,y i32}i32{return x+y}"),
		Entry("declaration with an input block and no spacing",
			"func threshold{limit f64}(value f64)u8{return u8(0)}",
			"func threshold(limit f64){value f64}u8{return u8(0)}"),
		Entry("declaration with a gap before the input block",
			"func average {} (first chan f64) chan f64 {\n    return first\n}",
			"func average () {first chan f64} chan f64 {\n    return first\n}"),
		Entry("declaration with a gap before the trigger parens",
			"func tick (n u8) {}",
			"func tick() {n u8} {}"),
		Entry("input default",
			"func foo{gain f64 = 1.0}() {}",
			"func foo(gain f64 = 1.0){} {}"),
		Entry("named single output",
			"func f(x f64) result f64 { return x }",
			"func f() {x f64} result f64 { return x }"),
		Entry("multi-line input block with a trailing comment",
			`func dog{
    cat chan f32, // blue green
} () u8 {
    return 12
}`,
			`func dog(
    cat chan f32, // blue green
) {} u8 {
    return 12
}`),
		Entry("braces in a function body are not calls",
			`func clamp{hi f64} (v f64) f64 {
    if v > hi {
        return hi
    }
    for i := v {
        v = v - 1.0
    }
    return v
}`,
			`func clamp(hi f64) {v f64} f64 {
    if v > hi {
        return hi
    }
    for i := v {
        v = v - 1.0
    }
    return v
}`),
		Entry("two declarations and a flow",
			`func a(x f64) f64 { return x }
func b{k f64} (x f64) f64 { return x * k }
sensor -> b{k=2.0} -> out`,
			`func a() {x f64} f64 { return x }
func b(k f64) {x f64} f64 { return x * k }
sensor -> b(k=2.0) -> out`),
		Entry("stage and sequence bodies keep their braces",
			`sequence main {
    stage pressurize {
        1 -> valve
        wait{500ms} => wait
    }
    stage wait {
        time.wait {250ms}
        0 -> valve
    }
}
start => main`,
			`sequence main {
    stage pressurize {
        1 -> valve
        wait(500ms) => wait
    }
    stage wait {
        time.wait(250ms)
        0 -> valve
    }
}
start => main`),
		Entry("routing table and inline stage keep their braces",
			`sequence s {
    stage a {
        flag -> select{} -> {
            true: wait{1s} => b,
            false: stage {
                0 -> x
            }
        }
    }
    stage b {}
}`,
			`sequence s {
    stage a {
        flag -> select() -> {
            true: wait(1s) => b,
            false: stage {
                0 -> x
            }
        }
    }
    stage b {}
}`),
		Entry("dotted call whose last segment is a keyword",
			"sensor -> stable.for{duration=1s} -> out",
			"sensor -> stable.for(duration=1s) -> out"),
		Entry("expression call is unchanged",
			"len(ch) -> out",
			"len(ch) -> out"),
		Entry("import and authority blocks are unchanged",
			`import (
    time
    control
)
authority (
    200
    valve 100
)
time.interval{1s} -> valve`,
			`import (
    time
    control
)
authority (
    200
    valve 100
)
time.interval(1s) -> valve`),
		Entry("braces in comments and strings are unchanged",
			"// wait{2s} is the old form\n"+
				"sensor -> emit{msg=\"{x}\"} -> out /* name{ */",
			"// wait{2s} is the old form\n"+
				"sensor -> emit(msg=\"{x}\") -> out /* name{ */"),
		Entry("line comment between a call's name and its brace",
			"sensor -> scale // factor\n{factor=2.0} -> out",
			"sensor -> scale(factor=2.0) // factor\n -> out"),
		Entry("block comment between a call's name and its brace",
			"sensor -> scale /* factor */ {factor=2.0} -> out",
			"sensor -> scale(factor=2.0) /* factor */  -> out"),
		Entry("multi-byte runes before a call",
			"// température — ok\nsensor -> scale{factor=2.0} -> out",
			"// température — ok\nsensor -> scale(factor=2.0) -> out"),
		Entry("empty", "", ""),
	)

	DescribeTable("Should leave a construct it cannot close unchanged",
		func(src string) {
			Expect(v4.SwapBrackets(src)).To(Equal(src))
		},
		Entry("call with no closing brace", "sensor -> wait{2s -> out"),
		Entry("declaration with no trigger parens", "func f{x f64}"),
		Entry("declaration with no closing trigger paren", "func f(x f64"),
		Entry("declaration with no name", "func"),
		Entry("declaration with only a name", "func f"),
	)

	// A document mid-edit can hold many open brackets. The rewrite must stay linear in
	// the token count, so a broken program cannot stall the upgrade. Doubling the input
	// doubles a linear rewrite's time and quadruples a quadratic one on any runner.
	DescribeTable("Should stay linear on brackets that never close",
		func(unit, want string, n int) {
			Expect(v4.SwapBrackets(strings.Repeat(unit, n))).
				To(Equal(strings.Repeat(want, n)))
			Expect(growth(unit, n)).To(BeNumerically("<", 3))
		},
		Entry("open braces", "{", "{", 25_000),
		// Literals lex fast, so they lengthen a scan past each header at little cost.
		Entry("declarations with no body",
			"func f(0,0,0,0,0,0,0,0,0,0) ", "func f() {0,0,0,0,0,0,0,0,0,0} ", 2_000),
	)
})

// growth returns how many times longer SwapBrackets takes on 2n copies of unit than on
// n. Each size keeps its fastest of three runs, since a busy runner only adds time.
func growth(unit string, n int) float64 {
	small, large := strings.Repeat(unit, n), strings.Repeat(unit, 2*n)
	fastSmall, fastLarge := time.Duration(math.MaxInt64), time.Duration(math.MaxInt64)
	for range 3 {
		fastSmall = min(fastSmall, elapsed(small))
		fastLarge = min(fastLarge, elapsed(large))
	}
	return float64(fastLarge) / float64(fastSmall)
}

func elapsed(src string) time.Duration {
	start := time.Now()
	v4.SwapBrackets(src)
	return time.Since(start)
}

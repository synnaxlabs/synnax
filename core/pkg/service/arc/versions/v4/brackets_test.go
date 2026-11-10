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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	symboltestutil "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
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
		Entry("expression call that starts a flow",
			"len(ch) -> out",
			"(len(ch)) -> out"),
		Entry("expression calls in a routing table and a stage",
			`sequence s {
    stage a {
        flag -> select{} -> {true: math.abs(x) -> y, false: 0 -> y}
        len(ch) => b
    }
    stage b {}
}`,
			`sequence s {
    stage a {
        flag -> select() -> {true: (math.abs(x)) -> y, false: 0 -> y},
        (len(ch)) => b
    }
    stage b {}
}`),
		Entry("expression call after a declaration that follows a flow",
			"a -> b\nlimit := 1.0\nlen(ch) -> out",
			"a -> b\nlimit := 1.0\n(len(ch)) -> out"),
		Entry("expression call inside an expression is unchanged",
			"x := 1 + len(ch)\nsensor -> scale{factor=len(ch)} -> out",
			"x := 1 + len(ch)\nsensor -> scale(factor=len(ch)) -> out"),
		Entry("expression call inside an index is unchanged",
			"x := arr[len(ch)]", "x := arr[len(ch)]"),
		Entry("name apart from its paren is unchanged",
			"sensor -> filter (50.0) -> out\nsensor -> filter\n(50.0) -> out",
			"sensor -> filter (50.0) -> out\nsensor -> filter\n(50.0) -> out"),
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

	It("Should rewrite a function call in a flow to a program that compiles",
		func(ctx SpecContext) {
			got := v4.SwapBrackets(`func median3(a f64, b f64, c f64) f64 {
    return a
}
median3(pt_1, pt_2, pt_3) -> tank_pt`)
			Expect(got).To(Equal(`func median3() {a f64, b f64, c f64} f64 {
    return a
}
(median3(pt_1, pt_2, pt_3)) -> tank_pt`))
			channels := []symbol.Symbol{{
				Name: "tank_pt",
				Kind: symbol.KindChannel,
				Type: types.WriteChan(types.F64()),
				ID:   4,
			}}
			for i, name := range []string{"pt_1", "pt_2", "pt_3"} {
				channels = append(channels, symbol.Symbol{
					Name: name,
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   i + 1,
				})
			}
			root := symboltestutil.NewRoot(nil, channels...)
			parsed := MustSucceed(text.Parse(text.Text{Raw: got}))
			ir, diags := text.Analyze(ctx, parsed, root)
			Expect(diags.Ok()).To(BeTrue(), diags.String())
			MustSucceed(text.Compile(ctx, ir))
		})

	DescribeTable("Should keep a wrapped call apart from the stage item before it",
		func(old, want string) {
			got := v4.SwapBrackets(old)
			Expect(got).To(Equal(want))
			prog := MustSucceed(parser.Parse(got))
			stage := prog.TopLevelItem(0).SequenceDeclaration().SequenceItem(0).
				StageDeclaration()
			Expect(stage.StageBody().AllStageItem()).To(HaveLen(2))
		},
		Entry(
			"after a declaration",
			"sequence s {\n  stage a {\n    n := 0\n    reset(n)\n  }\n}",
			"sequence s {\n  stage a {\n    n := 0,\n    (reset(n))\n  }\n}",
		),
		Entry(
			"after a flow that ends in an expression",
			"sequence s {\n  stage a {\n    a -> b + 1\n    len(ch) -> out\n  }\n}",
			"sequence s {\n  stage a {\n    a -> b + 1,\n    (len(ch)) -> out\n  }\n}",
		),
	)

	DescribeTable("Should keep a wrapped call apart from the top-level item before it",
		func(old, want string) {
			got := v4.SwapBrackets(old)
			Expect(got).To(Equal(want))
			prog := MustSucceed(parser.Parse(got))
			Expect(prog.AllTopLevelItem()).To(HaveLen(2))
		},
		Entry("after a declaration",
			"limit := 100.0\nmedian3(pt_1, pt_2) -> tank_pt",
			"limit := 100.0\n(median3(pt_1, pt_2)) -> tank_pt"),
		Entry("after an assignment",
			"count = 0\nlen(ch) -> out",
			"count = 0\n(len(ch)) -> out"),
		Entry("after a flow that ends in an expression",
			"a -> b + 1\nlen(ch) -> out",
			"a -> b + 1\nlen (ch) -> out"),
		Entry("after a flow that ends in a channel",
			"a -> b\nmath.abs(x) -> out",
			"a -> b\nmath.abs (x) -> out"),
		Entry("after a flow that ends in a stage",
			"a => stage {\n  0 -> b\n} -> b + 1\nlen(ch) -> out",
			"a => stage {\n  0 -> b\n} -> b + 1\nlen (ch) -> out"),
	)

	It("Should split a call from a top-level flow before it with the same graph",
		func(ctx SpecContext) {
			got := v4.SwapBrackets("a -> b + 1\nlen(ch) -> out")
			Expect(got).To(Equal("a -> b + 1\nlen (ch) -> out"))
			// An unused declaration ends the flow without adding to the graph.
			want := analyzeAndCompile(
				ctx,
				"a -> b + 1\nfunc split() {}\n(len(ch)) -> out",
			)
			gotIR := analyzeAndCompile(ctx, got)
			Expect(gotIR.Nodes).To(Equal(want.Nodes))
			Expect(gotIR.Edges).To(Equal(want.Edges))
			Expect(gotIR.Edges).To(HaveLen(3))
		})

	DescribeTable("Should lex names as a Core with dashed names did",
		func(old, want string) {
			cfg := parser.Config{AllowDashedNames: true}
			got := v4.SwapBrackets(old, cfg)
			Expect(got).To(Equal(want))
			MustSucceed(parser.Parse(got, cfg))
		},
		Entry("dashed declaration",
			"func my-fn{k f64}(x f64) f64 {\n    if x > k {\n        return 1.0\n    }"+
				"\n    return 0.0\n}",
			"func my-fn(k f64){x f64} f64 {\n    if x > k {\n        return 1.0\n    }"+
				"\n    return 0.0\n}"),
		Entry("dashed stage name",
			"sequence s {\n  stage pre-press {\n    a -> b{x=1}\n  }\n}",
			"sequence s {\n  stage pre-press {\n    a -> b(x=1)\n  }\n}"),
		Entry("dashed call",
			"tank-pt -> my-fn{k=2.0} -> out",
			"tank-pt -> my-fn(k=2.0) -> out"),
		Entry("dashed expression call", "my-fn(x) -> out", "(my-fn(x)) -> out"),
		Entry("spaced subtraction",
			"func f(a f64, b f64) f64 {\n    return a - b\n}",
			"func f() {a f64, b f64} f64 {\n    return a - b\n}"),
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
	// the token count, so a broken program cannot stall the upgrade. Growing the input
	// 32 times grows a linear rewrite's time about 32 times and a quadratic one's by
	// hundreds, so the bound holds against a noisy runner.
	DescribeTable("Should stay linear on brackets that never close",
		func(unit, want string, n int) {
			Expect(v4.SwapBrackets(strings.Repeat(unit, n))).
				To(Equal(strings.Repeat(want, n)))
			Expect(growth(unit, n)).To(BeNumerically("<", 3*growthFactor))
		},
		Entry("open braces", "{", "{", 250),
		// Literals lex fast, so they lengthen a scan past each header at little cost.
		Entry("declarations with no body",
			"func f(0,0,0,0,0,0,0,0,0,0) ", "func f() {0,0,0,0,0,0,0,0,0,0} ", 150),
	)
})

func analyzeAndCompile(ctx SpecContext, src string) ir.IR {
	GinkgoHelper()
	root := symboltestutil.NewRoot(nil,
		symbol.Symbol{
			Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 1,
		},
		symbol.Symbol{
			Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2,
		},
		symbol.Symbol{
			Name: "ch",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.String()),
			ID:   3,
		},
		symbol.Symbol{
			Name: "out",
			Kind: symbol.KindChannel,
			Type: types.WriteChan(types.I64()),
			ID:   4,
		},
	)
	parsed := MustSucceed(text.Parse(text.Text{Raw: src}))
	program, diags := text.Analyze(ctx, parsed, root)
	Expect(diags.Ok()).To(BeTrue(), diags.String())
	MustSucceed(text.Compile(ctx, program))
	return program
}

const growthFactor = 32

// measured is the least time that perCall measures over. It is many times the 16 ms
// tick of the Windows clock, so one tick changes the result little.
const measured = 100 * time.Millisecond

// growth returns how many times longer SwapBrackets takes on growthFactor*n copies of
// unit than on n.
func growth(unit string, n int) float64 {
	small, large := strings.Repeat(unit, n), strings.Repeat(unit, growthFactor*n)
	return float64(perCall(large)) / float64(perCall(small))
}

// perCall returns the time of one SwapBrackets call on src, averaged over calls that
// take at least measured in all. It keeps the fastest of three averages, since a busy
// runner only adds time.
func perCall(src string) time.Duration {
	fastest := time.Duration(math.MaxInt64)
	for range 3 {
		var calls time.Duration
		start := time.Now()
		for ; time.Since(start) < measured; calls++ {
			v4.SwapBrackets(src)
		}
		fastest = min(fastest, time.Since(start)/calls)
	}
	return fastest
}

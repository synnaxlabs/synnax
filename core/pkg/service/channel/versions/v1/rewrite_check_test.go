// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/authority"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	symtestutil "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Rewriter check", func() {
	// Each channel of renamed held its key before the upgrade, which renamed it to the
	// key with "_2". The channel "log" holds strings and keeps its name.
	renamed := map[string]telem.DataType{
		"Metrics":  telem.Float64T,
		"sensor":   telem.Float64T,
		"now":      telem.Float64T,
		"duration": telem.Float64T,
		"s":        telem.Float64T,
		"psi":      telem.Float64T,
		"factor":   telem.Float64T,
		"time":     telem.Float64T,
		"flag":     telem.Uint8T,
		"out":      telem.Float64T,
	}
	var (
		w v1.Rewriter
		// upgraded resolves each channel name as the Core does after the upgrade.
		upgraded symtestutil.StaticResolver
	)
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		channels := map[v1.Key]v1.TextChannel{
			100: {Name: "log", DataType: telem.StringT},
		}
		upgraded = make(symtestutil.StaticResolver, 0, len(renamed)+1)
		upgraded = append(upgraded, symbol.Symbol{
			Name: "log",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.String()),
			ID:   100,
		})
		names := make([]string, 0, len(renamed))
		for name := range renamed {
			names = append(names, name)
		}
		slices.Sort(names)
		for i, name := range names {
			key := v1.Key(i + 1)
			dt := renamed[name]
			channels[key] = v1.TextChannel{Name: name + "_2", DataType: dt}
			upgraded = append(upgraded, symbol.Symbol{
				Name: name + "_2",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.FromTelem(dt)),
				ID:   int(key),
			})
			Expect(resource.StageRename(
				ctx,
				db,
				resource.ID{Type: resource.TypeChannel, Key: key.String()},
				name,
				resource.ReasonClash,
			)).To(Succeed())
		}
		w = v1.Rewriter{
			Renames: MustSucceed(v1.TextRename(db, channels)),
			Ambient: stl.NewSymbols(),
		}
	})
	// errorsOf returns the error diagnostics of the live analyzer on src, a calculation
	// or a program, with the channel names that r resolves.
	errorsOf := func(
		ctx context.Context,
		src v1.Source,
		r symbol.Resolver,
	) []diagnostics.Diagnostic {
		GinkgoHelper()
		root := symbol.NewRoot(r, stl.NewSymbols())
		if src.Form == v1.FormCalculation {
			block := MustSucceed(parser.ParseBlock("{"+src.Text+"}", w.Config))
			aCtx := acontext.NewRoot(ctx, block, root).WithConfig(w.Config)
			statement.AnalyzeFunctionBody(aCtx)
			return aCtx.Diagnostics.Errors()
		}
		t := MustSucceed(text.Parse(text.Text{Raw: src.Text}, w.Config))
		aCtx := acontext.NewRoot(ctx, t.AST, root).WithConfig(w.Config)
		analyzer.AnalyzeProgram(aCtx)
		return aCtx.Diagnostics.Errors()
	}
	keptLines := func(res v1.Result) []int {
		var lines []int
		for _, r := range res.References {
			if r.Kept {
				lines = append(lines, r.Line)
			}
		}
		return lines
	}
	calculation := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormCalculation}
	}
	program := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormProgram}
	}
	// graphErrorsOf returns the error diagnostics of the live graph analyzer, which
	// binds the functions of a graph as a v0.62 Core did, on a graph of fns.
	graphErrorsOf := func(
		ctx context.Context,
		fns ir.Functions,
		r symbol.Resolver,
	) []diagnostics.Diagnostic {
		GinkgoHelper()
		root := symbol.NewRoot(r, stl.NewSymbols())
		symbol.AutoImportModules(root)
		g := MustSucceed(graph.Parse(graph.Graph{Functions: slices.Clone(fns)}))
		_, diag := graph.Analyze(ctx, g, root)
		return diag.Errors()
	}
	// authoritiesOf returns the authorities of the program src as the live analyzer
	// binds them, with the channel names that r resolves.
	authoritiesOf := func(
		ctx context.Context,
		src v1.Source,
		r symbol.Resolver,
	) ir.Authorities {
		GinkgoHelper()
		root := symbol.NewRoot(r, stl.NewSymbols())
		t := MustSucceed(text.Parse(text.Text{Raw: src.Text}, w.Config))
		aCtx := acontext.NewRoot(ctx, t.AST, root).WithConfig(w.Config)
		analyzer.AnalyzeProgram(aCtx)
		a := authority.Analyze(aCtx)
		Expect(aCtx.Diagnostics.Errors()).To(BeEmpty())
		return a
	}
	function := func(key, body string, inputs ...string) ir.Function {
		fn := ir.Function{
			Key:     key,
			Body:    ir.Body{Raw: body},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
		}
		for _, in := range inputs {
			fn.Inputs = append(fn.Inputs, types.Param{Name: in, Type: types.F64()})
		}
		return fn
	}
	// bodyOf returns the source of the body of fns[i], as the Arc text pass builds it.
	bodyOf := func(fns ir.Functions, i int) v1.Source {
		keys := make([]string, len(fns))
		for j, f := range fns {
			keys[j] = f.Key
		}
		return v1.Source{
			Text:      fns[i].Body.Raw,
			Form:      v1.FormBody,
			Function:  fns[i].Key,
			Inputs:    fns[i].Inputs,
			Outputs:   fns[i].Outputs,
			Functions: keys,
		}
	}

	It("Should report a text that does not parse as not parsed when it holds no old "+
		"name (Result.Parsed: false when the text does not parse)", func(
		ctx SpecContext,
	) {
		res := MustSucceed(w.Rewrite(ctx, calculation("return other *")))
		Expect(res.Parsed).To(BeFalse())
	})

	DescribeTable("Should keep each reference that reads a renamed channel when a "+
		"capture keeps the text (Reference.Kept)",
		func(ctx SpecContext, src v1.Source, lines []int) {
			res := MustSucceed(w.Rewrite(ctx, src))
			Expect(res.Text).To(Equal(src.Text))
			for _, r := range res.References {
				Expect(r.Binding).To(Equal(symbol.KindChannel))
			}
			Expect(keptLines(res)).To(Equal(lines))
		},
		Entry("a function apart from the capture", program(
			"func a() f64 {\n\treturn Metrics\n}\n\n"+
				"func b() f64 {\n\tMetrics_2 := 2.0\n\treturn Metrics * Metrics_2\n}",
		), []int{2, 7}),
		Entry("a read outside the captured block", calculation(
			"if Metrics > 0 {\n\tMetrics_2 := 1.0\n\treturn Metrics * Metrics_2\n}\n"+
				"return Metrics",
		), []int{1, 3, 5}),
		Entry("another renamed channel", program(
			"func a() f64 {\n\treturn sensor\n}\n\n"+
				"func b() f64 {\n\tMetrics_2 := 2.0\n\treturn Metrics * Metrics_2\n}",
		), []int{2, 7}),
	)

	DescribeTable("Should write or keep a read of the channel that the analyzer bound "+
		"before a local of the old name took the name",
		func(ctx SpecContext, src v1.Source) {
			Expect(errorsOf(ctx, src, w.Renames)).To(BeEmpty())
			res := MustSucceed(w.Rewrite(ctx, src))
			if len(keptLines(res)) > 0 {
				return
			}
			after := v1.Source{Form: src.Form, Text: res.Text}
			Expect(errorsOf(ctx, after, upgraded)).
				To(BeEmpty(), "upgraded to %q", res.Text)
		},
		Entry("the initializer of the local",
			calculation("Metrics := Metrics * 2\nreturn Metrics")),
		Entry("a read before the local",
			calculation("y := Metrics * 1\nMetrics := 2.0\nreturn y + Metrics")),
		Entry("a function before a global of the name",
			program("func f() f64 {\n\treturn Metrics\n}\nMetrics := 1.0")),
	)

	// The compiler resolves each name in the finished scope, so a later local or global
	// of the new name would take the rewritten read.
	DescribeTable("Should keep a read of the channel when a symbol of the new name "+
		"declared after it would take the read in the compiled program",
		func(ctx SpecContext, src v1.Source, lines []int) {
			res := MustSucceed(w.Rewrite(ctx, src))
			Expect(res.Text).To(Equal(src.Text))
			Expect(keptLines(res)).To(Equal(lines))
		},
		Entry("a local after the read", calculation(
			"y := Metrics * 1\nMetrics_2 := 2.0\nreturn y + Metrics_2",
		), []int{1}),
		Entry("a local after a block that reads", calculation(
			"if Metrics > 0 {\n\treturn 1.0\n}\nMetrics_2 := 2.0\nreturn Metrics_2",
		), []int{1}),
		Entry("a global after a function that reads", program(
			"func f() f64 {\n\treturn Metrics\n}\nMetrics_2 := 1.0",
		), []int{2}),
	)

	It("Should report a read of a channel named like a prelude function when a later "+
		"global holds its name", func(ctx SpecContext) {
		// Before the global, the root binds the channel ahead of the prelude.
		src := program("func f() f64 {\n\treturn now\n}\nnow := 1.0")
		res := MustSucceed(w.Rewrite(ctx, src))
		Expect(res.Text).To(Equal(src.Text))
		Expect(keptLines(res)).To(Equal([]int{2}))
	})

	It("Should report a read in the second line of a multi-line initializer of a "+
		"local of the old name", func(ctx SpecContext) {
		src := calculation("Metrics := f`a\n{Metrics}`\nreturn 1.0")
		res := MustSucceed(w.Rewrite(ctx, src))
		Expect(res.Text).To(Equal(src.Text))
		Expect(keptLines(res)).To(Equal([]int{2}))
	})

	DescribeTable("Should keep and report a graph body that a v0.62 Core could not "+
		"bind",
		func(ctx SpecContext, fns ir.Functions, i int) {
			Expect(graphErrorsOf(ctx, fns, w.Renames)).To(ContainElement(
				HaveField("Message", ContainSubstring("conflicts with existing")),
			))
			res := MustSucceed(w.Rewrite(ctx, bodyOf(fns, i)))
			Expect(res.Parsed).To(BeFalse())
			Expect(res.Text).To(Equal(fns[i].Body.Raw))
			Expect(res.References).To(HaveEach(HaveField("Kept", BeTrue())))
		},
		Entry("an input named like an earlier function", ir.Functions{
			function("other", "{ return x }", "x"),
			function("scale", "{ return Metrics }", "other"),
		}, 1),
		Entry("two functions of one key", ir.Functions{
			function("scale", "{ return x }", "x"),
			function("scale", "{ return Metrics * x }", "x"),
		}, 1),
	)

	DescribeTable("Should write no new name into an identifier that is not a reference",
		func(ctx SpecContext, src v1.Source, want string) {
			res := MustSucceed(w.Rewrite(ctx, src))
			if len(errorsOf(ctx, src, w.Renames)) == 0 {
				after := v1.Source{Form: src.Form, Text: res.Text}
				Expect(errorsOf(ctx, after, upgraded)).To(
					BeEmpty(),
					"v0.62 accepted %q; upgraded to %q",
					src.Text,
					res.Text,
				)
			}
			Expect(res.Text).To(Equal(want))
		},
		Entry("the key of a named argument",
			program("import time\ntime.wait(duration=1000ms) -> flag"),
			"import time\ntime.wait(duration=1000ms) -> flag_2"),
		Entry("the key of a named argument in the stored v4 export sample",
			program("sensor -> scale(factor=2.0) -> out"),
			"sensor_2 -> scale(factor=2.0) -> out_2"),
		Entry("a routing key in the v0 export sample's flow",
			program("sensor -> scale{factor: 2.0} -> out"),
			"sensor_2 -> scale{factor: 2.0} -> out_2"),
		Entry("the unit of a numeric literal",
			calculation("x i64 ns := 1s\nreturn f64(x)"),
			"x i64 ns := 1s\nreturn f64(x)"),
		Entry("the unit of a declared type",
			calculation("x f64 psi := 1.0\nreturn x"),
			"x f64 psi := 1.0\nreturn x"),
		Entry("the unit of a cast",
			calculation("return f64 psi(Metrics)"),
			"return f64 psi(Metrics_2)"),
	)

	// The analyzer binds inline stage and sequence bodies, and authority entries,
	// after every top-level declaration.
	DescribeTable("Should leave a read in an inline stage or sequence that a later "+
		"global of the old name takes",
		func(ctx SpecContext, src v1.Source, want string) {
			Expect(errorsOf(ctx, src, w.Renames)).To(BeEmpty())
			res := MustSucceed(w.Rewrite(ctx, src))
			after := v1.Source{Form: src.Form, Text: res.Text}
			Expect(errorsOf(ctx, after, upgraded)).
				To(BeEmpty(), "upgraded to %q", res.Text)
			Expect(res.Text).To(Equal(want))
			Expect(keptLines(res)).To(BeEmpty())
		},
		Entry("an inline stage",
			program("flag -> stage {\n\tMetrics -> log\n}\nMetrics := \"a\""),
			"flag_2 -> stage {\n\tMetrics -> log\n}\nMetrics := \"a\""),
		Entry("an inline stage in a named sequence",
			program("sequence main {\n\tstage a {\n\t\tflag -> stage {\n"+
				"\t\t\tMetrics -> log\n\t\t}\n\t}\n}\nMetrics := \"a\""),
			"sequence main {\n\tstage a {\n\t\tflag_2 -> stage {\n"+
				"\t\t\tMetrics -> log\n\t\t}\n\t}\n}\nMetrics := \"a\""),
		Entry("an inline sequence",
			program("flag -> sequence {\n\tMetrics -> log\n}\nMetrics := \"a\""),
			"flag_2 -> sequence {\n\tMetrics -> log\n}\nMetrics := \"a\""),
	)

	It("Should keep and report a read in an inline stage when a later global holds "+
		"the new name", func(ctx SpecContext) {
		src := program("flag -> stage {\n\tMetrics -> out\n}\nMetrics_2 := \"a\"")
		Expect(errorsOf(ctx, src, w.Renames)).To(BeEmpty())
		res := MustSucceed(w.Rewrite(ctx, src))
		Expect(res.Text).To(Equal(src.Text))
		Expect(keptLines(res)).To(ContainElement(2))
	})

	It("Should leave an authority entry that a later global of the old name takes",
		func(ctx SpecContext) {
			src := program("authority (Metrics 200)\nMetrics := 1.0")
			before := authoritiesOf(ctx, src, w.Renames)
			res := MustSucceed(w.Rewrite(ctx, src))
			Expect(res.Text).To(Equal(src.Text))
			Expect(keptLines(res)).To(BeEmpty())
			Expect(authoritiesOf(ctx, program(res.Text), upgraded)).To(Equal(before))
		})

	It("Should keep and report an authority entry when a later global holds the new "+
		"name", func(ctx SpecContext) {
		src := program("authority (Metrics 200)\nMetrics_2 := 1.0")
		res := MustSucceed(w.Rewrite(ctx, src))
		Expect(res.Text).To(Equal(src.Text))
		Expect(keptLines(res)).To(Equal([]int{1}))
	})

	It("Should leave a bare name that a module takes in a graph body, since graph "+
		"mode imports each module", func(ctx SpecContext) {
		root := symbol.NewRoot(w.Renames, stl.NewSymbols())
		symbol.AutoImportModules(root)
		bound := MustSucceed(root.Resolve(ctx, "time"))
		Expect(bound.Kind).To(Equal(symbol.KindModuleAlias))
		fns := ir.Functions{function("scale", "{ return time }", "x")}
		res := MustSucceed(w.Rewrite(ctx, bodyOf(fns, 0)))
		Expect(res.References).To(HaveLen(1))
		Expect(res.References[0].Binding).To(Equal(bound.Kind))
		Expect(res.Text).To(Equal("{ return time }"))
	})
})

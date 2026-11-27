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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
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
	// Channel 1 held "Metrics" and channel 2 held "sensor" before the upgrade.
	var w v1.Rewriter
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		channels := map[v1.Key]v1.TextChannel{}
		for i, r := range [][2]string{{"Metrics", "Metrics_2"}, {"sensor", "sensor_2"}} {
			key := v1.Key(i + 1)
			channels[key] = v1.TextChannel{Name: r[1], DataType: telem.Float64T}
			Expect(resource.StageRename(
				ctx,
				db,
				resource.ID{Type: resource.TypeChannel, Key: key.String()},
				r[0],
				resource.ReasonClash,
			)).To(Succeed())
		}
		w = v1.Rewriter{
			Renames: MustSucceed(v1.TextRename(db, channels)),
			Ambient: stl.NewSymbols(),
		}
	})
	// upgraded resolves each channel name as the Core does after the upgrade.
	channel := func(name string, id int) symbol.Symbol {
		return symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   id,
		}
	}
	upgraded := symtestutil.StaticResolver{
		channel("Metrics_2", 1),
		channel("sensor_2", 2),
	}
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
})

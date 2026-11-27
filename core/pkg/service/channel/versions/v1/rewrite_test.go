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
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Rewriter", func() {
	// Channel 1 held "Metrics" before the upgrade, which renamed it to "Metrics_2".
	var w v1.Rewriter
	BeforeEach(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		Expect(resource.StageRename(
			ctx,
			db,
			resource.ID{Type: resource.TypeChannel, Key: "1"},
			"Metrics",
			resource.ReasonClash,
		)).To(Succeed())
		renames := MustSucceed(v1.TextRename(db, map[v1.Key]v1.TextChannel{
			1: {Name: "Metrics_2", DataType: telem.Float64T},
		}))
		w = v1.Rewriter{Renames: renames, Ambient: stl.NewSymbols()}
	})
	rewrite := func(ctx context.Context, src v1.Source) v1.Result {
		GinkgoHelper()
		return MustSucceed(w.Rewrite(ctx, src))
	}
	kept := func(res v1.Result) []v1.Reference {
		var refs []v1.Reference
		for _, r := range res.References {
			if r.Kept {
				refs = append(refs, r)
			}
		}
		return refs
	}
	calculation := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormCalculation}
	}
	program := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormProgram}
	}
	body := func(text string, inputs ...string) v1.Source {
		src := v1.Source{
			Text:     text,
			Form:     v1.FormBody,
			Function: "scale",
			Outputs:  types.Params{{Name: "output", Type: types.F64()}},
		}
		for _, in := range inputs {
			src.Inputs = append(src.Inputs, types.Param{Name: in, Type: types.F64()})
		}
		return src
	}

	It("Should write the new name where a reference reads the channel", func(
		ctx SpecContext,
	) {
		res := rewrite(ctx, calculation("return Metrics * 2"))
		Expect(res.Text).To(Equal("return Metrics_2 * 2"))
		Expect(res.References).To(ConsistOf(v1.Reference{
			Name:    "Metrics",
			NewName: "Metrics_2",
			Line:    1,
			Column:  7,
			Binding: symbol.KindChannel,
			Bound:   true,
		}))
	})

	It("Should write the new name into a placeholder that stays splittable", func(
		ctx SpecContext,
	) {
		res := rewrite(ctx, calculation("s := f\"{Metrics}\"\nreturn Metrics"))
		Expect(res.Text).To(Equal("s := f\"{Metrics_2}\"\nreturn Metrics_2"))
		segments := MustSucceed(literal.FmtStrSplit("{Metrics_2}", literal.StringFlags{
			Format: true,
		}))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].Text).To(Equal("Metrics_2"))
	})

	It("Should write the new name over a placeholder name written with an escape", func(
		ctx SpecContext,
	) {
		res := rewrite(ctx, calculation("s := f\"{Metr\\u0069cs}\"\nreturn 1"))
		Expect(res.Text).To(Equal("s := f\"{Metrics_2}\"\nreturn 1"))
	})

	It("Should leave a reference that a local of the old name takes", func(
		ctx SpecContext,
	) {
		text := "Metrics f64 := 1.0\nreturn Metrics * 2"
		res := rewrite(ctx, calculation(text))
		Expect(res.Text).To(Equal(text))
		Expect(kept(res)).To(BeEmpty())
	})

	It("Should leave the text and keep the reference when a local holds the new name",
		func(ctx SpecContext) {
			text := "Metrics_2 f64 := 2.0\nreturn Metrics * Metrics_2"
			res := rewrite(ctx, calculation(text))
			Expect(res.Text).To(Equal(text))
			Expect(kept(res)).To(ConsistOf(v1.Reference{
				Name:     "Metrics",
				NewName:  "Metrics_2",
				Line:     2,
				Column:   7,
				Binding:  symbol.KindChannel,
				Bound:    true,
				Captured: true,
				Kept:     true,
			}))
		})

	It("Should leave a graph function body that reads its input of the old name", func(
		ctx SpecContext,
	) {
		res := rewrite(ctx, body("{ return Metrics * 2 }", "Metrics"))
		Expect(res.Text).To(Equal("{ return Metrics * 2 }"))
		Expect(kept(res)).To(BeEmpty())
	})

	It("Should write the new name into a graph function body that reads the channel",
		func(ctx SpecContext) {
			res := rewrite(ctx, body("{ return Metrics * x }", "x"))
			Expect(res.Text).To(Equal("{ return Metrics_2 * x }"))
		})

	It("Should leave a reference that a function parameter of the old name takes",
		func(ctx SpecContext) {
			text := "func scale(Metrics f64) f64 {\n\treturn Metrics * 2\n}"
			res := rewrite(ctx, program(text))
			Expect(res.Text).To(Equal(text))
			Expect(kept(res)).To(BeEmpty())
		})

	It("Should write the new name into a function of a program that reads the channel",
		func(ctx SpecContext) {
			res := rewrite(ctx, program("func read() f64 {\n\treturn Metrics\n}"))
			Expect(res.Text).To(Equal("func read() f64 {\n\treturn Metrics_2\n}"))
		})

	It("Should leave a transition to a stage of the old name", func(ctx SpecContext) {
		text := "sequence main {\n\tstage first {\n\t\t1 => Metrics\n\t}\n" +
			"\tstage Metrics {\n\t}\n}"
		res := rewrite(ctx, program(text))
		Expect(res.Text).To(Equal(text))
		Expect(kept(res)).To(BeEmpty())
	})

	It("Should take no head or member of a qualified name", func(ctx SpecContext) {
		for _, text := range []string{"return Metrics.x", "return x.Metrics"} {
			res := rewrite(ctx, calculation(text))
			Expect(res.Text).To(Equal(text), text)
			Expect(res.References).To(BeEmpty(), text)
		}
	})

	It("Should leave text that does not parse and keep each reference", func(
		ctx SpecContext,
	) {
		res := rewrite(ctx, calculation("return Metrics *"))
		Expect(res.Text).To(Equal("return Metrics *"))
		Expect(res.Parsed).To(BeFalse())
		Expect(res.References).To(ConsistOf(v1.Reference{
			Name:    "Metrics",
			NewName: "Metrics_2",
			Line:    1,
			Column:  7,
			Kept:    true,
		}))
	})

	It("Should read a '-' as a minus with dashed names off", func(ctx SpecContext) {
		res := rewrite(ctx, calculation("return Metrics-1"))
		Expect(res.Text).To(Equal("return Metrics_2-1"))
	})

	It("Should read a '-' as part of a name with dashed names on", func(
		ctx SpecContext,
	) {
		w.Config = parser.Config{AllowDashedNames: true}
		res := rewrite(ctx, calculation("return Metrics-1"))
		Expect(res.Text).To(Equal("return Metrics-1"))
	})
})

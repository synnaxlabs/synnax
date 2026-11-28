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
	"encoding/json"
	"os"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// corpusCase is one stored text of the golden corpus of the text passes.
type corpusCase struct {
	Name   string `json:"name"`
	Dashed bool   `json:"dashed"`
	Form   string `json:"form"`
	Text   string `json:"text"`
	// Inputs are the input names of the function of a body, and Functions the keys of
	// each function of its graph.
	Inputs    []string `json:"inputs,omitempty"`
	Functions []string `json:"functions,omitempty"`
}

// corpusResult is what the live parser and the pass make of a corpusCase.
type corpusResult struct {
	corpusCase
	Parsed     bool            `json:"parsed"`
	References []corpusBinding `json:"references"`
	Rewritten  string          `json:"rewritten"`
}

// corpusBinding is a bare identifier of the old name of a renamed channel, with the
// kind of the symbol that it resolves to in scope.
type corpusBinding struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Binding  string `json:"binding"`
	Captured bool   `json:"captured"`
	Kept     bool   `json:"kept"`
}

const corpusGolden = "testdata/text_corpus.golden.json"

// calc returns a corpus case of a calculation that a Core with dashed names stored.
func calc(name, text string) corpusCase {
	return corpusCase{Name: name, Dashed: true, Form: "calculation", Text: text}
}

// corpus holds texts that a v0.62 Core stored: dashed names, format strings, scopes,
// graph bodies, and the text of an export sample. Channels held "pt-01", "Metrics",
// "sensor", and "time" before the upgrade renamed them.
var corpus = []corpusCase{
	calc("a flow node", "pt-01 -> out{}"),
	calc("an expression", "return pt-01 * 2"),
	calc("an assignment target", "pt-01 = 5"),
	calc("an authority entry", "authority (valve-cmd 200)"),
	calc("a flow arrow after the name", "pt-01->out{}"),
	calc("a compound assignment after the name", "pt-01-=1"),
	calc("each of several names", "x := pt-01 + pt-01\nreturn x"),
	calc("a placeholder", `x := f"{pt-01}"`),
	calc("a placeholder with a spec", `x := f"v={pt-01:.2f}"`),
	calc("a raw format string", `x := rf"{pt-01}"`),
	calc("a multi-line format string", "x := f`{pt-01}\n`"),
	calc("a placeholder after an escape", `x := f"\"{pt-01}"`),
	calc("an escape inside a placeholder name", `x := f"{pt\u002d01}"`),
	calc("a name after digits", "return 10pt-01"),
	calc("a name after a not-equal", "return 1!=pt-01"),
	calc("a dashed name that no channel holds", "return pt-02 * 2"),
	calc("a longer dashed name", "return pt-01-b"),
	calc("a subtraction with spaces", "return pt - 01"),
	calc("a line comment", "// pt-01\nreturn 1"),
	calc("a block comment", "/* pt-01 */ return 1"),
	calc("a string", `x := "pt-01"`),
	calc("escaped braces", `x := f"{{pt-01}}"`),
	calc("a trailing comment that hides the brace", "return Metrics * 2 // x"),
	calc("a bare name", "return Metrics + 1"),
	{
		Name: "a dash with dashed names off",
		Form: "calculation",
		Text: "return Metrics-1",
	},
	calc("a local of the old name", "Metrics f64 := 1.0\nreturn Metrics * 2"),
	calc("a local of the new name", "Metrics_2 f64 := 2.0\nreturn Metrics * Metrics_2"),
	calc(
		"a read before a local of the old name",
		"Metrics := Metrics * 2\nreturn Metrics",
	),
	{
		Name:   "a function before a global of the old name",
		Dashed: true,
		Form:   "program",
		Text:   "func f() f64 {\n\treturn Metrics\n}\nMetrics := 1.0",
	},
	calc("a qualified head", "return Metrics.x"),
	calc("a qualified member", "return x.Metrics"),
	calc("text that does not parse", "return Metrics *"),
	{
		Name:   "a graph body that reads its input of the old name",
		Dashed: true,
		Form:   "body",
		Text:   "{ return Metrics * 2 }",
		Inputs: []string{"Metrics"},
	},
	{
		Name:   "a graph body that reads the channel",
		Dashed: true,
		Form:   "body",
		Text:   "{ return Metrics * x }",
		Inputs: []string{"x"},
	},
	{
		Name:   "a graph body that names a module",
		Dashed: true,
		Form:   "body",
		Text:   "{ return time }",
	},
	{
		Name:      "a graph body that calls another function of its graph",
		Dashed:    true,
		Form:      "body",
		Text:      "{ return g() + Metrics }",
		Functions: []string{"g", "f"},
	},
	{
		Name:      "a graph body with an input named like an earlier function",
		Dashed:    true,
		Form:      "body",
		Text:      "{ return Metrics }",
		Inputs:    []string{"Metrics"},
		Functions: []string{"Metrics", "f"},
	},
	{
		Name:   "a function parameter of the old name",
		Dashed: true,
		Form:   "program",
		Text:   "func scale(Metrics f64) f64 {\n\treturn Metrics * 2\n}",
	},
	{
		Name:   "a stage of the old name",
		Dashed: true,
		Form:   "program",
		Text: "sequence main {\n\tstage first {\n\t\t1 => Metrics\n\t}\n" +
			"\tstage Metrics {\n\t}\n}",
	},
	{
		// The import writes the "{factor=2.0}" of the v0 sample in this form.
		Name:   "the flow of the v0 text export sample",
		Dashed: true,
		Form:   "program",
		Text:   "sensor -> scale{factor: 2.0} -> out",
	},
}

var _ = Describe("Text corpus", func() {
	It("Should parse, bind, and rewrite the corpus as recorded with the v0.62 grammar",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			channels := map[v1.Key]v1.TextChannel{}
			for i, r := range [][2]string{
				{"pt-01", "pt_01"},
				{"Metrics", "Metrics_2"},
				{"sensor", "sensor_2"},
				{"time", "time_2"},
			} {
				key := v1.Channel{Leaseholder: 1, LocalKey: v1.LocalKey(i + 1)}.Key()
				channels[key] = v1.TextChannel{Name: r[1], DataType: telem.Float64T}
				Expect(resource.StageRename(
					ctx,
					db,
					resource.ID{Type: resource.TypeChannel, Key: key.String()},
					r[0],
					resource.ReasonSeparator,
				)).To(Succeed())
			}
			renames := MustSucceed(v1.TextRename(db, channels))
			forms := map[string]v1.Form{
				"calculation": v1.FormCalculation,
				"body":        v1.FormBody,
				"program":     v1.FormProgram,
			}
			results := make([]corpusResult, 0, len(corpus))
			for _, c := range corpus {
				w := v1.Rewriter{
					Renames: renames,
					Config:  parser.Config{AllowDashedNames: c.Dashed},
					Ambient: stl.NewSymbols(),
				}
				src := v1.Source{
					Text:     c.Text,
					Form:     forms[c.Form],
					Function: "f",
					Index:    slices.Index(c.Functions, "f"),
				}
				for _, key := range c.Functions {
					src.Functions = append(src.Functions, v1.GraphFunction{Key: key})
				}
				for _, in := range c.Inputs {
					src.Inputs = append(
						src.Inputs,
						types.Param{Name: in, Type: types.F64()},
					)
				}
				out := MustSucceed(w.Rewrite(ctx, src))
				res := corpusResult{
					corpusCase: c,
					Parsed:     out.Parsed,
					Rewritten:  out.Text,
				}
				for _, r := range out.References {
					b := corpusBinding{
						Name:     r.Name,
						Line:     r.Line,
						Column:   r.Column,
						Binding:  "none",
						Captured: r.Captured,
						Kept:     r.Kept,
					}
					if r.Bound {
						b.Binding = r.Binding.String()
					}
					res.References = append(res.References, b)
				}
				results = append(results, res)
			}
			if os.Getenv("UPDATE_TEXT_CORPUS") == "1" {
				got := MustSucceed(json.MarshalIndent(results, "", "  "))
				Expect(os.WriteFile(corpusGolden, append(got, '\n'), 0o644)).
					To(Succeed())
			}
			var want []corpusResult
			Expect(json.Unmarshal(MustSucceed(os.ReadFile(corpusGolden)), &want)).
				To(Succeed())
			// A change to the Arc grammar or analyzer that changes a result here must
			// first copy a parser frozen at v0.62 for the text passes.
			Expect(results).To(Equal(want))
		})
})

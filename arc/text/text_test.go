package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Text", func() {
	Describe("Parse", func() {
		It("Should correctly parse a text-based arc program", func() {
			source := `
			func add(a i64, b i64) i64 {
				return a + b
			}

			stage adder{} (a i64, b i64) i64 {
				return add(a, b)
			}

			stage print{} () {
			}

			adder{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
		})
	})

	Describe("Analyze", func() {
		It("Should correctly analyze a text-based arc program", func() {
			source := `
			func add(a i64, b i64) i64 {
				return a + b
			}

			stage adder{} (a i64, b i64) i64 {
				return a + b
			}

			stage print{} () {
			}

			adder{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
			inter, diagnostics := text.Analyze(ctx, parsedText, nil)
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(1))
			Expect(inter.Stages).To(HaveLen(2))
			Expect(inter.Nodes).To(HaveLen(2))
			Expect(inter.Edges).To(HaveLen(1))

			f := inter.Functions[0]
			Expect(f.Key).To(Equal("add"))
			Expect(f.Params.Count()).To(Equal(2))
			v, ok := f.Params.Get("a")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(ir.I64{}))
			v, ok = f.Params.Get("b")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(ir.I64{}))

			s := inter.Stages[0]
			Expect(s.Key).To(Equal("adder"))
			Expect(s.Params.Count()).To(Equal(2))
			v, ok = s.Params.Get("a")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(ir.I64{}))
			v, ok = s.Params.Get("b")
			Expect(ok).To(BeTrue())
			Expect(v).To(Equal(ir.I64{}))

			n1 := inter.Nodes[0]
			Expect(n1.Key).To(Equal("adder_0"))
			Expect(n1.Type).To(Equal("adder"))
			Expect(n1.Config).To(HaveLen(0))
			Expect(n1.Channels.Read).ToNot(BeNil())
			Expect(n1.Channels.Read).To(BeEmpty())
			Expect(n1.Channels.Write).ToNot(BeNil())
			Expect(n1.Channels.Write).To(BeEmpty())
		})
	})

})

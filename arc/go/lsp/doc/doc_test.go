// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp/doc"
)

var _ = Describe("Doc", func() {
	Describe("New and Render", func() {
		It("Should render an empty doc as empty string", func() {
			d := doc.New()
			Expect(d.Render()).To(Equal(""))
		})

		It("Should render a single block without extra newlines", func() {
			d := doc.New(doc.Paragraph("Hello"))
			Expect(d.Render()).To(Equal("Hello"))
		})

		It("Should join multiple blocks with double newlines", func() {
			d := doc.New(
				doc.Paragraph("First"),
				doc.Paragraph("Second"),
			)
			Expect(d.Render()).To(Equal("First\n\nSecond"))
		})

		It("Should skip empty blocks in rendering", func() {
			d := doc.New(
				doc.Paragraph("First"),
				doc.List(), // empty list renders as ""
				doc.Paragraph("Second"),
			)
			Expect(d.Render()).To(Equal("First\n\nSecond"))
		})
	})

	Describe("Add", func() {
		It("Should append blocks to existing doc", func() {
			d := doc.New(doc.Paragraph("First"))
			d.Add(doc.Paragraph("Second"))
			Expect(d.Render()).To(Equal("First\n\nSecond"))
		})

		It("Should support chained Add calls", func() {
			d := doc.New()
			d.Add(doc.Paragraph("One")).Add(doc.Paragraph("Two"))
			Expect(d.Render()).To(Equal("One\n\nTwo"))
		})
	})

	Describe("Title", func() {
		It("Should render title without kind", func() {
			t := doc.Title("myFunc")
			Expect(t.Render()).To(Equal("#### myFunc"))
		})

		It("Should render title with kind", func() {
			t := doc.TitleWithKind("i32", "Type")
			Expect(t.Render()).To(Equal("#### i32\n##### Type"))
		})
	})

	Describe("Paragraph", func() {
		It("Should render text directly", func() {
			p := doc.Paragraph("This is a paragraph.")
			Expect(p.Render()).To(Equal("This is a paragraph."))
		})
	})

	Describe("Code", func() {
		It("Should render code block with language", func() {
			c := doc.Code("go", "func main() {}")
			Expect(c.Render()).To(Equal("```go\nfunc main() {}\n```"))
		})

		It("Should render arc code block", func() {
			c := doc.ArcCode("func add(a i32, b i32) i32")
			Expect(c.Render()).To(Equal("```arc\nfunc add(a i32, b i32) i32\n```"))
		})
	})

	Describe("Detail", func() {
		It("Should render detail without code formatting", func() {
			d := doc.Detail("Range", "-128 to 127", false)
			Expect(d.Render()).To(Equal("Range: -128 to 127"))
		})

		It("Should render detail with code formatting", func() {
			d := doc.Detail("Type", "i32", true)
			Expect(d.Render()).To(Equal("Type: `i32`"))
		})
	})

	Describe("Error", func() {
		It("Should render error without code", func() {
			e := doc.Error("Type mismatch")
			Expect(e.Render()).To(Equal("**Error**: Type mismatch"))
		})

		It("Should render error with code", func() {
			e := doc.ErrorWithCode("E001", "Type mismatch")
			Expect(e.Render()).To(Equal("**Error E001**: Type mismatch"))
		})
	})

	Describe("Hint", func() {
		It("Should render hint with italic label", func() {
			h := doc.Hint("Use explicit cast: i32(value)")
			Expect(h.Render()).To(Equal("_Hint_: Use explicit cast: i32(value)"))
		})
	})

	Describe("Fix", func() {
		It("Should render fix without code", func() {
			f := doc.Fix("Remove the unused variable", "")
			Expect(f.Render()).To(Equal("**Fix**: Remove the unused variable"))
		})

		It("Should render fix with code example", func() {
			f := doc.Fix("Cast the value explicitly", "i32(myFloat)")
			expected := "**Fix**: Cast the value explicitly\n\n```arc\ni32(myFloat)\n```"
			Expect(f.Render()).To(Equal(expected))
		})
	})

	Describe("List", func() {
		It("Should render empty list as empty string", func() {
			l := doc.List()
			Expect(l.Render()).To(Equal(""))
		})

		It("Should render unordered list", func() {
			l := doc.List("First item", "Second item", "Third item")
			expected := "- First item\n- Second item\n- Third item"
			Expect(l.Render()).To(Equal(expected))
		})

		It("Should render ordered list", func() {
			l := doc.OrderedList("Step one", "Step two", "Step three")
			expected := "1. Step one\n2. Step two\n3. Step three"
			Expect(l.Render()).To(Equal(expected))
		})
	})

	Describe("Bold", func() {
		It("Should render bold text", func() {
			b := doc.Bold("important")
			Expect(b.Render()).To(Equal("**important**"))
		})
	})

	Describe("Italic", func() {
		It("Should render italic text", func() {
			i := doc.Italic("emphasis")
			Expect(i.Render()).To(Equal("_emphasis_"))
		})
	})

	Describe("InlineCode", func() {
		It("Should render inline code", func() {
			c := doc.InlineCode("myVar")
			Expect(c.Render()).To(Equal("`myVar`"))
		})
	})

	Describe("Divider", func() {
		It("Should render horizontal rule", func() {
			d := doc.Divider()
			Expect(d.Render()).To(Equal("---"))
		})
	})

	Describe("Complex Documents", func() {
		It("Should render type hover documentation", func() {
			d := doc.New(
				doc.TitleWithKind("i32", "Type"),
				doc.Paragraph("Signed 32-bit integer."),
				doc.Detail("Range", "-2147483648 to 2147483647", false),
			)
			expected := "#### i32\n##### Type\n\nSigned 32-bit integer.\n\nRange: -2147483648 to 2147483647"
			Expect(d.Render()).To(Equal(expected))
		})

		It("Should render error diagnostic documentation", func() {
			d := doc.New(
				doc.ErrorWithCode("E001", "Type mismatch"),
				doc.Paragraph("Cannot assign f64 to i32."),
				doc.Hint("Use explicit cast: i32(value)"),
			)
			expected := "**Error E001**: Type mismatch\n\nCannot assign f64 to i32.\n\n_Hint_: Use explicit cast: i32(value)"
			Expect(d.Render()).To(Equal(expected))
		})

		It("Should render keyword hover documentation", func() {
			d := doc.New(
				doc.Title("func"),
				doc.Paragraph("Declares a function."),
				doc.ArcCode("func name(param type) returnType {\n    // body\n}"),
			)
			expected := "#### func\n\nDeclares a function.\n\n```arc\nfunc name(param type) returnType {\n    // body\n}\n```"
			Expect(d.Render()).To(Equal(expected))
		})
	})
})

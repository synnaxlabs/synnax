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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Get", func() {
	DescribeTable("extracts documentation from domains",
		func(domains map[string]resolution.Domain, expected string) {
			Expect(doc.Get(domains)).To(Equal(expected))
		},
		Entry("doc domain with string value",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{
					Name:   "value",
					Values: []resolution.ExpressionValue{{StringValue: "User represents a system user."}},
				}}},
			}, "User represents a system user."),
		Entry("doc domain with expression name only",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{Name: "Inline documentation"}}},
			}, "Inline documentation"),
		Entry("missing doc domain",
			map[string]resolution.Domain{
				"other": {Expressions: []resolution.Expression{{Name: "something"}}},
			}, ""),
		Entry("empty domains map", map[string]resolution.Domain{}, ""),
		Entry("nil domains map", nil, ""),
		Entry("doc domain with empty expressions",
			map[string]resolution.Domain{"doc": {Expressions: []resolution.Expression{}}}, ""),
		Entry("doc domain with empty values returns expression name",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{{Name: "fallback", Values: []resolution.ExpressionValue{}}}},
			}, "fallback"),
		Entry("takes first expression when multiple present",
			map[string]resolution.Domain{
				"doc": {Expressions: []resolution.Expression{
					{Name: "first", Values: []resolution.ExpressionValue{{StringValue: "First doc"}}},
					{Name: "second", Values: []resolution.ExpressionValue{{StringValue: "Second doc"}}},
				}},
			}, "First doc"),
	)
})

var _ = Describe("FormatGo", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatGo("Name", "")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatGo("Name", "doc text")).To(Equal("// Name doc text"))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatGo("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("// Name line1 line2 line3"))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "contains memory base addresses for multi-output functions, mapping function keys to their base addresses."
		result := doc.FormatGo("output_memory_bases", longDoc)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
		Expect(len(lines)).To(BeNumerically(">", 1), "expected multiple lines")
	})
	It("should normalize awkward line breaks in source text", func() {
		// This simulates the problematic input from .oracle files
		awkwardDoc := "contains memory base addresses for multi-output\nfunctions, mapping\nfunction keys to their base addresses."
		result := doc.FormatGo("output_memory_bases", awkwardDoc)
		lines := strings.SplitSeq(result, "\n")
		for line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
		// Verify that "functions, mapping" is not on its own short line
		Expect(result).NotTo(ContainSubstring("// functions, mapping\n"))
	})
	It("should preserve paragraph breaks (double newline)", func() {
		docWithParagraphs := "First paragraph text.\n\nSecond paragraph text."
		result := doc.FormatGo("Name", docWithParagraphs)
		Expect(result).To(ContainSubstring("//\n"))
	})
})

var _ = Describe("FormatTS", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatTS("Name", "")).To(Equal(""))
	})
	It("should return empty string for whitespace-only doc", func() {
		Expect(doc.FormatTS("Name", " ")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatTS("Name", "doc text")).To(Equal("/** Name doc text */"))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatTS("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("/** Name line1 line2 line3 */"))
	})
	It("should handle paragraph breaks in multi-line doc", func() {
		result := doc.FormatTS("Name", "line1\n\nline3")
		Expect(result).To(Equal("/**\n * Name line1\n *\n * line3\n */"))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "is the node that holds the lease for this channel. Mostly for internal use and other purposes."
		result := doc.FormatTS("leaseholder", longDoc)
		lines := strings.Split(result, "\n")
		Expect(len(lines)).To(BeNumerically(">", 1), "expected multiple lines")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
	})
})

var _ = Describe("FormatPyDocstring", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatPyDocstring("Name", "")).To(Equal(""))
	})
	It("should return empty string for whitespace-only doc", func() {
		Expect(doc.FormatPyDocstring("Name", " ")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatPyDocstring("Name", "doc text")).To(Equal(`"""Name doc text"""`))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatPyDocstring("Name", "line1\nline2\nline3")
		Expect(result).To(Equal(`"""Name line1 line2 line3"""`))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "is the node that holds the lease for this channel. Mostly for internal use and other purposes."
		result := doc.FormatPyDocstring("leaseholder", longDoc)
		lines := strings.Split(result, "\n")
		Expect(len(lines)).To(BeNumerically(">", 1), "expected multiple lines")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
	})
})

var _ = Describe("FormatPyComment", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatPyComment("Name", "")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatPyComment("Name", "doc text")).To(Equal("# Name doc text"))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatPyComment("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("# Name line1 line2 line3"))
	})
	It("should preserve paragraph breaks (double newline)", func() {
		result := doc.FormatPyComment("Name", "First paragraph.\n\nSecond paragraph.")
		Expect(result).To(Equal("# Name First paragraph.\n#\n# Second paragraph."))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "is the node that holds the lease for this channel. Mostly for internal use and other purposes."
		result := doc.FormatPyComment("leaseholder", longDoc)
		lines := strings.Split(result, "\n")
		Expect(len(lines)).To(BeNumerically(">", 1), "expected multiple lines")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
	})
})

var _ = Describe("FormatCpp", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatCpp("Name", "")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatCpp("Name", "doc text")).To(Equal("/// @brief Name doc text"))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatCpp("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("/// @brief Name line1 line2 line3"))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "contains memory base addresses for multi-output functions, mapping function keys to their base addresses."
		result := doc.FormatCpp("output_memory_bases", longDoc)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
		Expect(len(lines)).To(BeNumerically(">", 1), "expected multiple lines")
	})
	It("should normalize awkward line breaks in source text", func() {
		awkwardDoc := "contains memory base addresses for multi-output\nfunctions, mapping\nfunction keys to their base addresses."
		result := doc.FormatCpp("output_memory_bases", awkwardDoc)
		lines := strings.SplitSeq(result, "\n")
		for line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
		// Verify that "functions, mapping" is not on its own short line
		Expect(result).NotTo(ContainSubstring("/// functions, mapping\n"))
	})
	It("should preserve paragraph breaks (double newline)", func() {
		docWithParagraphs := "First paragraph text.\n\nSecond paragraph text."
		result := doc.FormatCpp("Name", docWithParagraphs)
		Expect(result).To(ContainSubstring("///\n"))
	})
})

var _ = Describe("FormatProto", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatProto("Name", "")).To(Equal(""))
	})
	It("should return empty string for whitespace-only doc", func() {
		Expect(doc.FormatProto("Name", " ")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatProto("Name", "doc text")).To(Equal("// Name doc text"))
	})
	It("should preserve paragraph breaks (double newline)", func() {
		result := doc.FormatProto("Name", "First paragraph.\n\nSecond paragraph.")
		Expect(result).To(Equal("// Name First paragraph.\n//\n// Second paragraph."))
	})
	It("should format multi-line doc by normalizing newlines", func() {
		result := doc.FormatProto("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("// Name line1 line2 line3"))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "contains memory base addresses for multi-output functions, mapping function keys to their base addresses."
		result := doc.FormatProto("output_memory_bases", longDoc)
		lines := strings.SplitSeq(result, "\n")
		for line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
	})
	It("should account for indentation when wrapping", func() {
		longDoc := "is the channel used to index this channel's values, associating each value with a timestamp."
		result := doc.FormatProto("index", longDoc, 2)
		lines := strings.SplitSeq(result, "\n")
		for line := range lines {
			Expect(2+len(line)).To(BeNumerically("<=", 88), "indented line exceeds 88 chars: %s", line)
		}
	})
})

var _ = Describe("FormatPyDocstringGoogle", func() {
	It("should return empty string when no docs", func() {
		Expect(doc.FormatPyDocstringGoogle("", nil)).To(Equal(""))
		Expect(doc.FormatPyDocstringGoogle("", []doc.FieldDoc{})).To(Equal(""))
	})
	It("should format class doc only", func() {
		result := doc.FormatPyDocstringGoogle("a status message.", nil)
		expected := `    """A status message.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should format fields only", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "unique identifier."},
			{Name: "name", Doc: "human-readable name."},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		expected := `    """
    Attributes:
        key: Unique identifier.
        name: Human-readable name.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should format class doc with fields", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "unique identifier."},
		}
		result := doc.FormatPyDocstringGoogle("a status message.", fields)
		expected := `    """A status message.

    Attributes:
        key: Unique identifier.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should wrap long class docs to 88 characters including indentation", func() {
		longDoc := "is the node that holds the lease for this channel. Mostly for internal use and other purposes."
		result := doc.FormatPyDocstringGoogle(longDoc, nil)
		lines := strings.Split(result, "\n")
		Expect(len(lines)).To(BeNumerically(">", 2), "expected wrapped lines")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
	})
	It("should preserve paragraph breaks in class docs", func() {
		result := doc.FormatPyDocstringGoogle("first paragraph.\n\nsecond paragraph.", nil)
		expected := `    """First paragraph.

    second paragraph.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should preserve paragraph breaks in field docs", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "first paragraph.\n\nsecond paragraph."},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		expected := `    """
    Attributes:
        key: First paragraph.

            second paragraph.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should normalize newlines in multi-line class doc", func() {
		result := doc.FormatPyDocstringGoogle("first line.\nsecond line.", nil)
		expected := `    """First line. second line.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should normalize newlines in multi-line field doc", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "first line.\nsecond line."},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		expected := `    """
    Attributes:
        key: First line. second line.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should wrap long field docs to 88 characters including indentation", func() {
		fields := []doc.FieldDoc{
			{Name: "leaseholder", Doc: "is the node that holds the lease for this channel. Mostly for internal use."},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		for line := range strings.SplitSeq(result, "\n") {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
		}
		Expect(result).To(ContainSubstring("        leaseholder: Is the node"))
		Expect(result).To(ContainSubstring("\n            "))
	})
	It("should skip fields with empty docs", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "has doc."},
			{Name: "name", Doc: ""},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		expected := `    """
    Attributes:
        key: Has doc.
    """`
		Expect(result).To(Equal(expected))
	})
})

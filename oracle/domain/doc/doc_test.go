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
		lines := strings.Split(result, "\n")
		for _, line := range lines {
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
	It("should format single-line doc", func() {
		Expect(doc.FormatTS("Name", "doc text")).To(Equal("/** Name doc text */"))
	})
	It("should format multi-line doc", func() {
		result := doc.FormatTS("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("/**\n * Name line1\n * line2\n * line3\n */"))
	})
	It("should handle empty lines in multi-line doc", func() {
		result := doc.FormatTS("Name", "line1\n\nline3")
		Expect(result).To(Equal("/**\n * Name line1\n *\n * line3\n */"))
	})
})

var _ = Describe("FormatPyDocstring", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatPyDocstring("Name", "")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatPyDocstring("Name", "doc text")).To(Equal(`"""Name doc text"""`))
	})
	It("should format multi-line doc", func() {
		result := doc.FormatPyDocstring("Name", "line1\nline2\nline3")
		Expect(result).To(Equal(`"""Name line1` + "\nline2\nline3" + `"""`))
	})
})

var _ = Describe("FormatPyComment", func() {
	It("should return empty string for empty doc", func() {
		Expect(doc.FormatPyComment("Name", "")).To(Equal(""))
	})
	It("should format single-line doc", func() {
		Expect(doc.FormatPyComment("Name", "doc text")).To(Equal("# Name doc text"))
	})
	It("should format multi-line doc", func() {
		result := doc.FormatPyComment("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("# Name line1\n# line2\n# line3"))
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
		lines := strings.Split(result, "\n")
		for _, line := range lines {
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
	It("should format single-line doc", func() {
		Expect(doc.FormatProto("Name", "doc text")).To(Equal("// Name doc text"))
	})
	It("should format multi-line doc by normalizing newlines (delegates to FormatGo)", func() {
		result := doc.FormatProto("Name", "line1\nline2\nline3")
		Expect(result).To(Equal("// Name line1 line2 line3"))
	})
	It("should wrap long text to 88 characters", func() {
		longDoc := "contains memory base addresses for multi-output functions, mapping function keys to their base addresses."
		result := doc.FormatProto("output_memory_bases", longDoc)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			Expect(len(line)).To(BeNumerically("<=", 88), "line exceeds 88 chars: %s", line)
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
	It("should handle multi-line class doc", func() {
		result := doc.FormatPyDocstringGoogle("first line.\nsecond line.", nil)
		expected := `    """First line.
    second line.
    """`
		Expect(result).To(Equal(expected))
	})
	It("should handle multi-line field doc", func() {
		fields := []doc.FieldDoc{
			{Name: "key", Doc: "first line.\nsecond line."},
		}
		result := doc.FormatPyDocstringGoogle("", fields)
		expected := `    """
    Attributes:
        key: First line.
            second line.
    """`
		Expect(result).To(Equal(expected))
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

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	"go.lsp.dev/protocol"
)

func TestWrapper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LSP Wrapper Suite")
}

var _ = Describe("ExtractMetadataFromURI", func() {
	It("Should extract metadata from valid block URI", func() {
		metadata := lsp.DocumentMetadata{
			IsBlock: true,
		}

		jsonData, err := json.Marshal(metadata)
		Expect(err).ToNot(HaveOccurred())

		encoded := base64.StdEncoding.EncodeToString(jsonData)
		uri := protocol.DocumentURI("arc://block/123#" + encoded)

		extracted := lsp.ExtractMetadataFromURI(uri)
		Expect(extracted).ToNot(BeNil())
		Expect(extracted.IsBlock).To(BeTrue())
	})

	It("Should return nil for non-block URI", func() {
		uri := protocol.DocumentURI("file:///some/file.arc")
		extracted := lsp.ExtractMetadataFromURI(uri)
		Expect(extracted).To(BeNil())
	})

	It("Should return nil for URI without metadata fragment", func() {
		uri := protocol.DocumentURI("arc://block/123")
		extracted := lsp.ExtractMetadataFromURI(uri)
		Expect(extracted).To(BeNil())
	})

	It("Should return nil for URI with invalid base64", func() {
		uri := protocol.DocumentURI("arc://block/123#invalid!!!base64")
		extracted := lsp.ExtractMetadataFromURI(uri)
		Expect(extracted).To(BeNil())
	})
})

var _ = Describe("WrapExpression", func() {
	Context("Single-line expression", func() {
		It("Should wrap simple return statement", func() {
			metadata := &lsp.DocumentMetadata{
				IsBlock: true,
			}

			expression := "return sensor * 1.8 + 32"
			wrapper := lsp.WrapExpression(expression, metadata)

			Expect(wrapper).ToNot(BeNil())
			Expect(wrapper.OriginalContent).To(Equal(expression))
			Expect(wrapper.LineOffset).To(Equal(1))
			Expect(wrapper.ColumnOffset).To(Equal(0))

			expected := "func __block() {\nreturn sensor * 1.8 + 32\n}"
			Expect(wrapper.WrappedContent).To(Equal(expected))
		})

		It("Should handle expression without trailing newline", func() {
			metadata := &lsp.DocumentMetadata{
				IsBlock: true,
			}

			expression := "return x * 2"
			wrapper := lsp.WrapExpression(expression, metadata)

			Expect(wrapper.WrappedContent).To(Equal("func __block() {\nreturn x * 2\n}"))
		})
	})

	Context("Multi-line expression", func() {
		It("Should wrap multi-line block without indentation", func() {
			metadata := &lsp.DocumentMetadata{
				IsBlock: true,
			}

			expression := "let temp_c = sensor\nlet temp_f = temp_c * 1.8 + 32\nreturn temp_f"
			wrapper := lsp.WrapExpression(expression, metadata)

			expected := `func __block() {
let temp_c = sensor
let temp_f = temp_c * 1.8 + 32
return temp_f
}`
			Expect(wrapper.WrappedContent).To(Equal(expected))
			Expect(wrapper.LineOffset).To(Equal(1))
			Expect(wrapper.ColumnOffset).To(Equal(0))
		})

		It("Should preserve existing indentation", func() {
			metadata := &lsp.DocumentMetadata{
				IsBlock: true,
			}

			// User might have indented their code
			expression := "  let y = x * 2\n  return y"
			wrapper := lsp.WrapExpression(expression, metadata)

			expected := `func __block() {
  let y = x * 2
  return y
}`
			Expect(wrapper.WrappedContent).To(Equal(expected))
		})
	})

	Context("Non-block", func() {
		It("Should return unwrapped content when metadata is nil", func() {
			expression := "func main() { return 0 }"
			wrapper := lsp.WrapExpression(expression, nil)

			Expect(wrapper.WrappedContent).To(Equal(expression))
			Expect(wrapper.OriginalContent).To(Equal(expression))
			Expect(wrapper.LineOffset).To(Equal(0))
			Expect(wrapper.ColumnOffset).To(Equal(0))
		})

		It("Should return unwrapped content when not a block", func() {
			metadata := &lsp.DocumentMetadata{
				IsBlock: false,
			}

			expression := "func main() { return 0 }"
			wrapper := lsp.WrapExpression(expression, metadata)

			Expect(wrapper.WrappedContent).To(Equal(expression))
			Expect(wrapper.LineOffset).To(Equal(0))
		})
	})
})

var _ = Describe("MapDiagnosticPosition", func() {
	var wrapper *lsp.WrapperContext

	BeforeEach(func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}
		expression := "let x = sensor * 2\nreturn x"
		wrapper = lsp.WrapExpression(expression, metadata)
	})

	It("Should map position from wrapped to original", func() {
		// Line 1 in wrapped = line 0 in original
		wrappedPos := protocol.Position{Line: 1, Character: 0}
		originalPos := wrapper.MapDiagnosticPosition(wrappedPos)

		Expect(originalPos.Line).To(Equal(uint32(0)))
		Expect(originalPos.Character).To(Equal(uint32(0)))
	})

	It("Should map position with column offset", func() {
		// Line 2, column 5 in wrapped
		wrappedPos := protocol.Position{Line: 2, Character: 5}
		originalPos := wrapper.MapDiagnosticPosition(wrappedPos)

		Expect(originalPos.Line).To(Equal(uint32(1))) // Line 2 - 1 offset
		Expect(originalPos.Character).To(Equal(uint32(5)))
	})

	It("Should clamp positions before expression to line 0", func() {
		// Function signature line (line 0 in wrapped)
		wrappedPos := protocol.Position{Line: 0, Character: 10}
		originalPos := wrapper.MapDiagnosticPosition(wrappedPos)

		Expect(originalPos.Line).To(Equal(uint32(0)))
		Expect(originalPos.Character).To(Equal(uint32(0)))
	})

	It("Should map range correctly", func() {
		wrappedRange := protocol.Range{
			Start: protocol.Position{Line: 1, Character: 4},
			End:   protocol.Position{Line: 1, Character: 9},
		}

		originalRange := wrapper.MapDiagnosticRange(wrappedRange)

		Expect(originalRange.Start.Line).To(Equal(uint32(0)))
		Expect(originalRange.Start.Character).To(Equal(uint32(4)))
		Expect(originalRange.End.Line).To(Equal(uint32(0)))
		Expect(originalRange.End.Character).To(Equal(uint32(9)))
	})

	It("Should map multi-line range", func() {
		wrappedRange := protocol.Range{
			Start: protocol.Position{Line: 1, Character: 0},
			End:   protocol.Position{Line: 2, Character: 8},
		}

		originalRange := wrapper.MapDiagnosticRange(wrappedRange)

		Expect(originalRange.Start.Line).To(Equal(uint32(0)))
		Expect(originalRange.Start.Character).To(Equal(uint32(0)))
		Expect(originalRange.End.Line).To(Equal(uint32(1)))
		Expect(originalRange.End.Character).To(Equal(uint32(8)))
	})
})

var _ = Describe("MapOriginalToWrapped", func() {
	var wrapper *lsp.WrapperContext

	BeforeEach(func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}
		expression := "return sensor * 2"
		wrapper = lsp.WrapExpression(expression, metadata)
	})

	It("Should map position from original to wrapped", func() {
		// Line 0 in original = line 1 in wrapped
		originalPos := protocol.Position{Line: 0, Character: 7}
		wrappedPos := wrapper.MapOriginalToWrapped(originalPos)

		Expect(wrappedPos.Line).To(Equal(uint32(1)))
		Expect(wrappedPos.Character).To(Equal(uint32(7)))
	})

	It("Should handle multi-line mapping", func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}
		expression := "let y = x\nreturn y"
		wrapper := lsp.WrapExpression(expression, metadata)

		originalPos := protocol.Position{Line: 1, Character: 7}
		wrappedPos := wrapper.MapOriginalToWrapped(originalPos)

		Expect(wrappedPos.Line).To(Equal(uint32(2)))
		Expect(wrappedPos.Character).To(Equal(uint32(7)))
	})
})

var _ = Describe("Edge Cases", func() {
	It("Should handle empty expression", func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}

		wrapper := lsp.WrapExpression("", metadata)

		Expect(wrapper.WrappedContent).To(Equal("func __block() {\n\n}"))
	})

	It("Should handle expression with only whitespace", func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}

		wrapper := lsp.WrapExpression("   \n  \n", metadata)

		Expect(wrapper.WrappedContent).To(ContainSubstring("func __block()"))
	})

	It("Should handle expression ending with newline", func() {
		metadata := &lsp.DocumentMetadata{
			IsBlock: true,
		}

		wrapper := lsp.WrapExpression("return x\n", metadata)

		// Should not add extra newline
		Expect(wrapper.WrappedContent).To(Equal("func __block() {\nreturn x\n}"))
	})
})

var _ = Describe("URL Encoding Regression", func() {
	It("Should handle URL-encoded URI fragments (= becomes %3D)", func() {
		// Monaco URL-encodes the fragment, turning = into %3D
		// Base64: eyJpc19ibG9jayI6dHJ1ZX0= (with trailing =)
		// URL-encoded: eyJpc19ibG9jayI6dHJ1ZX0%3D
		metadata := lsp.ExtractMetadataFromURI("arc://block/test123#eyJpc19ibG9jayI6dHJ1ZX0%3D")
		Expect(metadata).ToNot(BeNil())
		Expect(metadata.IsBlock).To(BeTrue())
	})

	It("Should handle unencoded URI fragments", func() {
		// Also support unencoded fragments for compatibility
		metadata := lsp.ExtractMetadataFromURI("arc://block/test#eyJpc19ibG9jayI6dHJ1ZX0=")
		Expect(metadata).ToNot(BeNil())
		Expect(metadata.IsBlock).To(BeTrue())
	})

	It("Should handle base64 with double padding (%3D%3D)", func() {
		// Some base64 strings have double padding
		// {"is_block": true, "test": "value"}
		// Base64 with double padding encoded by Monaco
		encoded := "eyJpc19ibG9jayI6IHRydWUsICJ0ZXN0IjogInZhbHVlIn0%3D%3D"
		metadata := lsp.ExtractMetadataFromURI(protocol.DocumentURI("arc://block/test#" + encoded))
		Expect(metadata).ToNot(BeNil())
		Expect(metadata.IsBlock).To(BeTrue())
	})

	It("Should return nil for invalid URL encoding", func() {
		// Malformed percent encoding
		metadata := lsp.ExtractMetadataFromURI("arc://block/test#invalid%")
		Expect(metadata).To(BeNil())
	})
})

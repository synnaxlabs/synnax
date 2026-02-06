// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("PositionToOffset", func() {
	It("Should return 0 for position at origin", func() {
		Expect(lsp.PositionToOffset("hello", protocol.Position{Line: 0, Character: 0})).To(Equal(0))
	})

	It("Should return correct offset within first line", func() {
		Expect(lsp.PositionToOffset("hello world", protocol.Position{Line: 0, Character: 5})).To(Equal(5))
	})

	It("Should return correct offset on second line", func() {
		Expect(lsp.PositionToOffset("hello\nworld", protocol.Position{Line: 1, Character: 3})).To(Equal(9))
	})

	It("Should clamp to content length for out-of-bounds line", func() {
		Expect(lsp.PositionToOffset("hello", protocol.Position{Line: 5, Character: 0})).To(Equal(5))
	})

	It("Should clamp to content length for out-of-bounds character", func() {
		Expect(lsp.PositionToOffset("hi", protocol.Position{Line: 0, Character: 100})).To(Equal(2))
	})

	It("Should handle empty content", func() {
		Expect(lsp.PositionToOffset("", protocol.Position{Line: 0, Character: 0})).To(Equal(0))
	})

	It("Should handle multi-line content", func() {
		content := "line1\nline2\nline3"
		Expect(lsp.PositionToOffset(content, protocol.Position{Line: 2, Character: 2})).To(Equal(14))
	})
})

var _ = Describe("ApplyIncrementalChange", func() {
	It("Should insert at the beginning", func() {
		result := lsp.ApplyIncrementalChange("world", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Text: "hello ",
		})
		Expect(result).To(Equal("hello world"))
	})

	It("Should insert in the middle", func() {
		result := lsp.ApplyIncrementalChange("helo", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 0, Character: 3},
			},
			Text: "l",
		})
		Expect(result).To(Equal("hello"))
	})

	It("Should insert at the end", func() {
		result := lsp.ApplyIncrementalChange("hello", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: " world",
		})
		Expect(result).To(Equal("hello world"))
	})

	It("Should delete text", func() {
		result := lsp.ApplyIncrementalChange("hello world", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 11},
			},
			Text: "",
		})
		Expect(result).To(Equal("hello"))
	})

	It("Should replace with different-length text", func() {
		result := lsp.ApplyIncrementalChange("hello world", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 11},
			},
			Text: "there",
		})
		Expect(result).To(Equal("hello there"))
	})

	It("Should handle multi-line edits", func() {
		content := "line1\nline2\nline3"
		result := lsp.ApplyIncrementalChange(content, protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			Text: "replaced",
		})
		Expect(result).To(Equal("line1\nreplaced\nline3"))
	})

	It("Should handle empty document", func() {
		result := lsp.ApplyIncrementalChange("", protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			Text: "new content",
		})
		Expect(result).To(Equal("new content"))
	})

	It("Should handle cross-line replacement", func() {
		content := "line1\nline2\nline3"
		result := lsp.ApplyIncrementalChange(content, protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 2, Character: 2},
			},
			Text: "X",
		})
		Expect(result).To(Equal("linXne3"))
	})
})

var _ = Describe("IsFullReplacement", func() {
	It("Should return true for zero range and zero range length", func() {
		Expect(lsp.IsFullReplacement(protocol.TextDocumentContentChangeEvent{
			Text: "new content",
		})).To(BeTrue())
	})

	It("Should return false when range is specified", func() {
		Expect(lsp.IsFullReplacement(protocol.TextDocumentContentChangeEvent{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			Text: "new",
		})).To(BeFalse())
	})

	It("Should return false when range length is non-zero", func() {
		Expect(lsp.IsFullReplacement(protocol.TextDocumentContentChangeEvent{
			RangeLength: 5,
			Text:        "new",
		})).To(BeFalse())
	})
})

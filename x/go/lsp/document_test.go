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
	"github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

var _ = Describe("Document", func() {
	Describe("PositionToOffset", func() {
		It("Should convert a position on the first line", func() {
			Expect(lsp.PositionToOffset("hello", protocol.Position{Line: 0, Character: 3})).To(Equal(3))
		})

		It("Should convert a position on a subsequent line", func() {
			content := "hello\nworld"
			Expect(lsp.PositionToOffset(content, protocol.Position{Line: 1, Character: 2})).To(Equal(8))
		})

		It("Should clamp beyond-content positions to content length", func() {
			Expect(lsp.PositionToOffset("hi", protocol.Position{Line: 5, Character: 0})).To(Equal(2))
		})

		It("Should clamp character beyond line length", func() {
			Expect(lsp.PositionToOffset("hi", protocol.Position{Line: 0, Character: 100})).To(Equal(2))
		})

		It("Should return 0 for position at start", func() {
			Expect(lsp.PositionToOffset("hello", protocol.Position{Line: 0, Character: 0})).To(Equal(0))
		})

		It("Should handle empty content", func() {
			Expect(lsp.PositionToOffset("", protocol.Position{Line: 0, Character: 0})).To(Equal(0))
		})

		It("Should handle multi-line content with later line", func() {
			content := "line1\nline2\nline3"
			Expect(lsp.PositionToOffset(content, protocol.Position{Line: 2, Character: 2})).To(Equal(14))
		})
	})

	Describe("IsFullReplacement", func() {
		It("Should detect a full replacement", func() {
			change := protocol.TextDocumentContentChangeEvent{Text: "new content"}
			Expect(lsp.IsFullReplacement(change)).To(BeTrue())
		})

		It("Should detect an incremental change", func() {
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 5},
				},
				Text: "world",
			}
			Expect(lsp.IsFullReplacement(change)).To(BeFalse())
		})

		It("Should detect an incremental change via RangeLength", func() {
			change := protocol.TextDocumentContentChangeEvent{
				RangeLength: 5,
				Text:        "new",
			}
			Expect(lsp.IsFullReplacement(change)).To(BeFalse())
		})
	})

	Describe("ApplyIncrementalChange", func() {
		It("Should apply a single edit", func() {
			content := "hello world"
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 6},
					End:   protocol.Position{Line: 0, Character: 11},
				},
				Text: "there",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal("hello there"))
		})

		It("Should apply an insertion", func() {
			content := "helloworld"
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 5},
					End:   protocol.Position{Line: 0, Character: 5},
				},
				Text: " ",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal("hello world"))
		})

		It("Should apply a deletion", func() {
			content := "hello world"
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 5},
					End:   protocol.Position{Line: 0, Character: 6},
				},
				Text: "",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal("helloworld"))
		})

		It("Should apply a multi-line edit", func() {
			content := "line1\nline2\nline3"
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 5},
				},
				Text: "replaced",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal("line1\nreplaced\nline3"))
		})

		It("Should handle empty document", func() {
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Text: "new content",
			}
			Expect(lsp.ApplyIncrementalChange("", change)).To(Equal("new content"))
		})

		It("Should handle cross-line replacement", func() {
			content := "line1\nline2\nline3"
			change := protocol.TextDocumentContentChangeEvent{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 3},
					End:   protocol.Position{Line: 2, Character: 2},
				},
				Text: "X",
			}
			Expect(lsp.ApplyIncrementalChange(content, change)).To(Equal("linXne3"))
		})
	})

	Describe("SplitLines", func() {
		It("Should split Unix-style endings", func() {
			Expect(lsp.SplitLines("a\nb\nc")).To(Equal([]string{"a", "b", "c"}))
		})

		It("Should normalize Windows-style endings", func() {
			Expect(lsp.SplitLines("a\r\nb\r\nc")).To(Equal([]string{"a", "b", "c"}))
		})

		It("Should handle mixed endings", func() {
			Expect(lsp.SplitLines("a\nb\r\nc")).To(Equal([]string{"a", "b", "c"}))
		})

		It("Should return a single-element slice for no newlines", func() {
			Expect(lsp.SplitLines("hello")).To(Equal([]string{"hello"}))
		})
	})

	Describe("GetLine", func() {
		It("Should return the requested line", func() {
			line, ok := lsp.GetLine("first\nsecond\nthird", 1)
			Expect(ok).To(BeTrue())
			Expect(line).To(Equal("second"))
		})

		It("Should return false for out-of-bounds line", func() {
			_, ok := lsp.GetLine("hello", 5)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("IsWordChar", func() {
		It("Should accept alphanumeric characters", func() {
			Expect(lsp.IsWordChar('a')).To(BeTrue())
			Expect(lsp.IsWordChar('Z')).To(BeTrue())
			Expect(lsp.IsWordChar('5')).To(BeTrue())
		})

		It("Should accept underscores", func() {
			Expect(lsp.IsWordChar('_')).To(BeTrue())
		})

		It("Should reject special characters", func() {
			Expect(lsp.IsWordChar(' ')).To(BeFalse())
			Expect(lsp.IsWordChar('+')).To(BeFalse())
			Expect(lsp.IsWordChar('.')).To(BeFalse())
		})
	})

	Describe("GetWordAtPosition", func() {
		It("Should extract a word in the middle of a line", func() {
			Expect(lsp.GetWordAtPosition("hello world", protocol.Position{Line: 0, Character: 7})).To(Equal("world"))
		})

		It("Should extract a word at the beginning of a line", func() {
			Expect(lsp.GetWordAtPosition("hello world", protocol.Position{Line: 0, Character: 0})).To(Equal("hello"))
		})

		It("Should return empty for position beyond line", func() {
			Expect(lsp.GetWordAtPosition("hi", protocol.Position{Line: 0, Character: 10})).To(Equal(""))
		})

		It("Should return empty for position on empty content", func() {
			Expect(lsp.GetWordAtPosition("", protocol.Position{Line: 0, Character: 0})).To(Equal(""))
		})

		It("Should extract word from multi-line content", func() {
			Expect(lsp.GetWordAtPosition("first\nsecond", protocol.Position{Line: 1, Character: 2})).To(Equal("second"))
		})
	})

	Describe("GetWordRangeAtPosition", func() {
		It("Should return correct range for a word", func() {
			r := lsp.GetWordRangeAtPosition("hello world", protocol.Position{Line: 0, Character: 7})
			Expect(r).ToNot(BeNil())
			Expect(r.Start.Character).To(Equal(uint32(6)))
			Expect(r.End.Character).To(Equal(uint32(11)))
		})

		It("Should return nil for empty content", func() {
			Expect(lsp.GetWordRangeAtPosition("", protocol.Position{Line: 0, Character: 0})).To(BeNil())
		})

		It("Should return nil for position beyond line", func() {
			Expect(lsp.GetWordRangeAtPosition("hi", protocol.Position{Line: 0, Character: 10})).To(BeNil())
		})
	})
})

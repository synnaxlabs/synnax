// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("FindCompletion", func() {
	var items []protocol.CompletionItem

	BeforeEach(func() {
		items = []protocol.CompletionItem{
			{Label: "sensor", Detail: protocol.NewOptional("chan f32"), Kind: protocol.CompletionItemKindVariable},
			{Label: "pressure", Detail: protocol.NewOptional("chan f64"), Kind: protocol.CompletionItemKindVariable},
			{Label: "len", Detail: protocol.NewOptional("func"), Kind: protocol.CompletionItemKindFunction},
		}
	})

	It("should find an existing completion item by label", func() {
		item := MustBeOk(testutil.FindCompletion(items, "sensor"))
		Expect(item.Label).To(Equal("sensor"))
		Expect(testutil.ItemDetail(item)).To(Equal("chan f32"))
	})

	It("should return false for a non-existent label", func() {
		_, found := testutil.FindCompletion(items, "nonexistent")
		Expect(found).To(BeFalse())
	})

	It("should find the correct item when multiple items exist", func() {
		item := MustBeOk(testutil.FindCompletion(items, "len"))
		Expect(item.Kind).To(Equal(protocol.CompletionItemKindFunction))
	})

	It("should return false for an empty label", func() {
		_, found := testutil.FindCompletion(items, "")
		Expect(found).To(BeFalse())
	})

	It("should return false when items slice is empty", func() {
		_, found := testutil.FindCompletion([]protocol.CompletionItem{}, "sensor")
		Expect(found).To(BeFalse())
	})

	It("should match the exact label and not a prefix", func() {
		_, found := testutil.FindCompletion(items, "sens")
		Expect(found).To(BeFalse())
	})
})

var _ = Describe("HasCompletion", func() {
	var items []protocol.CompletionItem

	BeforeEach(func() {
		items = []protocol.CompletionItem{
			{Label: "sensor", Detail: protocol.NewOptional("chan f32")},
			{Label: "pressure", Detail: protocol.NewOptional("chan f64")},
			{Label: "now", Detail: protocol.NewOptional("func")},
		}
	})

	It("should return true for an existing label", func() {
		Expect(testutil.HasCompletion(items, "sensor")).To(BeTrue())
	})

	It("should return false for a non-existent label", func() {
		Expect(testutil.HasCompletion(items, "temperature")).To(BeFalse())
	})

	It("should return false for an empty items slice", func() {
		Expect(testutil.HasCompletion([]protocol.CompletionItem{}, "sensor")).To(BeFalse())
	})

	It("should return true for the last item in the slice", func() {
		Expect(testutil.HasCompletion(items, "now")).To(BeTrue())
	})

	It("should not match partial labels", func() {
		Expect(testutil.HasCompletion(items, "press")).To(BeFalse())
	})

	It("should be case-sensitive", func() {
		Expect(testutil.HasCompletion(items, "Sensor")).To(BeFalse())
	})
})

var _ = Describe("HoverContents", func() {
	It("should return the empty string for a nil hover", func() {
		Expect(testutil.HoverContents(nil)).To(Equal(""))
	})

	It("should extract the value from markup content", func() {
		hover := &protocol.Hover{Contents: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "**sensor** chan f32",
		}}
		Expect(testutil.HoverContents(hover)).To(Equal("**sensor** chan f32"))
	})

	It("should extract a plain string", func() {
		hover := &protocol.Hover{Contents: protocol.String("sensor chan f32")}
		Expect(testutil.HoverContents(hover)).To(Equal("sensor chan f32"))
	})

	It("should return the empty string when contents are unset", func() {
		Expect(testutil.HoverContents(&protocol.Hover{})).To(Equal(""))
	})
})

var _ = Describe("ItemDetail", func() {
	It("should return the detail when set", func() {
		item := protocol.CompletionItem{Detail: protocol.NewOptional("chan f32")}
		Expect(testutil.ItemDetail(item)).To(Equal("chan f32"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.ItemDetail(protocol.CompletionItem{})).To(Equal(""))
	})
})

var _ = Describe("ItemInsertText", func() {
	It("should return the insert text when set", func() {
		item := protocol.CompletionItem{InsertText: protocol.NewOptional("sensor")}
		Expect(testutil.ItemInsertText(item)).To(Equal("sensor"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.ItemInsertText(protocol.CompletionItem{})).To(Equal(""))
	})
})

var _ = Describe("ItemTextEdit", func() {
	It("should return a plain text edit", func() {
		edit := &protocol.TextEdit{
			Range:   protocol.Range{End: protocol.Position{Character: 3}},
			NewText: "sensor",
		}
		item := protocol.CompletionItem{TextEdit: edit}
		Expect(testutil.ItemTextEdit(item)).To(BeIdenticalTo(edit))
	})

	It("should return nil when unset", func() {
		Expect(testutil.ItemTextEdit(protocol.CompletionItem{})).To(BeNil())
	})

	It("should return nil for an insert-replace edit", func() {
		item := protocol.CompletionItem{
			TextEdit: &protocol.InsertReplaceEdit{NewText: "sensor"},
		}
		Expect(testutil.ItemTextEdit(item)).To(BeNil())
	})
})

var _ = Describe("DiagnosticCode", func() {
	It("should return a string code", func() {
		d := protocol.Diagnostic{Code: protocol.String("ARC001")}
		Expect(testutil.DiagnosticCode(d)).To(Equal("ARC001"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.DiagnosticCode(protocol.Diagnostic{})).To(Equal(""))
	})

	It("should return the empty string for a numeric code", func() {
		d := protocol.Diagnostic{Code: protocol.Integer(42)}
		Expect(testutil.DiagnosticCode(d)).To(Equal(""))
	})
})

var _ = Describe("DiagnosticMessage", func() {
	It("should return a plain string message", func() {
		d := protocol.Diagnostic{Message: protocol.String("undefined symbol")}
		Expect(testutil.DiagnosticMessage(d)).To(Equal("undefined symbol"))
	})

	It("should extract the value from a markup message", func() {
		d := protocol.Diagnostic{Message: &protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: "undefined **symbol**",
		}}
		Expect(testutil.DiagnosticMessage(d)).To(Equal("undefined **symbol**"))
	})

	It("should return the empty string when unset", func() {
		Expect(testutil.DiagnosticMessage(protocol.Diagnostic{})).To(Equal(""))
	})
})

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
			{Label: "sensor", Detail: "chan f32", Kind: protocol.CompletionItemKindVariable},
			{Label: "pressure", Detail: "chan f64", Kind: protocol.CompletionItemKindVariable},
			{Label: "len", Detail: "func", Kind: protocol.CompletionItemKindFunction},
		}
	})

	It("should find an existing completion item by label", func() {
		item := MustBeOk(testutil.FindCompletion(items, "sensor"))
		Expect(item.Label).To(Equal("sensor"))
		Expect(item.Detail).To(Equal("chan f32"))
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
			{Label: "sensor", Detail: "chan f32"},
			{Label: "pressure", Detail: "chan f64"},
			{Label: "now", Detail: "func"},
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

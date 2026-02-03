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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Completion", func() {
	var (
		server *lsp.Server
		ctx    context.Context
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		ctx = context.Background()
		server = MustSucceed(lsp.New())
		server.SetClient(&MockClient{})
		uri = "file:///test.arc"
	})

	Describe("Basic Completion", func() {
		It("should return built-in completions", func() {
			content := "func test() {\n    i\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 5)
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))
		})
	})

	Describe("Context-Aware Completion", func() {
		It("should return empty completions in single-line comment", func() {
			content := "// comment here"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 10)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return empty completions in multi-line comment", func() {
			content := "/* multi\nline\ncomment */"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 2)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return only types in type annotation position", func() {
			content := "func foo(x "
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 11)
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))

			for _, item := range completions.Items {
				Expect(item.Kind).To(Equal(protocol.CompletionItemKindClass),
					"Expected only type completions, got: %s (kind: %v)", item.Label, item.Kind)
			}

			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' keyword in type annotation context")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' keyword in type annotation context")
		})

		It("should return types matching prefix in type annotation position", func() {
			content := "func foo(x i"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 12)
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))

			for _, item := range completions.Items {
				Expect(item.Label).To(HavePrefix("i"), "Expected items with 'i' prefix, got: %s", item.Label)
			}
		})

		It("should not show keywords in expression context", func() {
			content := "x := "
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 5)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' keyword in expression context")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' keyword in expression context")
		})

		It("should show functions and values in expression context", func() {
			content := "x := "
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 5)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "len")).To(BeTrue(), "Should show 'len' function in expression context")
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue(), "Should show 'now' function in expression context")
		})

		It("should show keywords at statement start", func() {
			content := "func foo() { "
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 13)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "if")).To(BeTrue(), "Should show 'if' keyword at statement start")
			Expect(HasCompletion(completions.Items, "return")).To(BeTrue(), "Should show 'return' keyword at statement start")
		})

		It("should not show types at statement start", func() {
			content := "func foo() { "
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 0, 13)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "i32")).To(BeFalse(), "Should not show 'i32' type at statement start")
		})
	})

	Describe("GlobalResolver", func() {
		It("should include global variables from GlobalResolver in completion", func() {
			// Create a mock GlobalResolver with a global variable
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			// Create server with GlobalResolver
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			// Use the same pattern as hover test - valid Arc code
			content := "func test() i32 {\n    return myGlobal\n}"
			OpenDocument(server, ctx, uri, content)

			// Request completion in the middle of typing "myGlobal" -> "myG|"
			// Simulating user typing "myG" and requesting completion
			completions := Completion(server, ctx, uri, 1, 14) // after "myG" in "return myGlobal"
			Expect(completions).ToNot(BeNil())

			// Check that myGlobal is in the completion list
			item, found := FindCompletion(completions.Items, "myGlobal")
			Expect(found).To(BeTrue(), "Expected to find 'myGlobal' in completion items")
			Expect(item.Kind).To(Equal(protocol.CompletionItemKindVariable))
			Expect(item.Detail).To(Equal("i32"))
		})

		It("should not show GlobalResolver symbols when prefix doesn't match", func() {
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return xyz\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 14)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "myGlobal")).To(BeFalse(), "Expected NOT to find 'myGlobal' in completion items when prefix doesn't match")
		})
	})

	Describe("Config Parameter Completion", func() {
		var globalResolver symbol.MapResolver

		BeforeEach(func() {
			globalResolver = symbol.MapResolver{
				"myTask": symbol.Symbol{
					Name: "myTask",
					Kind: symbol.KindFunction,
					Type: types.Function(types.FunctionProperties{
						Config: types.Params{
							{Name: "threshold", Type: types.F64()},
							{Name: "timeout", Type: types.I64()},
							{Name: "channel", Type: types.Chan(types.F64())},
						},
					}),
				},
				"sensorCh": symbol.Symbol{
					Name: "sensorCh",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
				},
			}
		})

		It("should suggest all config parameters in empty config block", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{}\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 11)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeTrue(), "Should suggest 'threshold' parameter")
			Expect(HasCompletion(completions.Items, "timeout")).To(BeTrue(), "Should suggest 'timeout' parameter")
			Expect(HasCompletion(completions.Items, "channel")).To(BeTrue(), "Should suggest 'channel' parameter")
		})

		It("should filter out already-provided parameters", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{threshold=1.0, timeout=100}\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 26)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeFalse(), "Should NOT suggest already-provided 'threshold' parameter")
			Expect(HasCompletion(completions.Items, "channel")).To(BeTrue(), "Should still suggest 'channel' parameter")
		})

		It("should filter by prefix when typing parameter name", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{threshold=1.0}\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 13)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeTrue(), "Should suggest 'threshold' matching prefix 'th'")
			Expect(HasCompletion(completions.Items, "timeout")).To(BeFalse(), "Should NOT suggest 'timeout' not matching prefix 'th'")
		})

		It("should show type details for config parameters", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{}\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 11)
			Expect(completions).ToNot(BeNil())

			thresholdItem, found := FindCompletion(completions.Items, "threshold")
			Expect(found).To(BeTrue())
			Expect(thresholdItem.Detail).To(Equal("f64"))
			Expect(thresholdItem.Kind).To(Equal(protocol.CompletionItemKindProperty))
		})

		It("should suggest channel symbols for chan type parameters", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{channel=sensorCh}\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 1, 19)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "sensorCh")).To(BeTrue(), "Should suggest 'sensorCh' channel for chan type parameter")
		})
	})

	Describe("Stage Body Completion", func() {
		var globalResolver symbol.MapResolver

		BeforeEach(func() {
			globalResolver = symbol.MapResolver{
				"vent_vlv_cmd": symbol.Symbol{
					Name: "vent_vlv_cmd",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   1,
				},
				"press_vlv_cmd": symbol.Symbol{
					Name: "press_vlv_cmd",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   2,
				},
				"press_pt": symbol.Symbol{
					Name: "press_pt",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F64()),
					ID:   3,
				},
			}
		})

		It("should suggest channels inside stage body", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        \n    }\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 2, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' channel inside stage")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue(), "Should suggest 'press_vlv_cmd' channel inside stage")
			Expect(HasCompletion(completions.Items, "press_pt")).To(BeTrue(), "Should suggest 'press_pt' channel inside stage")
		})

		It("should suggest channels with prefix filter inside stage body", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        v\n    }\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 2, 9)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' matching prefix 'v'")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeFalse(), "Should NOT suggest 'press_vlv_cmd' not matching prefix 'v'")
		})

		It("should suggest channels inside stage after flow statement", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        1 -> vent_vlv_cmd,\n        \n    }\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 3, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' channel")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue(), "Should suggest 'press_vlv_cmd' channel")
		})

		It("should suggest channels with prefix after flow statement", func() {
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        1 -> vent_vlv_cmd,\n        v\n    }\n}"
			OpenDocument(server, ctx, uri, content)

			completions := Completion(server, ctx, uri, 3, 9)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' matching prefix 'v'")
		})
	})
})

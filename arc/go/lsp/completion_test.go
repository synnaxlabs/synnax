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
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Completion", func() {
	var (
		server *lsp.Server
		docURI uri.URI
	)

	BeforeEach(func() {
		server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
		server.SetClient(&MockClient{})
		docURI = "file:///test.arc"
	})

	Describe("Basic Completion", func() {
		It("should return built-in completions", func(ctx SpecContext) {
			content := "func test() {\n    i\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 5)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).ToNot(BeEmpty())
		})
	})

	Describe("Context-Aware Completion", func() {
		It("should return empty completions in single-line comment", func(ctx SpecContext) {
			content := "// comment here"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 10)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return empty completions in multi-line comment", func(ctx SpecContext) {
			content := "/* multi\nline\ncomment */"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 2)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return only types in type annotation position", func(ctx SpecContext) {
			content := "func foo(x "
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 11)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).ToNot(BeEmpty())

			for _, item := range completions.Items {
				Expect(item.Kind).To(Equal(protocol.CompletionItemKindClass),
					"Expected only type completions, got: %s (kind: %v)", item.Label, item.Kind)
			}

			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' keyword in type annotation context")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' keyword in type annotation context")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeFalse(), "Should not show 'sequence' keyword in type annotation context")
			Expect(HasCompletion(completions.Items, "stage")).To(BeFalse(), "Should not show 'stage' keyword in type annotation context")
			Expect(HasCompletion(completions.Items, "next")).To(BeFalse(), "Should not show 'next' keyword in type annotation context")
		})

		It("should return types matching prefix in type annotation position", func(ctx SpecContext) {
			content := "func foo(x i"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 12)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).ToNot(BeEmpty())

			for _, item := range completions.Items {
				Expect(item.Label).To(HavePrefix("i"), "Expected items with 'i' prefix, got: %s", item.Label)
			}
		})

		It("should not show keywords in expression context", func(ctx SpecContext) {
			content := "x := "
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' keyword in expression context")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' keyword in expression context")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeFalse(), "Should not show 'sequence' keyword in expression context")
			Expect(HasCompletion(completions.Items, "stage")).To(BeFalse(), "Should not show 'stage' keyword in expression context")
			Expect(HasCompletion(completions.Items, "next")).To(BeFalse(), "Should not show 'next' keyword in expression context")
		})

		It("should show functions and values in expression context", func(ctx SpecContext) {
			content := "x := "
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "len")).To(BeTrue(), "Should show 'len' function in expression context")
			Expect(HasCompletion(completions.Items, "time.now")).To(BeTrue(), "Should show 'time.now' function in expression context")
		})

		It("should show function keywords at statement start inside func body", func(ctx SpecContext) {
			content := "func foo() {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "if")).To(BeTrue(), "Should show 'if' keyword at statement start in func body")
			Expect(HasCompletion(completions.Items, "return")).To(BeTrue(), "Should show 'return' keyword at statement start in func body")
			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' inside func body")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeFalse(), "Should not show 'sequence' inside func body")
			Expect(HasCompletion(completions.Items, "stage")).To(BeFalse(), "Should not show 'stage' inside func body")
			Expect(HasCompletion(completions.Items, "next")).To(BeFalse(), "Should not show 'next' inside func body")
		})

		It("should show top-level keywords at top level", func(ctx SpecContext) {
			content := "seq"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 3)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "sequence")).To(BeTrue(), "Should show 'sequence' keyword at top level")
			Expect(HasCompletion(completions.Items, "i32")).To(BeFalse(), "Should not show 'i32' type at top level")
			Expect(HasCompletion(completions.Items, "f64")).To(BeFalse(), "Should not show 'f64' type at top level")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' at top level")
			Expect(HasCompletion(completions.Items, "return")).To(BeFalse(), "Should not show 'return' at top level")
			Expect(HasCompletion(completions.Items, "stage")).To(BeFalse(), "Should not show 'stage' at top level")
			Expect(HasCompletion(completions.Items, "next")).To(BeFalse(), "Should not show 'next' at top level")
		})

		It("should insert a sequence snippet without a nested stage block", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "seq")
			completions := Completion(server, ctx, docURI, 0, 3)
			item, found := FindCompletion(completions.Items, "sequence")
			Expect(found).To(BeTrue())
			Expect(ItemInsertText(item)).ToNot(ContainSubstring("stage"),
				"the sequence snippet should not pre-populate a nested stage block")
			Expect(ItemInsertText(item)).To(Equal("sequence ${1:name} {\n\t$0\n}"))
		})

		It("should show func keyword at top level", func(ctx SpecContext) {
			content := "fu"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 2)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "func")).To(BeTrue(), "Should show 'func' keyword at top level")
		})

		It("should show only stage keyword inside a sequence body", func(ctx SpecContext) {
			content := "sequence main {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "stage")).To(BeTrue(), "Should show 'stage' keyword inside sequence body")
			Expect(HasCompletion(completions.Items, "next")).To(BeFalse(), "Should not show 'next' inside sequence body")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' inside sequence body")
			Expect(HasCompletion(completions.Items, "return")).To(BeFalse(), "Should not show 'return' inside sequence body")
			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' inside sequence body")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeTrue(), "Should show 'sequence' inside sequence body (nested sequences)")
			Expect(HasCompletion(completions.Items, "i32")).To(BeFalse(), "Should not show 'i32' type inside sequence body")
			Expect(HasCompletion(completions.Items, "f64")).To(BeFalse(), "Should not show 'f64' type inside sequence body")
		})

		It("should show channels and flow-compatible functions inside a sequence body", func(ctx SpecContext) {
			// A sequence body accepts flow statements and single
			// invocations, so channels, ExecFlow/ExecBoth functions, and
			// their module-qualified deep-search forms must surface.
			channels := []symbol.Symbol{
				{Name: "temperature_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, channels...)
			}}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)
			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(HasCompletion(completions.Items, "temperature_sensor")).To(BeTrue(),
				"channel references are valid flow nodes inside a sequence body")
			Expect(HasCompletion(completions.Items, "interval")).To(BeTrue(),
				"the bare alias for the flow-only time.interval should appear inside a sequence body")
			Expect(HasCompletion(completions.Items, "time.interval")).To(BeTrue(),
				"deep-search should surface time.interval as a qualified completion inside a sequence body")
			Expect(HasCompletion(completions.Items, "pow")).To(BeFalse(),
				"the WASM-only math.pow should not appear in a flow context")
		})

		It("should not offer completions at a sequence declaration name slot", func(ctx SpecContext) {
			// Prefix `ma` would otherwise match `math` (a module in scope);
			// at a declaration-name slot the user is introducing an
			// identifier, so nothing should be suggested.
			OpenArcDocument(server, ctx, docURI, "sequence ma")
			completions := Completion(server, ctx, docURI, 0, 11)
			Expect(completions.Items).To(BeEmpty(),
				"the slot for a new sequence name introduces an identifier — no existing symbols should be offered")
		})

		It("should not offer completions at a stage declaration name slot", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "sequence main {\n    stage ma\n}")
			completions := Completion(server, ctx, docURI, 1, 12)
			Expect(completions.Items).To(BeEmpty(),
				"the slot for a new stage name introduces an identifier — no existing symbols should be offered")
		})

		It("should suggest only modules after the 'import' keyword", func(ctx SpecContext) {
			// After `import ` the next identifier is a module path, so the
			// dropdown must contain only modules — channels, functions, and
			// other unrelated symbols are not valid import targets.
			channels := []symbol.Symbol{
				{Name: "temperature_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, channels...)
			}}))
			server.SetClient(&MockClient{})

			OpenArcDocument(server, ctx, docURI, "import ")
			completions := Completion(server, ctx, docURI, 0, 7)
			Expect(HasCompletion(completions.Items, "temperature_sensor")).To(BeFalse(),
				"channels are not modules; must not appear in import-path position")
			Expect(HasCompletion(completions.Items, "interval")).To(BeFalse(),
				"bare module-member aliases are not modules; must not appear in import-path position")
			Expect(HasCompletion(completions.Items, "math")).To(BeTrue(),
				"modules must be suggested in import-path position")
			Expect(HasCompletion(completions.Items, "time")).To(BeTrue(),
				"modules must be suggested in import-path position")
		})

		It("should suggest only modules matching the prefix when partially typed after 'import'", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import ma")
			completions := Completion(server, ctx, docURI, 0, 9)
			Expect(HasCompletion(completions.Items, "math")).To(BeTrue(),
				"matching modules must be suggested in import-path position")
			Expect(HasCompletion(completions.Items, "time")).To(BeFalse(),
				"non-matching modules must not appear")
		})

		It("should not surface internal modules in import-path completion", func(ctx SpecContext) {
			// The `error` module is marked Internal — its members are
			// emitted by lowering passes (out-of-bounds checks, etc.), not
			// called from user source — so it must not appear as an
			// importable module.
			OpenArcDocument(server, ctx, docURI, "import er")
			completions := Completion(server, ctx, docURI, 0, 9)
			Expect(HasCompletion(completions.Items, "error")).To(BeFalse(),
				"the internal `error` module must not be offered as an import target")
		})

		It("should not treat a new line after 'import math' as an import-path slot", func(ctx SpecContext) {
			// A bare identifier on a new line below a complete import
			// statement is normal code, not part of the import — the
			// dropdown must surface general completions (channels, bare
			// aliases, modules, etc.), not be restricted to modules only.
			channels := []symbol.Symbol{
				{Name: "temperature_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, channels...)
			}}))
			server.SetClient(&MockClient{})

			OpenArcDocument(server, ctx, docURI, "import math\nte")
			completions := Completion(server, ctx, docURI, 1, 2)
			Expect(HasCompletion(completions.Items, "temperature_sensor")).To(BeTrue(),
				"channels must appear in a normal statement position even when a prior line is an import")
		})

		It("should not offer completions at a func declaration name slot", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func ma")
			completions := Completion(server, ctx, docURI, 0, 7)
			Expect(completions.Items).To(BeEmpty(),
				"the slot for a new func name introduces an identifier — no existing symbols should be offered")
		})

		It("should show next keyword inside a stage body", func(ctx SpecContext) {
			content := "sequence main {\n    stage first {\n        \n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "next")).To(BeTrue(), "Should show 'next' keyword inside stage body")
			Expect(HasCompletion(completions.Items, "stage")).To(BeFalse(), "Should not show 'stage' inside stage body")
			Expect(HasCompletion(completions.Items, "if")).To(BeFalse(), "Should not show 'if' inside stage body")
			Expect(HasCompletion(completions.Items, "return")).To(BeFalse(), "Should not show 'return' inside stage body")
			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' inside stage body")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeTrue(), "Should show 'sequence' inside stage body (inline sequences)")
		})

		It("should not show types at statement start", func(ctx SpecContext) {
			content := "func foo() {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "i32")).To(BeFalse(), "Should not show 'i32' type at statement start")
		})
	})

	Describe("Nested If Inside Function", func() {
		It("should show function keywords inside nested if block", func(ctx SpecContext) {
			content := "func foo() {\n    if x > 0 {\n        \n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "if")).To(BeTrue(), "Should show 'if' inside nested if block")
			Expect(HasCompletion(completions.Items, "return")).To(BeTrue(), "Should show 'return' inside nested if block")
			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' inside nested if block")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeFalse(), "Should not show 'sequence' inside nested if block")
		})
	})

	Describe("Block Expression Completion", func() {
		It("should show function keywords in block expression", func(ctx SpecContext) {
			blockURI := uri.URI("arc://block/test")
			content := ""
			OpenArcDocument(server, ctx, blockURI, content)

			completions := Completion(server, ctx, blockURI, 0, 0)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "if")).To(BeTrue(), "Should show 'if' in block expression")
			Expect(HasCompletion(completions.Items, "return")).To(BeTrue(), "Should show 'return' in block expression")
			Expect(HasCompletion(completions.Items, "func")).To(BeFalse(), "Should not show 'func' in block expression")
			Expect(HasCompletion(completions.Items, "sequence")).To(BeFalse(), "Should not show 'sequence' in block expression")
		})
	})

	Describe("GlobalResolver", func() {
		It("should include global variables from GlobalResolver in completion", func(ctx SpecContext) {
			// Create a mock GlobalResolver with a global variable
			globalResolver := []symbol.Symbol{{
				Name: "myGlobal",
				Type: types.I32(),
				Kind: symbol.KindVariable,
			}}

			// Create server with GlobalResolver
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			// Use the same pattern as hover test - valid Arc code
			content := "func test() i32 {\n    return myGlobal\n}"
			OpenArcDocument(server, ctx, docURI, content)

			// Request completion in the middle of typing "myGlobal" -> "myG|"
			// Simulating user typing "myG" and requesting completion
			completions := Completion(server, ctx, docURI, 1, 14) // after "myG" in "return myGlobal"
			Expect(completions).ToNot(BeNil())

			// Check that myGlobal is in the completion list
			item, found := FindCompletion(completions.Items, "myGlobal")
			Expect(found).To(BeTrue(), "Expected to find 'myGlobal' in completion items")
			Expect(item.Kind).To(Equal(protocol.CompletionItemKindVariable))
			Expect(ItemDetail(item)).To(Equal("i32"))
		})

		It("should not show GlobalResolver symbols when prefix doesn't match", func(ctx SpecContext) {
			globalResolver := []symbol.Symbol{{
				Name: "myGlobal",
				Type: types.I32(),
				Kind: symbol.KindVariable,
			}}

			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return xyz\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 14)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "myGlobal")).To(BeFalse(), "Expected NOT to find 'myGlobal' in completion items when prefix doesn't match")
		})
	})

	Describe("Parenthesized Expression Completion", func() {
		It("should suggest channels inside parenthesized expression after return", func(ctx SpecContext) {
			globalResolver := []symbol.Symbol{{
				Name: "output_sensor",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   1,
			}}

			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() f64 {\n    return (o\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 13)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "output_sensor")).To(BeTrue(),
				"Should suggest 'output_sensor' channel inside parenthesized return expression")
		})
	})

	Describe("Input Parameter Completion", func() {
		var globalResolver []symbol.Symbol

		BeforeEach(func() {
			globalResolver = []symbol.Symbol{{
				Name: "myTask",
				Kind: symbol.KindFunction,
				Type: types.Function(types.FunctionProperties{
					Inputs: types.Params{
						{Name: "threshold", Type: types.F64()},
						{Name: "timeout", Type: types.I64()},
						{Name: "channel", Type: types.Chan(types.F64())},
					},
				}),
			}, {
				Name: "sensorCh",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			}}
		})

		It("should suggest all input parameters in empty input block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{}\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 11)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeTrue(), "Should suggest 'threshold' parameter")
			Expect(HasCompletion(completions.Items, "timeout")).To(BeTrue(), "Should suggest 'timeout' parameter")
			Expect(HasCompletion(completions.Items, "channel")).To(BeTrue(), "Should suggest 'channel' parameter")
		})

		It("should filter out already-provided parameters", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{threshold=1.0, timeout=100}\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 26)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeFalse(), "Should NOT suggest already-provided 'threshold' parameter")
			Expect(HasCompletion(completions.Items, "channel")).To(BeTrue(), "Should still suggest 'channel' parameter")
		})

		It("should filter by prefix when typing parameter name", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{threshold=1.0}\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 13)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "threshold")).To(BeTrue(), "Should suggest 'threshold' matching prefix 'th'")
			Expect(HasCompletion(completions.Items, "timeout")).To(BeFalse(), "Should NOT suggest 'timeout' not matching prefix 'th'")
		})

		It("should show type details for input parameters", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{}\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 11)
			Expect(completions).ToNot(BeNil())

			thresholdItem, found := FindCompletion(completions.Items, "threshold")
			Expect(found).To(BeTrue())
			Expect(ItemDetail(thresholdItem)).To(Equal("f64"))
			Expect(thresholdItem.Kind).To(Equal(protocol.CompletionItemKindProperty))
		})

		It("should suggest channel symbols for chan type parameters", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    myTask{channel=sensorCh}\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 19)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "sensorCh")).To(BeTrue(), "Should suggest 'sensorCh' channel for chan type parameter")
		})
	})

	Describe("Authority Block Completion", func() {
		var globalResolver []symbol.Symbol

		BeforeEach(func() {
			globalResolver = []symbol.Symbol{{
				Name: "vent_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   1,
			}, {
				Name: "press_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   2,
			}, {
				Name: "myGlobal",
				Kind: symbol.KindVariable,
				Type: types.I32(),
			}}
		})

		It("should suggest authority keyword at top level", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "auth"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 4)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "authority")).To(BeTrue())
		})

		It("should suggest channels inside authority block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "authority (\n    200\n    \n)"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue())
		})

		It("should not suggest non-channel symbols inside authority block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "authority (\n    200\n    \n)"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "myGlobal")).To(BeFalse())
		})

		It("should filter out already-listed channels", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "authority (\n    200\n    vent_vlv_cmd 100\n    \n)"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 3, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeFalse())
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue())
		})

		It("should filter by prefix inside authority block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "authority (\n    200\n    v\n)"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 5)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeFalse())
		})
	})

	Describe("Loop Keyword Completion", func() {
		It("should show for keyword at statement start inside func body", func(ctx SpecContext) {
			content := "func foo() {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "for")).To(BeTrue(), "Should show 'for' keyword at statement start in func body")
			Expect(HasCompletion(completions.Items, "break")).To(BeTrue(), "Should show 'break' keyword at statement start in func body")
			Expect(HasCompletion(completions.Items, "continue")).To(BeTrue(), "Should show 'continue' keyword at statement start in func body")
		})

		It("should not show for keyword at top level", func(ctx SpecContext) {
			content := "fo"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 2)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "for")).To(BeFalse(), "Should not show 'for' at top level")
			Expect(HasCompletion(completions.Items, "break")).To(BeFalse(), "Should not show 'break' at top level")
			Expect(HasCompletion(completions.Items, "continue")).To(BeFalse(), "Should not show 'continue' at top level")
		})

		It("should not show loop keywords inside sequence body", func(ctx SpecContext) {
			content := "sequence main {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "for")).To(BeFalse(), "Should not show 'for' inside sequence body")
			Expect(HasCompletion(completions.Items, "break")).To(BeFalse(), "Should not show 'break' inside sequence body")
			Expect(HasCompletion(completions.Items, "continue")).To(BeFalse(), "Should not show 'continue' inside sequence body")
		})

		It("should not show loop keywords in expression context", func(ctx SpecContext) {
			content := "func foo() {\n    x := \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "for")).To(BeFalse(), "Should not show 'for' in expression context")
			Expect(HasCompletion(completions.Items, "break")).To(BeFalse(), "Should not show 'break' in expression context")
			Expect(HasCompletion(completions.Items, "continue")).To(BeFalse(), "Should not show 'continue' in expression context")
		})

		It("should show for snippet with correct insert text", func(ctx SpecContext) {
			content := "func foo() {\n    fo\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 6)
			Expect(completions).ToNot(BeNil())

			item, found := FindCompletion(completions.Items, "for")
			Expect(found).To(BeTrue())
			Expect(ItemInsertText(item)).To(ContainSubstring("range"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
		})

		It("should show range function in expression context", func(ctx SpecContext) {
			content := "func foo() {\n    x := r\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 10)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "range")).To(BeTrue(), "Should show 'range' function in expression context")
		})
	})

	Describe("Stage Body Completion", func() {
		var globalResolver []symbol.Symbol

		BeforeEach(func() {
			globalResolver = []symbol.Symbol{{
				Name: "vent_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   1,
			}, {
				Name: "press_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   2,
			}, {
				Name: "press_pt",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
				ID:   3,
			}}
		})

		It("should suggest channels inside stage body", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        \n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' channel inside stage")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue(), "Should suggest 'press_vlv_cmd' channel inside stage")
			Expect(HasCompletion(completions.Items, "press_pt")).To(BeTrue(), "Should suggest 'press_pt' channel inside stage")
		})

		It("should suggest channels with prefix filter inside stage body", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        v\n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 2, 9)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' matching prefix 'v'")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeFalse(), "Should NOT suggest 'press_vlv_cmd' not matching prefix 'v'")
		})

		It("should suggest channels inside stage after flow statement", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        1 -> vent_vlv_cmd\n        \n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 3, 8)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' channel")
			Expect(HasCompletion(completions.Items, "press_vlv_cmd")).To(BeTrue(), "Should suggest 'press_vlv_cmd' channel")
		})

		It("should suggest channels with prefix after flow statement", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, globalResolver...) }}))
			server.SetClient(&MockClient{})

			content := "sequence main {\n    stage first {\n        1 -> vent_vlv_cmd\n        v\n    }\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 3, 9)
			Expect(completions).ToNot(BeNil())

			Expect(HasCompletion(completions.Items, "vent_vlv_cmd")).To(BeTrue(), "Should suggest 'vent_vlv_cmd' matching prefix 'v'")
		})
	})

	Describe("Module Qualified Completion", func() {
		var channelsWithChannels []symbol.Symbol

		BeforeEach(func() {
			channelsWithChannels = []symbol.Symbol{
				{Name: "sy_node_1_metrics_time", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
				{Name: "temperature_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
		})
		_ = channelsWithChannels

		It("Should return module members for 'math.a' prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "math.a"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 6)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "avg")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "add")).To(BeFalse(),
				"Internal symbol math.add should not appear in module-qualified completion")
		})

		It("Should return all members for bare 'math.' prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "math."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "avg")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "pow")).To(BeFalse(),
				"Internal symbol math.pow should not appear in module-qualified completion")
			Expect(HasCompletion(completions.Items, "add")).To(BeFalse(),
				"Internal symbol math.add should not appear in module-qualified completion")
		})

		It("Should return only WASM time members for 'time.' prefix in func block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    time.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue(),
				"WASM function time.now should appear in func block")
			Expect(HasCompletion(completions.Items, "interval")).To(BeFalse(),
				"Flow function time.interval should not appear in func block")
			Expect(HasCompletion(completions.Items, "wait")).To(BeFalse(),
				"Flow function time.wait should not appear in func block")
		})

		It("Should return control.set_authority for 'control.' prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "control."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 8)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "set_authority")).To(BeTrue())
		})

		It("Should return control.set_authority for 'control.set_a' prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "control.set_a"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 13)
			Expect(completions).ToNot(BeNil())
			item, found := FindCompletion(completions.Items, "set_authority")
			Expect(found).To(BeTrue())
			Expect(item.FilterText).To(Equal(protocol.NewOptional("control.set_authority")))
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("control.set_authority{$0}"))
		})

		It("Should suggest module names at top-level when typing a partial module name", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			OpenArcDocument(server, ctx, docURI, "trig => contr")
			completionsControl := Completion(server, ctx, docURI, 0, 13)
			Expect(completionsControl).ToNot(BeNil())

			OpenArcDocument(server, ctx, docURI, "trig => mat")
			completionsMath := Completion(server, ctx, docURI, 0, 11)
			Expect(completionsMath).ToNot(BeNil())

			controlHasModule := HasCompletion(completionsControl.Items, "control")
			mathHasModule := HasCompletion(completionsMath.Items, "math")

			Expect(controlHasModule).To(Equal(mathHasModule),
				"control should be suggested as a top-level identifier iff math is (i.e., they should behave consistently)")
		})

		It("Should not surface internal members like 'panic' for 'error.' prefix", func(ctx SpecContext) {
			// The `error` module and its `panic` member are marked Internal
			// because panic is emitted by lowering passes (out-of-bounds
			// checks, etc.), not called from user source. Completion must
			// not surface either to user code.
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    error.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 10)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "panic")).To(BeFalse(),
				"internal module members must not appear in user-facing completions")
		})

		It("Should set FilterText with qualified name", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "math."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())

			item, found := FindCompletion(completions.Items, "avg")
			Expect(found).To(BeTrue())
			Expect(item.FilterText).To(Equal(protocol.NewOptional("math.avg")))
		})

		It("Should set TextEdit that replaces the full module prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "math."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())

			item, found := FindCompletion(completions.Items, "avg")
			Expect(found).To(BeTrue())
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("math.avg{$0}"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
			Expect(ItemTextEdit(item).Range.Start.Character).To(Equal(uint32(0)))
			Expect(ItemTextEdit(item).Range.End.Character).To(Equal(uint32(5)))
		})

		It("Should exclude channel symbols from module-qualified results", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, channelsWithChannels...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    time.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "sy_node_1_metrics_time")).To(BeFalse(),
				"Should not show channel names in module-qualified completions")
		})

		It("Should exclude channels even with partial member prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, channelsWithChannels...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    time.n\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 10)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue())
			Expect(HasCompletion(completions.Items, "sy_node_1_metrics_time")).To(BeFalse())
			Expect(HasCompletion(completions.Items, "temperature_sensor")).To(BeFalse())
		})

		It("Should return nothing for unknown module prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    fake.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("Should not affect unqualified completions", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil, channelsWithChannels...) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    t\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 5)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "temperature_sensor")).To(BeTrue(),
				"Unqualified prefix should still show channels")
		})
	})

	Describe("ExecContext Filtering", func() {
		It("should not show internal symbols inside func block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    math.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "pow")).To(BeFalse(),
				"Internal symbol math.pow should not appear in module-qualified completion")
		})

		It("should not show flow functions inside func block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    time.\n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 9)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue(),
				"WASM function time.now should appear in func block")
			Expect(HasCompletion(completions.Items, "interval")).To(BeFalse(),
				"Flow function time.interval should not appear in func block")
			Expect(HasCompletion(completions.Items, "wait")).To(BeFalse(),
				"Flow function time.wait should not appear in func block")
		})

		It("should show flow functions at top level", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "time."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "interval")).To(BeTrue(),
				"Flow function time.interval should appear at top level")
			Expect(HasCompletion(completions.Items, "wait")).To(BeTrue(),
				"Flow function time.wait should appear at top level")
		})

		It("should show ExecBoth functions at top level", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "time."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue(),
				"ExecBoth function time.now should appear at top level")
		})

		It("should not show WASM-only functions at top level", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "math."
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 0, 5)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "pow")).To(BeFalse(),
				"WASM-only function math.pow should not appear at top level")
		})

		It("should not show unqualified flow-only functions in func block", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    \n}"
			OpenArcDocument(server, ctx, docURI, content)

			completions := Completion(server, ctx, docURI, 1, 4)
			Expect(completions).ToNot(BeNil())
			Expect(HasCompletion(completions.Items, "interval")).To(BeFalse(),
				"Flow-only function interval should not appear in func block")
			Expect(HasCompletion(completions.Items, "wait")).To(BeFalse(),
				"Flow-only function wait should not appear in func block")
			Expect(HasCompletion(completions.Items, "now")).To(BeTrue(),
				"ExecBoth function now should appear in func block")
			Expect(HasCompletion(completions.Items, "pow")).To(BeFalse(),
				"Internal symbol pow should not appear in unqualified completion")
		})
	})

	Describe("Module Completion Metadata", func() {
		// When a module symbol (e.g. math, time) surfaces in autocomplete, the
		// completion item must use the Module kind and a descriptive detail
		// string rather than the "invalid" fallback from types.Type.String(),
		// and selecting an unimported module must auto-insert the
		// corresponding `import <name>` declaration at the top of the file.
		//
		// Tests use a func body for the cursor position because a bare
		// identifier at the top level is a parse error that can prevent the
		// analyzer from registering imports, and the user-reported bug
		// occurs while editing inside a body block.

		It("labels an unimported module with Module kind and a non-'invalid' detail", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 1, 7)
			Expect(completions).ToNot(BeNil())
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue(), "math module should be suggested for prefix 'mat'")
			Expect(item.Kind).To(Equal(protocol.CompletionItemKindModule))
			Expect(ItemDetail(item)).ToNot(Equal("invalid"),
				"module Detail must not be the 'invalid' fallback from types.Type.String()")
			Expect(ItemDetail(item)).To(Equal("module"))
		})

		It("labels an already-imported module with Module kind and a non-'invalid' detail", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import math\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 3, 7)
			Expect(completions).ToNot(BeNil())
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue(), "math module should be suggested for prefix 'mat'")
			Expect(item.Kind).To(Equal(protocol.CompletionItemKindModule))
			Expect(ItemDetail(item)).ToNot(Equal("invalid"))
			Expect(ItemDetail(item)).To(Equal("module"))
		})

		It("inserts a loose import statement when no imports exist", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 1, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(HaveLen(1))
			edit := item.AdditionalTextEdits[0]
			Expect(edit.NewText).To(Equal("import math\n\n"),
				"a single import must use the loose form, not the multi-line block form")
			Expect(edit.Range.Start.Line).To(Equal(uint32(0)))
			Expect(edit.Range.Start.Character).To(Equal(uint32(0)))
			Expect(edit.Range.End.Line).To(Equal(uint32(0)))
			Expect(edit.Range.End.Character).To(Equal(uint32(0)))
		})

		It("adds to an existing import block instead of creating a second block", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import (\n    time\n)\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 5, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(HaveLen(1))
			edit := item.AdditionalTextEdits[0]
			Expect(edit.NewText).To(Equal("import (\n    time\n    math\n)\n"))
			Expect(edit.Range.Start.Line).To(Equal(uint32(0)))
			Expect(edit.Range.Start.Character).To(Equal(uint32(0)))
			Expect(edit.Range.End.Line).To(Equal(uint32(2)))
			Expect(edit.Range.End.Character).To(Equal(uint32(1)))
		})

		It("consolidates a loose import statement into a single block when adding a module", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 3, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(HaveLen(1))
			edit := item.AdditionalTextEdits[0]
			Expect(edit.NewText).To(Equal("import (\n    time\n    math\n)\n"))
			Expect(edit.Range.Start.Line).To(Equal(uint32(0)))
			Expect(edit.Range.Start.Character).To(Equal(uint32(0)))
			Expect(edit.Range.End.Line).To(Equal(uint32(0)))
			Expect(edit.Range.End.Character).To(Equal(uint32(11)))
		})

		It("consolidates multiple loose import statements into one block", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\nimport control\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 4, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(HaveLen(2))
			replace := item.AdditionalTextEdits[0]
			Expect(replace.NewText).To(Equal("import (\n    time\n    control\n    math\n)\n"))
			Expect(replace.Range.Start).To(Equal(protocol.Position{Line: 0, Character: 0}))
			Expect(replace.Range.End).To(Equal(protocol.Position{Line: 0, Character: 11}))
			deleteOld := item.AdditionalTextEdits[1]
			Expect(deleteOld.NewText).To(BeEmpty())
			Expect(deleteOld.Range.Start).To(Equal(protocol.Position{Line: 1, Character: 0}))
			Expect(deleteOld.Range.End).To(Equal(protocol.Position{Line: 2, Character: 0}))
		})

		It("preserves comments sitting between two loose import statements", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\n// keep this\nimport control\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 5, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(HaveLen(2))
			replace := item.AdditionalTextEdits[0]
			Expect(replace.NewText).To(Equal("import (\n    time\n    control\n    math\n)\n"))
			Expect(replace.Range.Start).To(Equal(protocol.Position{Line: 0, Character: 0}))
			Expect(replace.Range.End).To(Equal(protocol.Position{Line: 0, Character: 11}))
			deleteOld := item.AdditionalTextEdits[1]
			Expect(deleteOld.NewText).To(BeEmpty())
			Expect(deleteOld.Range.Start).To(Equal(protocol.Position{Line: 2, Character: 0}))
			Expect(deleteOld.Range.End).To(Equal(protocol.Position{Line: 3, Character: 0}))
		})

		It("does not attach an import edit when the module is already imported", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import math\n\nfunc test() {\n    mat\n}")
			completions := Completion(server, ctx, docURI, 3, 7)
			item, found := FindCompletion(completions.Items, "math")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(BeEmpty(),
				"an already-imported module must not produce a duplicate import edit")
		})
	})

	Describe("Module Member Auto-Import", func() {
		// Selecting a module member completion (e.g. `time.now` reached via
		// either the qualified `time.` prefix or a bare prefix like `now`)
		// must auto-import the source module when it isn't already in scope,
		// the same way a bare module selection does.

		It("attaches an import edit to a qualified member when the module is not imported", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n    time.no\n}")
			completions := Completion(server, ctx, docURI, 1, 11)
			item, found := FindCompletion(completions.Items, "now")
			Expect(found).To(BeTrue(), "time.now should be suggested for prefix 'time.no'")
			Expect(item.AdditionalTextEdits).To(HaveLen(1))
			edit := item.AdditionalTextEdits[0]
			Expect(edit.NewText).To(Equal("import time\n\n"))
		})

		It("does not attach an import edit to a qualified member when the module is already imported", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\n\nfunc test() {\n    time.no\n}")
			completions := Completion(server, ctx, docURI, 3, 11)
			item, found := FindCompletion(completions.Items, "now")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(BeEmpty())
		})

		It("does not double-import when a dot-completion follows a freshly auto-imported module", func(ctx SpecContext) {
			// Reproduces the sequence: user accepts the `math` completion
			// (auto-imports it), types `.`, then triggers completion on
			// `math.`. The `avg` completion must not carry another auto-
			// import edit — the module is already in scope.
			OpenArcDocument(server, ctx, docURI, "import math\n\nmath.")
			completions := Completion(server, ctx, docURI, 2, 5)
			item, found := FindCompletion(completions.Items, "avg")
			Expect(found).To(BeTrue())
			Expect(item.AdditionalTextEdits).To(BeEmpty(),
				"the module is already imported in the source; the dot completion must not re-add the import")
		})

		It("surfaces module members under their qualified name for a bare-name prefix and qualifies on insert", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {\n    no\n}")
			completions := Completion(server, ctx, docURI, 1, 6)
			item, found := FindCompletion(completions.Items, "time.now")
			Expect(found).To(BeTrue(),
				"a bare 'no' prefix should surface qualified module members in addition to any bare alias")
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("time.now($0)"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
			Expect(item.AdditionalTextEdits).To(HaveLen(1))
			Expect(item.AdditionalTextEdits[0].NewText).To(Equal("import time\n\n"))
		})

		It("does not attach an import edit to a bare-name qualified suggestion when the module is already imported", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\n\nfunc test() {\n    no\n}")
			completions := Completion(server, ctx, docURI, 3, 6)
			item, found := FindCompletion(completions.Items, "time.now")
			Expect(found).To(BeTrue())
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("time.now($0)"))
			Expect(item.AdditionalTextEdits).To(BeEmpty())
		})

		It("appends an input-block snippet when a function completes in a flow context", func(ctx SpecContext) {
			// Flow contexts (sequence body, stage body, top level) invoke
			// functions with an input block — the inserted text must end
			// in `{$0}` and use snippet format so the cursor lands inside.
			OpenArcDocument(server, ctx, docURI, "import math\n\nsequence main {\n    math.av\n}")
			completions := Completion(server, ctx, docURI, 3, 11)
			item, found := FindCompletion(completions.Items, "avg")
			Expect(found).To(BeTrue())
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("math.avg{$0}"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
		})

		It("appends a call-parens snippet when a function completes in an imperative context", func(ctx SpecContext) {
			// A function body is imperative/WASM; functions invoke with
			// parens — the snippet ends in `($0)`.
			OpenArcDocument(server, ctx, docURI, "import time\n\nfunc test() {\n    time.no\n}")
			completions := Completion(server, ctx, docURI, 3, 11)
			item, found := FindCompletion(completions.Items, "now")
			Expect(found).To(BeTrue())
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("time.now($0)"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
		})

		It("appends an input-block snippet for bare-name deep-search results in a flow context", func(ctx SpecContext) {
			// The deep-search path that surfaces `time.wait` for a bare
			// `wai` prefix must follow the same context-aware suffix rule.
			OpenArcDocument(server, ctx, docURI, "sequence main {\n    wai\n}")
			completions := Completion(server, ctx, docURI, 1, 7)
			item, found := FindCompletion(completions.Items, "time.wait")
			Expect(found).To(BeTrue())
			Expect(ItemTextEdit(item)).ToNot(BeNil())
			Expect(ItemTextEdit(item).NewText).To(Equal("time.wait{$0}"))
			Expect(item.InsertTextFormat).To(Equal(protocol.InsertTextFormatSnippet))
		})

		It("does not append an invocation suffix to non-function symbols", func(ctx SpecContext) {
			// Channels are values, not callable — completing a channel
			// must not wrap it in `{}` or `()`.
			channels := []symbol.Symbol{
				{Name: "temperature_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			}
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, channels...)
			}}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, docURI, "sequence main {\n    temp\n}")
			completions := Completion(server, ctx, docURI, 1, 8)
			item, found := FindCompletion(completions.Items, "temperature_sensor")
			Expect(found).To(BeTrue())
			Expect(ItemInsertText(item)).To(BeEmpty(),
				"a channel symbol must not carry a snippet invocation suffix")
			Expect(item.InsertTextFormat).ToNot(Equal(protocol.InsertTextFormatSnippet))
		})

		It("filters bare-name qualified suggestions by execution context", func(ctx SpecContext) {
			// time.interval is flow-only and must not appear in a func body
			// even via the bare-name deep-search path.
			OpenArcDocument(server, ctx, docURI, "func test() {\n    inter\n}")
			completions := Completion(server, ctx, docURI, 1, 9)
			_, found := FindCompletion(completions.Items, "time.interval")
			Expect(found).To(BeFalse(),
				"flow-only time.interval must not appear inside a func body")
		})
	})

	Describe("Same-Line After Opening Brace", func() {
		DescribeTable("returns no completions when the cursor is on the same line as the opening brace",
			func(ctx SpecContext, content string, line, char uint32) {
				OpenArcDocument(server, ctx, docURI, content)
				completions := Completion(server, ctx, docURI, line, char)
				Expect(completions).ToNot(BeNil())
				Expect(completions.Items).To(BeEmpty())
			},
			Entry("func, cursor flush against brace",
				"func cat() {", uint32(0), uint32(12)),
			Entry("func, cursor after trailing space",
				"func cat() { ", uint32(0), uint32(13)),
			Entry("sequence, cursor flush against brace",
				"sequence main {", uint32(0), uint32(15)),
			Entry("sequence, cursor after trailing space",
				"sequence main { ", uint32(0), uint32(16)),
			Entry("stage, cursor flush against brace",
				"sequence main {\n    stage cat {", uint32(1), uint32(15)),
			Entry("stage, cursor after trailing space",
				"sequence main {\n    stage cat { ", uint32(1), uint32(16)),
		)

		DescribeTable("still returns block-body completions on the next line",
			func(ctx SpecContext, content string, line, char uint32, expectedLabel string) {
				OpenArcDocument(server, ctx, docURI, content)
				completions := Completion(server, ctx, docURI, line, char)
				Expect(completions).ToNot(BeNil())
				Expect(HasCompletion(completions.Items, expectedLabel)).To(BeTrue(),
					"Expected to find %q completion on the new line inside the block", expectedLabel)
			},
			Entry("func body offers return",
				"func cat() {\n    \n}", uint32(1), uint32(4), "return"),
			Entry("sequence body offers stage",
				"sequence main {\n    \n}", uint32(1), uint32(4), "stage"),
			Entry("stage body offers next",
				"sequence main {\n    stage cat {\n        \n    }\n}", uint32(2), uint32(8), "next"),
		)
	})
})

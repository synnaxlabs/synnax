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
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Rename breaker", func() {
	const docURI = uri.URI("file:///breaker.arc")
	var renamedKinds []string

	openServer := func(
		ctx context.Context,
		dashed bool,
		content string,
		channels ...string,
	) *lsp.Server {
		GinkgoHelper()
		syms := make([]symbol.Symbol, len(channels))
		for i, name := range channels {
			syms[i] = symbol.Symbol{
				Name:       name,
				Type:       types.Chan(types.F32()),
				Kind:       symbol.KindChannel,
				ID:         100 + i,
				Renameable: true,
			}
		}
		renamedKinds = nil
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot:          func() *symbol.Symbol { return NewRoot(nil, syms...) },
			AllowDashedNames: dashed,
			OnRename: func(_ context.Context, sym *symbol.Symbol, _, _ string) error {
				renamedKinds = append(renamedKinds, sym.Kind.String())
				return nil
			},
		}))
		server.SetClient(&MockClient{})
		OpenArcDocument(server, ctx, docURI, content)
		return server
	}
	rename := func(
		ctx context.Context,
		server *lsp.Server,
		line, character uint32,
		newName string,
	) *protocol.WorkspaceEdit {
		GinkgoHelper()
		return MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: character},
			NewName:      newName,
		}))
	}
	// apply writes edits into content, with positions counted in code points as the
	// server counts them.
	apply := func(content string, edits []protocol.TextEdit) string {
		lines := strings.Split(content, "\n")
		sorted := slices.Clone(edits)
		slices.SortFunc(sorted, func(a, b protocol.TextEdit) int {
			if a.Range.Start.Line != b.Range.Start.Line {
				return int(b.Range.Start.Line) - int(a.Range.Start.Line)
			}
			return int(b.Range.Start.Character) - int(a.Range.Start.Character)
		})
		for _, e := range sorted {
			runes := []rune(lines[e.Range.Start.Line])
			start := int(e.Range.Start.Character)
			end := min(int(e.Range.End.Character), len(runes))
			lines[e.Range.Start.Line] = string(runes[:start]) + e.NewText +
				string(runes[end:])
		}
		return strings.Join(lines, "\n")
	}
	parseErrors := func(content string, dashed bool) string {
		_, diags := parser.Parse(content, parser.Config{AllowDashedNames: dashed})
		if diags == nil || diags.Ok() {
			return ""
		}
		return diags.String()
	}
	identifiers := func(content string, dashed bool) []string {
		var names []string
		lexer := parser.NewLexer(content, parser.Config{AllowDashedNames: dashed})
		for _, t := range lexer.GetAllTokens() {
			if t.GetTokenType() == parser.ArcLexerIDENTIFIER {
				names = append(names, t.GetText())
			}
		}
		return names
	}

	It("Should keep an authority entry valid after a rename to a quoted name",
		func(ctx SpecContext) {
			content := "authority (sensor 100)\nfunc test() {\n" +
				"    x f32 := sensor + 1.0\n}"
			Expect(parseErrors(content, false)).To(BeEmpty())
			server := openServer(ctx, false, content, "sensor")
			result := rename(ctx, server, 2, 13, "PT-FU 01")
			Expect(result).ToNot(BeNil())
			renamed := apply(content, result.Changes[docURI])
			Expect(parseErrors(renamed, false)).To(BeEmpty(), renamed)
		},
	)

	It("Should rename again a channel that a first rename quoted",
		func(ctx SpecContext) {
			content := "func test() {\n    x f32 := sensor + 1.0\n}"
			server := openServer(ctx, false, content, "sensor")
			first := rename(ctx, server, 1, 13, "PT-FU 01")
			Expect(first).ToNot(BeNil())
			renamed := apply(content, first.Changes[docURI])
			Expect(renamed).To(ContainSubstring("x f32 := 'PT-FU 01' + 1.0"))
			server = openServer(ctx, false, renamed, "PT-FU 01")
			Expect(rename(ctx, server, 1, 15, "PT-FU 02")).ToNot(BeNil())
		},
	)

	It("Should rename the channel that a quoted reference under the cursor names",
		func(ctx SpecContext) {
			content := "func test() {\n    sensor := 2.0\n" +
				"    x f32 := 'sensor' + sensor\n}"
			server := openServer(ctx, false, content, "sensor")
			result := rename(ctx, server, 2, 15, "renamed")
			Expect(result).ToNot(BeNil())
			Expect(renamedKinds).To(Equal([]string{symbol.KindChannel.String()}))
			Expect(apply(content, result.Changes[docURI])).To(Equal(
				"func test() {\n    sensor := 2.0\n" +
					"    x f32 := renamed + sensor\n}",
			))
		},
	)

	It("Should end the edit of a quoted reference at the end of its token",
		func(ctx SpecContext) {
			kelvin := string(rune(0x212A)) + "elvin"
			content := "func test() {\n    x f32 := '" + kelvin + "' + 1.0\n" +
				"    y := Kelvin * 2.0\n}"
			server := openServer(ctx, false, content, "Kelvin")
			result := rename(ctx, server, 2, 10, "renamed")
			Expect(result).ToNot(BeNil())
			edits := result.Changes[docURI]
			Expect(edits).To(HaveLen(2))
			Expect(edits[0].Range).To(Equal(protocol.Range{
				Start: protocol.Position{Line: 1, Character: 13},
				End:   protocol.Position{Line: 1, Character: 21},
			}))
			Expect(apply(content, edits)).To(Equal(
				"func test() {\n    x f32 := renamed + 1.0\n" +
					"    y := renamed * 2.0\n}",
			))
		},
	)

	It("Should keep a bare name apart from a dash that follows it in dashed mode",
		func(ctx SpecContext) {
			content := "func test() {\n    x f32 := 'sensor'-1.0\n" +
				"    y := sensor * 2.0\n}"
			Expect(parseErrors(content, true)).To(BeEmpty())
			server := openServer(ctx, true, content, "sensor")
			result := rename(ctx, server, 2, 10, "renamed_sensor")
			Expect(result).ToNot(BeNil())
			renamed := apply(content, result.Changes[docURI])
			Expect(identifiers(renamed, true)).To(Equal(
				[]string{"test", "x", "renamed_sensor", "y", "renamed_sensor"},
			), renamed)
		},
	)

	It("Should keep a bare name apart from a keyword that touches its quote",
		func(ctx SpecContext) {
			content := "func test() f32 {\n    y := sensor * 2.0\n" +
				"    return'sensor'\n}"
			Expect(parseErrors(content, false)).To(BeEmpty())
			server := openServer(ctx, false, content, "sensor")
			result := rename(ctx, server, 1, 10, "renamed_sensor")
			Expect(result).ToNot(BeNil())
			renamed := apply(content, result.Changes[docURI])
			Expect(identifiers(renamed, false)).To(Equal(
				[]string{"test", "y", "renamed_sensor", "renamed_sensor"},
			), renamed)
		},
	)
})

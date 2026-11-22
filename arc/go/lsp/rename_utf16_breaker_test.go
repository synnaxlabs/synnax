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
	"unicode/utf16"

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

var _ = Describe("UTF-16 rename breaker", func() {
	const docURI = uri.URI("file:///utf16_breaker.arc")
	channel := func(name string, t types.Type, id int) symbol.Symbol {
		return symbol.Symbol{
			Name: name, Type: t, Kind: symbol.KindChannel, ID: id, Renameable: true,
		}
	}
	// open starts a server over content with the channels x (f64), trig (u8), log
	// (str), and sensor (f64), plus extra.
	open := func(
		ctx SpecContext,
		content string,
		extra ...symbol.Symbol,
	) (*lsp.Server, *MockClient) {
		GinkgoHelper()
		channels := append([]symbol.Symbol{
			channel("x", types.Chan(types.F64()), 7),
			channel("trig", types.Chan(types.U8()), 9),
			channel("log", types.Chan(types.String()), 13),
			channel("sensor", types.Chan(types.F64()), 14),
		}, extra...)
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			OnRename: func(context.Context, *symbol.Symbol, string, string) error {
				return nil
			},
		}))
		client := &MockClient{}
		server.SetClient(client)
		OpenArcDocument(server, ctx, docURI, content)
		return server, client
	}
	errorsOf := func(diags []protocol.Diagnostic) []string {
		var out []string
		for _, d := range diags {
			if d.Severity == protocol.DiagnosticSeverityError {
				out = append(out, DiagnosticMessage(d))
			}
		}
		return out
	}
	rename := func(
		ctx SpecContext,
		server *lsp.Server,
		line, char uint32,
		newName string,
	) []protocol.TextEdit {
		GinkgoHelper()
		result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: char},
			NewName:      newName,
		}))
		Expect(result).ToNot(BeNil())
		return result.Changes[docURI]
	}
	// apply writes one-line edits into content, with columns in UTF-16 units as an LSP
	// client counts them.
	apply := func(content string, edits []protocol.TextEdit) string {
		GinkgoHelper()
		lines := strings.Split(content, "\n")
		sorted := slices.Clone(edits)
		slices.SortFunc(sorted, func(a, b protocol.TextEdit) int {
			if a.Range.Start.Line != b.Range.Start.Line {
				return int(b.Range.Start.Line) - int(a.Range.Start.Line)
			}
			return int(b.Range.Start.Character) - int(a.Range.Start.Character)
		})
		for _, e := range sorted {
			Expect(e.Range.End.Line).To(Equal(e.Range.Start.Line))
			units := utf16.Encode([]rune(lines[e.Range.Start.Line]))
			lines[e.Range.Start.Line] = string(
				utf16.Decode(units[:e.Range.Start.Character]),
			) + e.NewText + string(utf16.Decode(units[e.Range.End.Character:]))
		}
		return strings.Join(lines, "\n")
	}
	// textAt returns the text of the one-line range r of content, with columns in
	// UTF-16 units.
	textAt := func(content string, r protocol.Range) string {
		units := utf16.Encode([]rune(strings.Split(content, "\n")[r.Start.Line]))
		return string(utf16.Decode(units[r.Start.Character:r.End.Character]))
	}

	Describe("Format string placeholders", func() {
		// The channel x is read bare in a placeholder and on its own line. A rename to
		// a name that is not an identifier writes the quoted form into the placeholder,
		// so the program after the rename must still read that channel.
		DescribeTable("Should keep the program valid after a rename to a name that "+
			"needs the quoted form",
			func(ctx SpecContext, literal, newName string) {
				content := "func f() {\n    'log' = " + literal + "\n" +
					"    'log' = f\"{x}\"\n}\n'trig' -> f()\n"
				server, client := open(ctx, content)
				Expect(errorsOf(client.Diagnostics())).To(BeEmpty())
				renamed := apply(content, rename(ctx, server, 2, 15, newName))
				f64 := types.Chan(types.F64())
				_, after := open(ctx, renamed, channel(newName, f64, 30))
				Expect(errorsOf(after.Diagnostics())).To(BeEmpty(), renamed)
			},
			Entry("backslash in a format string", `f"{x}"`, `PT\01`),
			Entry("double quote in a format string", `f"{x}"`, `say "hi"`),
			Entry("backtick in a multi-line format string", "f`{x}`", "a`b"),
		)
	})

	Describe("Prepare rename of a quoted name", func() {
		It("Should offer the name without its quotes as the text to edit", func(
			ctx SpecContext,
		) {
			content := "func f() {\n    y := 'Stand A' + 1.0\n}\n'trig' -> f()\n"
			standA := channel("Stand A", types.Chan(types.F64()), 20)
			server, _ := open(ctx, content, standA)
			params := &protocol.PrepareRenameParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 1, Character: 12},
			}
			result := MustSucceed(server.PrepareRename(ctx, params))
			// A client fills its rename box with the placeholder, or with the text of
			// the range when the result is a range.
			var shown string
			switch r := result.(type) {
			case *protocol.PrepareRenamePlaceholder:
				shown = r.Placeholder
			case *protocol.Range:
				shown = textAt(content, *r)
			default:
				Fail("no rename offered")
			}
			Expect(shown).To(Equal("Stand A"))
		})

		It("Should offer a rename one past the end of a quoted name, as it does for a "+
			"bare name", func(ctx SpecContext) {
			content := "func f() {\n    y := 'Stand A' + sensor * 2.0\n}\n" +
				"'trig' -> f()\n"
			standA := channel("Stand A", types.Chan(types.F64()), 20)
			server, _ := open(ctx, content, standA)
			prepare := func(char uint32) protocol.PrepareRenameResult {
				params := &protocol.PrepareRenameParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 1, Character: char},
				}
				return MustSucceed(server.PrepareRename(ctx, params))
			}
			// One past the end of sensor.
			Expect(prepare(27)).ToNot(BeNil())
			// One past the end of 'Stand A'.
			Expect(prepare(18)).ToNot(BeNil())
		})
	})

	Describe("Rename of a local", func() {
		It("Should keep a bare channel reference on its channel when a local takes "+
			"the channel's name", func(ctx SpecContext) {
			content := "func f() {\n    count := 2.0\n    y := count + sensor\n}\n" +
				"'trig' -> f()\n"
			server, _ := open(ctx, content)
			result, err := server.Rename(ctx, &protocol.RenameParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 1, Character: 5},
				NewName:      "sensor",
			})
			// A refusal also keeps the program's meaning.
			if err != nil {
				return
			}
			renamed := apply(content, result.Changes[docURI])
			Expect(renamed).To(ContainSubstring("y := sensor + 'sensor'"), renamed)
		})
	})
})

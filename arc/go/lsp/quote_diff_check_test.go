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
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

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

const quoteDiffSeed = "func f() {\n" +
	"    x := 'ox-pt-1' + 'pt-2'\n" +
	"    s := `first\n" +
	"second 'quoted' \"text\"\n" +
	"`\n" +
	"    /* a 'comment'\n" +
	"    still */ y := \"it's\" // don't\n" +
	"    z := f\"{'ox-p'} and {x}\"\n" +
	"    m := f`{'pt-2'}\n" +
	"line`\n" +
	"}\n"

var quoteDiffPieces = []string{
	"`", "'", "\"", "/*", "*/", "//", "\n", "x", "{", "}", "f\"{", "\\", "'ox-p'",
	"é", " ", "é", "`a`", "\"b\"", "\r\n",
}

// runeBoundary moves i back to the start of the rune that holds it.
func runeBoundary(text string, i int) int {
	for i > 0 && i < len(text) && !utf8.RuneStart(text[i]) {
		i--
	}
	return i
}

// positionOf returns the position of the byte offset i of text, in UTF-16 units.
func positionOf(text string, i int) protocol.Position {
	start := strings.LastIndexByte(text[:i], '\n') + 1
	return protocol.Position{
		Line:      uint32(strings.Count(text[:i], "\n")),
		Character: uint32(len(utf16.Encode([]rune(text[start:i])))),
	}
}

// quoteSignature returns the parts of a completion that the quote scan decides and the
// analysis does not: the keywords, which only code offers, and the edits, which only a
// quoted name gives.
func quoteSignature(list *protocol.CompletionList) []string {
	var out []string
	for _, item := range list.Items {
		if item.Kind == protocol.CompletionItemKindKeyword {
			out = append(out, "keyword "+item.Label)
		}
		if edit := ItemTextEdit(item); edit != nil {
			out = append(
				out,
				fmt.Sprintf("%s %v %q", item.Label, edit.Range, edit.NewText),
			)
		}
	}
	slices.Sort(out)
	return out
}

var _ = Describe("Quote scan after edits (check)", func() {
	It("should complete after each of many edits as a fresh document does", func(
		ctx SpecContext,
	) {
		channels := []symbol.Symbol{{
			Name: "ox-pt-1",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   1,
		}, {
			Name: "pt-2",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		}}
		cfg := lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
		}
		edited := MustSucceed(lsp.New(cfg))
		edited.SetClient(&MockClient{})
		fresh := MustSucceed(lsp.New(cfg))
		fresh.SetClient(&MockClient{})
		const editedURI uri.URI = "file:///edited.arc"
		r := rand.New(rand.NewPCG(7, 13))
		text := quoteDiffSeed
		OpenArcDocument(edited, ctx, editedURI, text)
		version := int32(1)
		for n := range 1000 {
			if len(text) > 2000 {
				text = quoteDiffSeed
			}
			at := runeBoundary(text, r.IntN(len(text)+1))
			end := max(runeBoundary(text, min(len(text), at+r.IntN(6))), at)
			var insert strings.Builder
			for range r.IntN(3) {
				insert.WriteString(quoteDiffPieces[r.IntN(len(quoteDiffPieces))])
			}
			text = text[:at] + insert.String() + text[end:]
			version++
			ChangeDocument(edited, ctx, editedURI, text, version)
			freshURI := uri.URI(fmt.Sprintf("file:///fresh-%d.arc", n))
			OpenArcDocument(fresh, ctx, freshURI, text)
			for range 4 {
				pos := positionOf(text, runeBoundary(text, r.IntN(len(text)+1)))
				Expect(quoteSignature(
					Completion(edited, ctx, editedURI, pos.Line, pos.Character),
				)).To(Equal(quoteSignature(
					Completion(fresh, ctx, freshURI, pos.Line, pos.Character),
				)), "edit %d of %q at %v", n, text, pos)
			}
		}
	})
})

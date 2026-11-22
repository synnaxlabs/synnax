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
	"strconv"
	"strings"
	"testing"

	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// hoverSource returns a function that declares lines variables after its first
// statement. It passes analysis when its first statement does.
func hoverSource(first string, lines int) string {
	var b strings.Builder
	b.WriteString("func f() i32 {\n    " + first + "\n")
	for i := range lines {
		b.WriteString("    a" + strconv.Itoa(i) + " i32 := 1\n")
	}
	b.WriteString("    return a0\n}\n")
	return b.String()
}

// benchHover opens source with channels in scope and hovers the position on line 1 at
// char.
func benchHover(b *testing.B, source string, char uint32, channels ...symbol.Symbol) {
	b.Helper()
	ctx := b.Context()
	server, err := lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
		root := symbol.NewRoot(nil, stl.NewSymbols())
		for i := range channels {
			root.Parent.AddChild(&channels[i])
		}
		return root
	}})
	if err != nil {
		b.Fatal(err)
	}
	server.SetClient(&testutil.MockClient{})
	docURI := uri.URI("file:///bench.arc")
	if err = server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "arc",
			Version:    1,
			Text:       source,
		},
	}); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := server.DidClose(ctx, &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		}); err != nil {
			b.Error(err)
		}
	})
	params := &protocol.HoverParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: 1, Character: char},
	}
	// The first hover after a change lexes the document once.
	if _, err := server.Hover(ctx, params); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := server.Hover(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHover hovers a variable, a channel path, and a path that no channel holds,
// near the top of documents of growing length. Only the last fails analysis.
func BenchmarkHover(b *testing.B) {
	sensor := symbol.Symbol{
		Name: "stand.sensor",
		Kind: symbol.KindChannel,
		Type: types.Chan(types.F64()),
		ID:   1,
	}
	for _, lines := range []int{10, 1000, 10000} {
		n := "lines=" + strconv.Itoa(lines)
		b.Run(n+"/word", func(b *testing.B) {
			benchHover(b, hoverSource("x := 1", lines), 5)
		})
		b.Run(n+"/path", func(b *testing.B) {
			benchHover(b, hoverSource("s := stand.sensor", lines), 16, sensor)
		})
		b.Run(n+"/missing", func(b *testing.B) {
			benchHover(b, hoverSource("s := stand.missing", lines), 16)
		})
	}
}

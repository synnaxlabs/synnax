// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"fmt"
	"io/fs"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	lsptestutil "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// shape returns each field of syms and their descendants that analysis could write,
// one line per symbol, depth first.
func shape(syms []*symbol.Symbol) []string {
	var (
		lines []string
		walk  func(path string, s *symbol.Symbol)
	)
	walk = func(path string, s *symbol.Symbol) {
		path += "/" + s.Name
		var parent, deprecated string
		if s.Parent != nil {
			parent = s.Parent.Name
		}
		if s.Deprecated != nil {
			deprecated = s.Deprecated.QualifiedName()
		}
		lines = append(lines, fmt.Sprintf(
			"%s kind=%d type=%s id=%d source=%v exec=%d internal=%t exported=%t "+
				"user=%t reassigned=%t renameable=%t used=%t trigger=%v channels=%v "+
				"default=%v parent=%s deprecated=%s",
			path, s.Kind, s.Type, s.ID, s.SourceID, s.Exec, s.Internal, s.Exported,
			s.UserDefined, s.Reassigned, s.Renameable, s.Used, s.Trigger, s.Channels,
			s.DefaultValue, parent, deprecated,
		))
		for _, child := range s.Children() {
			walk(path, child)
		}
	}
	for _, s := range syms {
		walk("", s)
	}
	return lines
}

// sharedLSPSource is a program whose analysis resolves an imported module's member
// through a polymorphic call, with a parameter that the spec renames.
const sharedLSPSource = "import math\n\nfunc g(x f64) u8 {\n    if math.is_nan(x) {\n" +
	"        return 1\n    }\n    return 0\n}\n"

// sharedGraphBody is a graph function body that calls an auto-imported module's
// polymorphic member.
const sharedGraphBody = "{\n    if math.is_nan(ox_pt_1) {\n        return 1\n    }\n" +
	"    return 0\n}"

// useThroughLSP opens sharedLSPSource on a server whose roots share ambient, then asks
// for a hover, a completion, and a rename.
func useThroughLSP(ctx context.Context, ambient []*symbol.Symbol, id int) {
	GinkgoHelper()
	server := MustSucceed(lsp.New(lsp.Config{
		NewRoot: func() *symbol.Symbol { return symbol.NewRoot(nil, ambient) },
	}))
	server.SetClient(&lsptestutil.MockClient{})
	doc := uri.URI(fmt.Sprintf("file:///shared-%d.arc", id))
	lsptestutil.OpenDocument(server, ctx, doc, sharedLSPSource, "arc")
	at := func(line, character uint32) protocol.TextDocumentPositionParams {
		return protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: doc},
			Position:     protocol.Position{Line: line, Character: character},
		}
	}
	hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: at(3, 13),
	}))
	Expect(lsptestutil.HoverContents(hover)).To(ContainSubstring("is_nan"))
	completions := lsptestutil.Completion(server, ctx, doc, 3, 12)
	Expect(lo.Map(completions.Items, func(item protocol.CompletionItem, _ int) string {
		return item.Label
	})).To(ContainElement("is_nan"))
	edit := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
		TextDocumentPositionParams: at(2, 7),
		NewName:                    "y",
	}))
	Expect(edit.Changes[doc]).To(HaveLen(2))
	Expect(server.Shutdown(ctx)).To(Succeed())
}

var _ = Describe("A shared standard library", func() {
	It("Should serve concurrent compiles and LSP requests and stay unchanged", func(
		ctx SpecContext,
	) {
		shared := stl.NewSymbols()
		sources := lo.Map(
			MustSucceed(findRepositoryPrograms()),
			func(path string, _ int) string {
				return string(MustSucceed(fs.ReadFile(repository, path)))
			},
		)
		g := arc.Graph{Functions: ir.Functions{{
			Key:     "calculation",
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Body:    ir.Body{Raw: sharedGraphBody},
		}}}
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Go(func() {
				defer GinkgoRecover()
				for _, raw := range sources {
					MustSucceed(arc.CompileText(
						ctx,
						arc.Text{Raw: raw},
						newRepositoryRoot(shared),
					))
				}
				MustSucceed(arc.CompileGraph(ctx, g, newRepositoryRoot(shared)))
				useThroughLSP(ctx, shared, i)
			})
		}
		wg.Wait()
		Expect(shape(shared)).To(Equal(shape(stl.NewSymbols())))
	})
})

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

// sharedLSPSource is a program whose analysis resolves module members, a polymorphic
// call, and a local that the spec renames.
const sharedLSPSource = "func g(x f64) f64 {\n    return math.sqrt(x) + x\n}\n"

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
		TextDocumentPositionParams: at(1, 17),
	}))
	Expect(lsptestutil.HoverContents(hover)).To(ContainSubstring("sqrt"))
	completions := lsptestutil.Completion(server, ctx, doc, 1, 16)
	Expect(lo.Map(completions.Items, func(item protocol.CompletionItem, _ int) string {
		return item.Label
	})).To(ContainElement("sqrt"))
	edit := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
		TextDocumentPositionParams: at(0, 7),
		NewName:                    "y",
	}))
	Expect(edit.Changes[doc]).To(HaveLen(3))
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
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			Body:    ir.Body{Raw: "{ return math.sqrt(ox_pt_1) + ox_pt_2 }"},
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

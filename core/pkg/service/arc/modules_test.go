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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

var _ = Describe("Modules", func() {
	const source = "export func f() i64 { return 1 }"
	var (
		w    arc.Writer
		root directory.Key
	)
	createDirectory := func(
		ctx SpecContext,
		name string,
		parent directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		Expect(w.CreateDirectory(ctx, &d, parent)).To(Succeed())
		return d
	}
	createArc := func(ctx SpecContext, t arc.Type, name string, src arc.Source) arc.Arc {
		GinkgoHelper()
		a := newArc(t, arc.Base{Name: name, Source: src})
		Expect(w.Create(ctx, &a)).To(Succeed())
		return a
	}
	textSource := func(raw string) arc.Source {
		return arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}}
	}
	BeforeEach(func(ctx SpecContext) {
		w = svc.NewWriter(tx)
		var roots []ontology.Resource
		Expect(otg.NewRetrieve().
			WhereIDs(ontology.RootID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeArcDirectory).
			Entries(&roots).
			Exec(ctx, tx)).To(Succeed())
		root = MustSucceed(directory.KeyFromOntologyID(roots[0].ID))
	})

	It("Should resolve a module directly under Arcs", func(ctx SpecContext) {
		createArc(ctx, arc.TypeModule, "m", textSource(source))
		mod := MustSucceed(svc.NewRoot(tx).ResolveModule(ctx, "m"))
		Expect(mod.Kind).To(Equal(symbol.KindModule))
		Expect(mod.FindChild("f").Exported).To(BeTrue())
	})

	It("Should resolve a module by its directory path", func(ctx SpecContext) {
		lib := createDirectory(ctx, "lib", root)
		sub := createDirectory(ctx, "sub", lib.Key)
		a := createArc(ctx, arc.TypeModule, "m", textSource(source))
		Expect(w.Move(ctx, root, sub.Key, a.OntologyID())).To(Succeed())
		mod := MustSucceed(svc.NewRoot(tx).ResolveModule(ctx, "lib.sub.m"))
		Expect(mod.Name).To(Equal("lib.sub.m"))
		Expect(svc.NewRoot(tx).ResolveModule(ctx, "m")).Error().
			To(MatchError(symbol.ErrModuleNotFound))
	})

	It("Should report a path through a missing directory", func(ctx SpecContext) {
		Expect(svc.NewRoot(tx).ResolveModule(ctx, "nope.m")).Error().
			To(MatchError(symbol.ErrModuleNotFound))
	})

	It("Should refuse to import an automation", func(ctx SpecContext) {
		createArc(ctx, arc.TypeAutomation, "a", textSource(source))
		Expect(svc.NewRoot(tx).ResolveModule(ctx, "a")).Error().To(MatchError(
			ContainSubstring(`"a" is an automation and cannot be imported`),
		))
	})

	It("Should refuse to import a graph module", func(ctx SpecContext) {
		createArc(
			ctx,
			arc.TypeModule,
			"g",
			arc.Source{Variant: arc.GraphSource{}},
		)
		Expect(svc.NewRoot(tx).ResolveModule(ctx, "g")).Error().To(MatchError(
			ContainSubstring(`module "g" is a graph and cannot be imported`),
		))
	})

	It("Should resolve a user module in the LSP", func(ctx SpecContext) {
		w = svc.NewWriter(nil)
		a := createArc(ctx, arc.TypeModule, "lsp_module", textSource(source))
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
		})
		server := MustSucceed(svc.NewLSP())
		DeferCleanup(func() { Expect(server.Shutdown(ctx)).To(Succeed()) })
		client := &MockClient{}
		server.SetClient(client)
		OpenArcDocument(
			server,
			ctx,
			uri.URI("file:///test.arc"),
			"import lsp_module\n\nfunc g() i64 { return lsp_module.f() }",
		)
		Expect(client.Diagnostics()).To(BeEmpty())
	})

	It("Should report an unknown module", func(ctx SpecContext) {
		Expect(svc.NewRoot(tx).ResolveModule(ctx, "nope")).Error().
			To(MatchError(symbol.ErrModuleNotFound))
	})
})

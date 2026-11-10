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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/crdt"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

var _ = Describe("Modules", func() {
	const source = "export func f() i64 { return 1 }"
	var (
		tx   gorp.Tx
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
	resolveModule := func(ctx SpecContext, path string) (*symbol.Symbol, error) {
		return svc.NewRoot(tx, &channel.Lookups{}).ResolveModule(ctx, path)
	}
	Context("ResolveModule", func() {
		BeforeEach(func(ctx SpecContext) {
			tx = DeferClose(db.OpenTx())
			w = svc.NewWriter(tx)
			root = rootDirectory(ctx, tx).Key
		})

		It("Should resolve a module directly under Arcs", func(ctx SpecContext) {
			createArc(ctx, arc.TypeModule, "m", textSource(source))
			mod := MustSucceed(resolveModule(ctx, "m"))
			Expect(mod.Kind).To(Equal(symbol.KindModule))
			Expect(mod.FindChild("f").Exported).To(BeTrue())
		})

		It("Should resolve a module by its directory path", func(ctx SpecContext) {
			lib := createDirectory(ctx, "lib", root)
			sub := createDirectory(ctx, "sub", lib.Key)
			a := createArc(ctx, arc.TypeModule, "m", textSource(source))
			Expect(w.Move(ctx, root, sub.Key, a.ResourceID())).To(Succeed())
			mod := MustSucceed(resolveModule(ctx, "lib.sub.m"))
			Expect(mod.Name).To(Equal("lib.sub.m"))
			Expect(resolveModule(ctx, "m")).Error().
				To(MatchError(symbol.ErrModuleNotFound))
		})

		It("Should report a path through a missing directory", func(ctx SpecContext) {
			Expect(resolveModule(ctx, "nope.m")).Error().
				To(MatchError(symbol.ErrModuleNotFound))
		})

		It("Should refuse to import an automation", func(ctx SpecContext) {
			createArc(ctx, arc.TypeAutomation, "a", textSource(source))
			Expect(resolveModule(ctx, "a")).Error().To(MatchError(
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
			Expect(resolveModule(ctx, "g")).Error().To(MatchError(
				ContainSubstring(`module "g" is a graph and cannot be imported`),
			))
		})

		It("Should report an unknown module", func(ctx SpecContext) {
			Expect(resolveModule(ctx, "nope")).Error().
				To(MatchError(symbol.ErrModuleNotFound))
		})
	})

	It("Should resolve a user module in the LSP", func(ctx SpecContext) {
		a := newArc(
			arc.TypeModule,
			arc.Base{Name: "lsp_module", Source: textSource(source)},
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})
		server := MustSucceed(svc.NewLSP(lspConfig))
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

	It("Should compile again when a module's text changes before the check", func(
		ctx SpecContext,
	) {
		m := newArc(
			arc.TypeModule,
			arc.Base{Name: "checked_module", Source: textSource("")},
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &m)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, m.Base().Key)
			})).To(Succeed())
		})
		var compiles int
		mod := MustSucceed(channel.CompileConsistent(ctx, db, func(
			snap gorp.Snapshot,
			lookups *channel.Lookups,
		) (*symbol.Symbol, error) {
			compiles++
			mod, err := svc.NewRoot(snap, lookups).ResolveModule(ctx, "checked_module")
			if compiles == 1 {
				inserts := crdt.New(2).Insert(0, source)
				acts := make([]arc.Action, len(inserts))
				for i, in := range inserts {
					acts[i] = arc.NewInsertCharAction(arc.InsertCharPayload{
						ID:     in.ID,
						Origin: in.Origin,
						Side:   in.Side,
						Char:   in.Char,
					})
				}
				Expect(dispatch(ctx, m.Base().Key, "dk", acts)).To(Succeed())
			}
			return mod, err
		}))
		Expect(compiles).To(Equal(2))
		Expect(mod.FindChild("f")).ToNot(BeNil())
	})

	It("Should link a module's code into its importer's program", func(
		ctx SpecContext,
	) {
		m := newArc(arc.TypeModule, arc.Base{Name: "um_link_m", Source: textSource(`
export func f() {t u8} i64 { return 1 }
export sequence s { stage a { 1 -> um_link_seq } }
`)})
		a := newArc(arc.TypeAutomation, arc.Base{Name: "um_link_a", Source: textSource(`
import um_link_m

um_link_trig -> um_link_m.f() -> um_link_fn
um_link_trig == 1 => um_link_m.s
`)})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for name, dt := range map[string]telem.DataType{
				"um_link_trig": telem.Uint8T,
				"um_link_fn":   telem.Int64T,
				"um_link_seq":  telem.Uint8T,
			} {
				ch := channel.Channel{Name: name, Virtual: true, DataType: dt}
				if err := channelSvc.NewWriter(tx).Create(ctx, &ch); err != nil {
					return err
				}
			}
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &m); err != nil {
				return err
			}
			return w.Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key, m.Base().Key)
			})).To(Succeed())
		})
		result := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
		_, found := result.Program.Functions.Find("um_link_m.f")
		Expect(found).To(BeTrue())
		var scopes []string
		for _, stratum := range result.Program.Root.Strata {
			for _, member := range stratum {
				if member.Scope != nil {
					scopes = append(scopes, member.Scope.Key)
				}
			}
		}
		Expect(scopes).To(ContainElement("um_link_m.s"))
		Expect(result.Program.WASM).ToNot(BeEmpty())
	})
})

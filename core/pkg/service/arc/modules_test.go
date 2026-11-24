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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

var _ = Describe("Modules", func() {
	const source = "export func f() i64 { return 1 }"
	write := func(ctx SpecContext, f func(w arc.Writer) error) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return f(svc.NewWriter(tx))
		})).To(Succeed())
	}
	createDirectory := func(
		ctx SpecContext,
		name string,
		parent directory.Key,
	) directory.Directory {
		GinkgoHelper()
		d := directory.Directory{Name: name}
		write(
			ctx,
			func(w arc.Writer) error { return w.CreateDirectory(ctx, &d, parent) },
		)
		DeferCleanup(func(ctx SpecContext) {
			write(
				ctx,
				func(w arc.Writer) error { return w.DeleteDirectory(ctx, d.Key) },
			)
		})
		return d
	}
	createArc := func(ctx SpecContext, t arc.Type, name string, src arc.Source) arc.Arc {
		GinkgoHelper()
		a := newArc(t, arc.Base{Name: name, Source: src})
		write(ctx, func(w arc.Writer) error { return w.Create(ctx, &a) })
		DeferCleanup(func(ctx SpecContext) {
			write(ctx, func(w arc.Writer) error { return w.Delete(ctx, a.Base().Key) })
		})
		return a
	}
	textSource := func(raw string) arc.Source {
		return arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}}
	}
	// compileImport compiles a stored automation that imports path and calls f of the
	// module that name imports.
	importers := 0
	compileImport := func(ctx SpecContext, path, name string) error {
		GinkgoHelper()
		importers++
		importer := fmt.Sprintf("rm_importer_%d", importers)
		a := createArc(ctx, arc.TypeAutomation, importer, textSource(
			"import "+path+"\n\nfunc g() i64 { return "+name+".f() }",
		))
		_, err := svc.CompileProgram(ctx, a.Base().Key)
		return err
	}
	Context("Import", func() {
		var root directory.Key
		BeforeEach(func(ctx SpecContext) {
			tx := db.OpenTx()
			root = rootDirectory(ctx, tx).Key
			Expect(tx.Close()).To(Succeed())
		})

		It("Should import an exported function of a module directly under Arcs", func(
			ctx SpecContext,
		) {
			createArc(ctx, arc.TypeModule, "rm_m", textSource(source))
			Expect(compileImport(ctx, "rm_m", "rm_m")).To(Succeed())
		})

		It("Should import a module by its directory path", func(ctx SpecContext) {
			lib := createDirectory(ctx, "rm_lib", root)
			sub := createDirectory(ctx, "rm_sub", lib.Key)
			a := createArc(ctx, arc.TypeModule, "rm_m", textSource(source))
			write(ctx, func(w arc.Writer) error {
				return w.Move(ctx, root, sub.Key, a.ResourceID())
			})
			Expect(compileImport(ctx, "rm_lib.rm_sub.rm_m", "rm_m")).To(Succeed())
			Expect(compileImport(ctx, "rm_m", "rm_m")).
				To(MatchError(ContainSubstring(`import failed: unknown module "rm_m"`)))
		})

		It("Should report a path through a missing directory", func(ctx SpecContext) {
			Expect(compileImport(ctx, "rm_nope.rm_m", "rm_m")).To(MatchError(
				ContainSubstring(`import failed: unknown module "rm_nope.rm_m"`),
			))
		})

		It("Should refuse to import an automation", func(ctx SpecContext) {
			createArc(ctx, arc.TypeAutomation, "rm_a", textSource(source))
			Expect(compileImport(ctx, "rm_a", "rm_a")).To(MatchError(
				ContainSubstring(`"rm_a" is an automation and cannot be imported`),
			))
		})

		It("Should refuse to import a graph module", func(ctx SpecContext) {
			createArc(
				ctx,
				arc.TypeModule,
				"rm_g",
				arc.Source{Variant: arc.GraphSource{}},
			)
			Expect(compileImport(ctx, "rm_g", "rm_g")).To(MatchError(
				ContainSubstring(`module "rm_g" is a graph and cannot be imported`),
			))
		})

		It("Should report an unknown module", func(ctx SpecContext) {
			Expect(compileImport(ctx, "rm_nope", "rm_nope")).
				To(MatchError(ContainSubstring(`import failed: unknown module "rm_nope"`)))
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

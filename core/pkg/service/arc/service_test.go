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
	"sync/atomic"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"go.lsp.dev/uri"
)

var _ = Describe("CompileProgram", func() {
	It("Should retrieve and compile an Arc with a valid graph", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		a := newAutomation(arc.Base{
			Name: "test-arc",
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Functions: []ir.Function{
					{
						Key: "add",
						Inputs: types.Params{
							{Name: "a", Type: types.I64()},
							{Name: "b", Type: types.I64()},
						},
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.I64()},
						},
						Body: ir.Body{Raw: "{ return a + b }"},
					},
				},
			}}},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})

		result := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
		Expect(result.Key).To(Equal(a.Base().Key))
		Expect(result.Program.IR).ToNot(BeNil())
	})

	It("Should refuse a module, which never runs", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		m := newModule(arc.Base{
			Name: "never_runs",
			Source: arc.Source{
				Variant: arc.TextSource{Text: text.Text{Raw: "func f() {}"}},
			},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &m)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, m.Base().Key)
			})).To(Succeed())
		})
		Expect(svc.CompileProgram(ctx, m.Base().Key)).Error().
			To(MatchError(validate.ErrValidation))
	})

	It("Should return error when Arc does not exist", func(ctx SpecContext) {
		nonExistentKey := uuid.New()
		Expect(svc.CompileProgram(ctx, nonExistentKey)).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should return error when graph compilation fails", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		a := newAutomation(arc.Base{
			Name: "invalid-arc",
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Functions: []ir.Function{
					{
						Key: "source",
						Outputs: types.Params{
							{Name: ir.DefaultOutputParam, Type: types.F32()},
						},
					},
				},
				Nodes: []graph.Node{
					{Key: "src"},
				},
				Inputs: map[string]msgpack.EncodedJSON{
					"src": {"type": "source"},
				},
				Edges: graph.Edges{
					{
						Source: ir.Handle{Node: "src", Param: ir.DefaultOutputParam},
						Target: ir.Handle{Node: "nonexistent", Param: "input"},
					},
				},
			}}},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
		})

		Expect(svc.CompileProgram(ctx, a.Base().Key)).Error().
			To(MatchError(ContainSubstring("edge target node 'nonexistent' not found")))
	})
})

var _ = Describe("NewLSP", func() {
	It("Should create an LSP server without error", func(ctx SpecContext) {
		server := MustSucceed(svc.NewLSP(lspConfig))
		Expect(server).ToNot(BeNil())
	})

	DescribeTable(
		"Should reject a config with a missing function",
		func(clear func(*arc.LSPConfig), msg string) {
			cfg := lspConfig
			clear(&cfg)
			Expect(svc.NewLSP(cfg)).Error().To(MatchError(msg))
		},
		Entry(
			"allowed",
			func(c *arc.LSPConfig) { c.Allowed = nil },
			"allowed: must be non-nil",
		),
		Entry(
			"rename",
			func(c *arc.LSPConfig) { c.Rename = nil },
			"rename: must be non-nil",
		),
	)

	It("Should create multiple independent LSP servers", func(ctx SpecContext) {
		server1 := MustSucceed(svc.NewLSP(lspConfig))
		server2 := MustSucceed(svc.NewLSP(lspConfig))
		Expect(server1).ToNot(BeIdenticalTo(server2))
	})

	It("Should republish diagnostics when channels change", func(ctx SpecContext) {
		server := MustSucceed(svc.NewLSP(lspConfig))
		DeferCleanup(func() { Expect(server.Shutdown(ctx)).To(Succeed()) })
		client := &MockClient{}
		server.SetClient(client)

		uri := uri.URI("file:///test.arc")
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := test_lsp_channel\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(
			DiagnosticMessage(client.Diagnostics()[0]),
		).To(ContainSubstring("undefined symbol"))

		ch := channel.Channel{
			Name:     "test_lsp_channel",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())

		Eventually(func() int {
			return len(client.Diagnostics())
		}).Should(Equal(0))
	})

	Context("Read access", func() {
		const docURI = uri.URI("file:///access.arc")
		var visible, hidden channel.Channel
		BeforeEach(func(ctx SpecContext) {
			visible = channel.Channel{
				Name:     "lsp_visible_ch",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			hidden = channel.Channel{
				Name:     "lsp_hidden_ch",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := channelSvc.NewWriter(tx)
				return errors.Combine(w.Create(ctx, &visible), w.Create(ctx, &hidden))
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).DeleteMany(
						ctx,
						[]channel.Key{visible.Key(), hidden.Key()},
						false,
					)
				})).To(Succeed())
			})
		})
		hide := func(ids ...resource.ID) func(context.Context) (
			func(resource.ID) bool,
			error,
		) {
			return func(context.Context) (func(resource.ID) bool, error) {
				return func(id resource.ID) bool { return !lo.Contains(ids, id) }, nil
			}
		}
		open := func(
			ctx SpecContext,
			allowed func(context.Context) (func(resource.ID) bool, error),
			source string,
		) (*lsp.Server, *MockClient) {
			GinkgoHelper()
			server := MustSucceed(svc.NewLSP(arc.LSPConfig{
				Allowed: allowed,
				Rename:  noRename,
			}))
			DeferCleanup(func(ctx SpecContext) {
				Expect(server.Shutdown(ctx)).To(Succeed())
			})
			client := &MockClient{}
			server.SetClient(client)
			OpenArcDocument(server, ctx, docURI, source)
			return server, client
		}

		It("Should report a hidden channel as an undefined symbol", func(
			ctx SpecContext,
		) {
			_, client := open(
				ctx,
				hide(hidden.ResourceID()),
				"func test() {\n\ta := lsp_visible_ch\n\tb := lsp_hidden_ch\n}",
			)
			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(DiagnosticMessage(client.Diagnostics()[0])).
				To(ContainSubstring("undefined symbol: lsp_hidden_ch"))
		})

		It("Should leave a hidden channel out of completion", func(ctx SpecContext) {
			server, _ := open(
				ctx,
				hide(hidden.ResourceID()),
				"func test() {\n\tx := lsp_\n}",
			)
			Eventually(func(g Gomega) {
				items := Completion(server, ctx, docURI, 1, 10).Items
				g.Expect(HasCompletion(items, "lsp_visible_ch")).To(BeTrue())
				g.Expect(HasCompletion(items, "lsp_hidden_ch")).To(BeFalse())
			}).Should(Succeed())
		})

		It("Should report a hidden module as unknown", func(ctx SpecContext) {
			m := newModule(arc.Base{
				Name: "lsp_hidden_module",
				Source: arc.Source{Variant: arc.TextSource{
					Text: text.Text{Raw: "export func f() i64 { return 1 }"},
				}},
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &m)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Delete(ctx, m.Base().Key)
				})).To(Succeed())
			})
			_, client := open(
				ctx,
				hide(m.ResourceID()),
				"import lsp_hidden_module\n\n"+
					"func g() i64 { return lsp_hidden_module.f() }",
			)
			Expect(client.Diagnostics()).To(ContainElement(WithTransform(
				DiagnosticMessage,
				ContainSubstring(`unknown module "lsp_hidden_module"`),
			)))
		})

		It("Should complete the path of each readable text module", func(
			ctx SpecContext,
		) {
			textSource := func(raw string) arc.Source {
				return arc.Source{Variant: arc.TextSource{Text: text.Text{Raw: raw}}}
			}
			listed := newModule(arc.Base{Name: "m", Source: textSource("func f() {}")})
			hiddenModule := newModule(arc.Base{
				Name:   "lsp_hidden_path",
				Source: textSource("func f() {}"),
			})
			graph := newModule(arc.Base{
				Name:   "lsp_graph_path",
				Source: arc.Source{Variant: arc.GraphSource{}},
			})
			automation := newAutomation(arc.Base{
				Name:   "lsp_automation_path",
				Source: textSource("func f() {}"),
			})
			lib := directory.Directory{Name: "lsp_lib"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.NewWriter(tx)
				root := rootDirectory(ctx, tx).Key
				if err := w.CreateDirectory(ctx, &lib, root); err != nil {
					return err
				}
				for _, a := range []*arc.Arc{&listed, &hiddenModule, &graph, &automation} {
					if err := w.Create(ctx, a); err != nil {
						return err
					}
				}
				return w.Move(ctx, root, lib.Key, listed.ResourceID())
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					w := svc.NewWriter(tx)
					if err := w.Delete(
						ctx,
						listed.Base().Key,
						hiddenModule.Base().Key,
						graph.Base().Key,
						automation.Base().Key,
					); err != nil {
						return err
					}
					return w.DeleteDirectory(ctx, lib.Key)
				})).To(Succeed())
			})
			server, _ := open(ctx, hide(hiddenModule.ResourceID()), "import lsp_")
			items := Completion(server, ctx, docURI, 0, 11).Items
			Expect(HasCompletion(items, "lsp_lib.m")).To(BeTrue())
			Expect(HasCompletion(items, "lsp_hidden_path")).To(BeFalse())
			Expect(HasCompletion(items, "lsp_graph_path")).To(BeFalse())
			Expect(HasCompletion(items, "lsp_automation_path")).To(BeFalse())
		})

		It("Should read the access again for each analysis", func(ctx SpecContext) {
			var revealed atomic.Bool
			server, client := open(
				ctx,
				func(context.Context) (func(resource.ID) bool, error) {
					shown := revealed.Load()
					return func(id resource.ID) bool {
						return shown || id != hidden.ResourceID()
					}, nil
				},
				"func test() {\n\tx := lsp_hidden_ch\n}",
			)
			Expect(client.Diagnostics()).To(HaveLen(1))
			revealed.Store(true)
			ChangeDocument(
				server,
				ctx,
				docURI,
				"func test() {\n\ty := lsp_hidden_ch\n}",
				2,
			)
			Eventually(client.Diagnostics).Should(BeEmpty())
		})

		It("Should report an error of the access read", func(ctx SpecContext) {
			_, client := open(
				ctx,
				func(context.Context) (func(resource.ID) bool, error) {
					return nil, errors.New("access read failed")
				},
				"func test() {\n\tx := lsp_visible_ch\n}",
			)
			Expect(client.Diagnostics()).To(ContainElement(WithTransform(
				DiagnosticMessage,
				ContainSubstring("access read failed"),
			)))
		})
	})
})

var _ = Describe("NewRoot", func() {
	var ch channel.Channel
	BeforeEach(func(ctx SpecContext) {
		ch = channel.Channel{
			Name:     "new_root_ch",
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).
				To(Succeed())
		})
	})

	It("Should resolve a channel by the name it first had", func(ctx SpecContext) {
		root := svc.NewRoot(db, &channel.Lookups{})
		Expect(root.Resolve(ctx, "new_root_ch")).To(HaveField("ID", int(ch.Key())))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).
				Rename(ctx, ch.Key(), "new_root_ch_2", false)
		})).
			To(Succeed())
		Expect(root.Resolve(ctx, "new_root_ch")).To(HaveField("ID", int(ch.Key())))
	})

	It("Should resolve a renamed channel by its new name", func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).
				Rename(ctx, ch.Key(), "new_root_ch_3", false)
		})).
			To(Succeed())
		root := svc.NewRoot(db, &channel.Lookups{})
		Expect(root.Resolve(ctx, "new_root_ch_3")).To(HaveField("ID", int(ch.Key())))
		Expect(root.Resolve(ctx, "new_root_ch")).Error().
			To(BeAssignableToTypeOf(&symbol.UndefinedSymbolError{}))
	})
})

var _ = Describe("TaskKeys", func() {
	It("Should return the tasks of every Arc", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		deployed := newAutomation(arc.Base{
			Name:   "arc-task-keys",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		idle := newAutomation(arc.Base{
			Name:   "arc-no-task-keys",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		w := svc.NewWriter(tx)
		Expect(w.Create(ctx, &deployed)).To(Succeed())
		Expect(w.Create(ctx, &idle)).To(Succeed())
		t := MustSucceed(w.UpdateTask(
			ctx,
			deployed.Base().Key,
			arc.TaskUpdate{Rack: new(testRack.Key)},
		))
		Expect(svc.TaskKeys(
			ctx,
			tx,
			deployed.Base().Key,
			idle.Base().Key,
			uuid.New(),
		)).To(Equal([]task.Key{t.Key}))
	})
})

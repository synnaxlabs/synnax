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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"go.lsp.dev/uri"
)

var _ = Describe("CompileProgram", func() {
	It("Should retrieve and compile an Arc with a valid graph", func(ctx SpecContext) {
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
			Expect(svc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
		})

		result := MustSucceed(svc.CompileProgram(ctx, a.Base().Key))
		Expect(result.Key).To(Equal(a.Base().Key))
		Expect(result.Program.IR).ToNot(BeNil())
	})

	It("Should refuse a module, which never runs", func(ctx SpecContext) {
		m := newModule(arc.Base{
			Name: "never_runs",
			Source: arc.Source{
				Variant: arc.TextSource{Text: text.Text{Raw: "func f() {}"}},
			},
		})
		Expect(svc.NewWriter(tx).Create(ctx, &m)).To(Succeed())
		Expect(tx.Commit(ctx)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.NewWriter(nil).Delete(ctx, m.Base().Key)).To(Succeed())
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
			Expect(svc.NewWriter(nil).Delete(ctx, a.Base().Key)).To(Succeed())
		})

		Expect(svc.CompileProgram(ctx, a.Base().Key)).Error().
			To(MatchError(ContainSubstring("edge target node 'nonexistent' not found")))
	})
})

var _ = Describe("NewLSP", func() {
	It("Should create an LSP server without error", func(ctx SpecContext) {
		server := MustSucceed(svc.NewLSP())
		Expect(server).ToNot(BeNil())
	})

	It("Should create multiple independent LSP servers", func(ctx SpecContext) {
		server1 := MustSucceed(svc.NewLSP())
		server2 := MustSucceed(svc.NewLSP())
		Expect(server1).ToNot(BeIdenticalTo(server2))
	})

	It("Should republish diagnostics when channels change", func(ctx SpecContext) {
		server := MustSucceed(svc.NewLSP())
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
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())

		Eventually(func() int {
			return len(client.Diagnostics())
		}).Should(Equal(0))
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
		Expect(channelSvc.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(channelSvc.NewWriter(nil).Delete(ctx, ch.Key(), false)).
				To(Succeed())
		})
	})

	It("Should resolve a channel by the name it first had", func(ctx SpecContext) {
		root := svc.NewRoot(nil)
		Expect(root.Resolve(ctx, "new_root_ch")).To(HaveField("ID", int(ch.Key())))
		Expect(channelSvc.NewWriter(nil).Rename(ctx, ch.Key(), "new_root_ch_2", false)).
			To(Succeed())
		Expect(root.Resolve(ctx, "new_root_ch")).To(HaveField("ID", int(ch.Key())))
	})

	It("Should resolve a renamed channel by its new name", func(ctx SpecContext) {
		Expect(channelSvc.NewWriter(nil).Rename(ctx, ch.Key(), "new_root_ch_3", false)).
			To(Succeed())
		root := svc.NewRoot(nil)
		Expect(root.Resolve(ctx, "new_root_ch_3")).To(HaveField("ID", int(ch.Key())))
		Expect(root.Resolve(ctx, "new_root_ch")).Error().
			To(BeAssignableToTypeOf(&symbol.UndefinedSymbolError{}))
	})
})

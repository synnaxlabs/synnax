// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package authority_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/authority"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Authority", func() {
	var ctx = context.Background()

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("NewFactory", func() {
		It("Should create factory with state", func() {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth", Type: "set_authority"}},
				Functions: []graph.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, authority.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			factory := authority.NewFactory(s)
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var (
			factory node.Factory
			s       *state.State
		)
		BeforeEach(func() {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth", Type: "set_authority"}},
				Functions: []graph.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, authority.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
			factory = authority.NewFactory(s)
		})
		It("Should create node for set_authority type", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			Expect(MustSucceed(factory.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should create node for set_authority with a non-uint8 channel", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(99)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Channel).ToNot(BeNil())
			Expect(*changes[0].Channel).To(Equal(uint32(99)))
			Expect(changes[0].Authority).To(Equal(uint8(200)))
		})
		It("Should return NotFound for unknown type", func() {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("set_auth"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should parse channel config with specific channel", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
			// Verify by exercising the node and checking the authority change
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Channel).ToNot(BeNil())
			Expect(*changes[0].Channel).To(Equal(uint32(42)))
		})
		It("Should parse channel config with zero (global)", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(150)},
						{Name: "channel", Type: types.U8(), Value: uint32(0)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			// Verify by exercising the node and checking the authority change
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Channel).To(BeNil())
		})
	})

	Describe("Next", func() {
		var (
			s       *state.State
			factory node.Factory
			outputs []string
		)
		BeforeEach(func() {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth", Type: "set_authority"}},
				Functions: []graph.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, authority.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
			factory = authority.NewFactory(s)
			outputs = []string{}
		})

		It("Should buffer per-channel authority change", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(200)))
			Expect(changes[0].Channel).ToNot(BeNil())
			Expect(*changes[0].Channel).To(Equal(uint32(42)))
		})

		It("Should buffer global authority change", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(150)},
						{Name: "channel", Type: types.U8(), Value: uint32(0)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(150)))
			Expect(changes[0].Channel).To(BeNil())
		})

		It("Should buffer authority change for a non-uint8 channel", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(255)},
						{Name: "channel", Type: types.U8(), Value: uint32(99)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(string) {}})
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(255)))
			Expect(changes[0].Channel).ToNot(BeNil())
			Expect(*changes[0].Channel).To(Equal(uint32(99)))
		})

		It("Should fire only once before Reset", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}
			n.Next(nCtx)
			n.Next(nCtx)
			n.Next(nCtx)
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
		})

		It("Should not call MarkChanged", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(output string) {
				outputs = append(outputs, output)
			}})
			Expect(outputs).To(BeEmpty())
		})
	})

	Describe("Reset", func() {
		var (
			s       *state.State
			factory node.Factory
		)
		BeforeEach(func() {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth", Type: "set_authority"}},
				Functions: []graph.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, authority.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s = state.New(state.Config{IR: analyzed})
			factory = authority.NewFactory(s)
		})

		It("Should allow re-fire after Reset", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}
			n.Next(nCtx)
			changes := s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
			n.Reset()
			n.Next(nCtx)
			changes = s.FlushAuthorityChanges()
			Expect(changes).To(HaveLen(1))
		})

		It("Should produce same authority on re-fire", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			nCtx := node.Context{Context: ctx, MarkChanged: func(string) {}}
			n.Next(nCtx)
			first := s.FlushAuthorityChanges()
			Expect(first).To(HaveLen(1))
			n.Reset()
			n.Next(nCtx)
			second := s.FlushAuthorityChanges()
			Expect(second).To(HaveLen(1))
			Expect(second[0].Authority).To(Equal(first[0].Authority))
			Expect(*second[0].Channel).To(Equal(*first[0].Channel))
		})
	})

	Describe("IsOutputTruthy", func() {
		It("Should always return false", func() {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth", Type: "set_authority"}},
				Functions: []graph.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, authority.SymbolResolver)
			Expect(diagnostics.Ok()).To(BeTrue())
			s := state.New(state.Config{IR: analyzed})
			factory := authority.NewFactory(s)
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_authority",
					Config: types.Params{
						{Name: "value", Type: types.U8(), Value: uint8(200)},
						{Name: "channel", Type: types.U8(), Value: uint32(42)},
					},
				},
				State: s.Node("set_auth"),
			}
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n.IsOutputTruthy("")).To(BeFalse())
			Expect(n.IsOutputTruthy("output")).To(BeFalse())
			Expect(n.IsOutputTruthy("anything")).To(BeFalse())
		})
	})

	Describe("SymbolResolver", func() {
		It("Should resolve set_authority symbol", func() {
			sym, ok := authority.SymbolResolver["set_authority"]
			Expect(ok).To(BeTrue())
			Expect(sym.Name).To(Equal("set_authority"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should have optional input", func() {
			sym := authority.SymbolResolver["set_authority"]
			Expect(sym.Type.Inputs).To(HaveLen(1))
			Expect(sym.Type.Inputs[0].Name).To(Equal(ir.DefaultOutputParam))
			Expect(sym.Type.Inputs[0].Value).To(Equal(uint8(0)))
		})
		It("Should have config params", func() {
			sym := authority.SymbolResolver["set_authority"]
			Expect(sym.Type.Config).To(HaveLen(2))
			Expect(sym.Type.Config[0].Name).To(Equal("value"))
			Expect(sym.Type.Config[1].Name).To(Equal("channel"))
		})
	})
})

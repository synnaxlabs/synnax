// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/control"
	. "github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Control", func() {
	// setAuthority declares the native's input shape for building test configs.
	setAuthority := NodeSpec{Type: "set_authority", Inputs: types.Params{
		{Name: "value", Type: types.U8()},
		{Name: "channel", Type: types.WriteChan(types.U8())},
	}}
	// constConfig helper builds a set_authority config whose inputs are all
	// consts, compiled from a graph program (covering the graph -> State path).
	constConfig := func(ctx context.Context, value uint8, channel uint32) node.Config {
		return GraphConfig(ctx, NewGraphRoot(nil), "set_authority",
			msgpack.EncodedJSON{"value": value, "channel": channel})
	}

	Describe("NewModule", func() {
		It("Should create factory with state", func(ctx SpecContext) {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth"}},
				Inputs:    map[string]msgpack.EncodedJSON{"set_auth": {"type": "set_authority"}},
				Functions: []ir.Function{{Key: "set_authority"}},
			}
			inter, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			_ = node.New(inter)
			factory := control.NewHost(&control.ProgramState{})
			Expect(factory).ToNot(BeNil())
		})
	})

	Describe("Factory.Create", func() {
		var (
			factory        node.Factory
			s              *node.ProgramState
			authorityState *control.ProgramState
		)
		BeforeEach(func(ctx SpecContext) {
			g := graph.Graph{
				Nodes:     []graph.Node{{Key: "set_auth"}},
				Inputs:    map[string]msgpack.EncodedJSON{"set_auth": {"type": "set_authority"}},
				Functions: []ir.Function{{Key: "set_authority"}},
			}
			analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue())
			s = node.New(analyzed)
			authorityState = &control.ProgramState{}
			factory = control.NewHost(authorityState)
		})
		It("Should create node for set_authority type", func(ctx SpecContext) {
			constCfg := constConfig(ctx, 200, 42)
			Expect(MustSucceed(factory.Create(ctx, constCfg))).ToNot(BeNil())
			varCfg := setAuthority.Config(NewVarInput[uint8](200), uint32(42))
			Expect(MustSucceed(factory.Create(ctx, varCfg))).ToNot(BeNil())
		})
		It("Should create node for control.set_authority via CompoundFactory", func(ctx SpecContext) {
			compound := node.CompoundFactory{factory}
			cfg := constConfig(ctx, 200, 42)
			cfg.Node.Type = "control.set_authority"
			Expect(MustSucceed(compound.Create(ctx, cfg))).ToNot(BeNil())
		})
		It("Should return NotFound for unknown type", func(ctx SpecContext) {
			cfg := node.Config{
				Node:  ir.Node{Type: "unknown"},
				State: s.Node("set_auth"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should error when an input value is invalid", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type:   "set_authority",
					Inputs: types.Params{{Name: "value", Type: types.U8(), Value: []any{1}}},
				},
				State: s.Node("set_auth"),
			}
			Expect(factory.Create(ctx, cfg)).Error().To(BeAValidationPathError())
		})
	})

	Describe("Next", func() {
		var (
			authorityState *control.ProgramState
			factory        node.Factory
			outputs        []string
		)
		BeforeEach(func() {
			authorityState = &control.ProgramState{}
			factory = control.NewHost(authorityState)
			outputs = []string{}
		})

		DescribeTable("Should buffer a per-channel authority change",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
				changes := authorityState.Flush()
				Expect(changes).To(HaveLen(1))
				Expect(changes[0].Authority).To(Equal(uint8(200)))
				Expect(changes[0].Channel).ToNot(BeNil())
				Expect(*changes[0].Channel).To(Equal(uint32(42)))
			},
			Entry("const inputs", uint8(200), uint32(42)),
			Entry("var value", VarOf(uint8(200)), uint32(42)),
		)

		DescribeTable("Should buffer a global authority change",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
				changes := authorityState.Flush()
				Expect(changes).To(HaveLen(1))
				Expect(changes[0].Authority).To(Equal(uint8(150)))
				Expect(changes[0].Channel).To(BeNil())
			},
			Entry("const inputs", uint8(150), uint32(0)),
			Entry("var value", VarOf(uint8(150)), uint32(0)),
		)

		DescribeTable("Should fire only once before Reset",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				nCtx := node.Context{Context: ctx, MarkChanged: func(int) {}}
				n.Next(nCtx)
				n.Next(nCtx)
				n.Next(nCtx)
				changes := authorityState.Flush()
				Expect(changes).To(HaveLen(1))
				Expect(changes[0].Authority).To(Equal(uint8(200)))
			},
			Entry("const inputs", uint8(200), uint32(42)),
			Entry("var value", VarOf(uint8(200)), uint32(42)),
		)

		DescribeTable("Should not call MarkChanged",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				n.Next(node.Context{Context: ctx, MarkChanged: func(int) {
					// setAuthority declares no outputs; MarkChanged should never fire.
					outputs = append(outputs, "called")
				}})
				Expect(outputs).To(BeEmpty())
			},
			Entry("const inputs", uint8(200), uint32(42)),
			Entry("var value", VarOf(uint8(200)), uint32(42)),
		)

		It("Should read params through a graph-compiled config", func(ctx SpecContext) {
			n := MustSucceed(factory.Create(ctx, constConfig(ctx, 200, 42)))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			changes := authorityState.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(200)))
			Expect(*changes[0].Channel).To(Equal(uint32(42)))
		})

		It("Should use the var's declared initial before any write", func(ctx SpecContext) {
			cfg := setAuthority.Config(NewVarInput[uint8](5), uint32(42))
			n := MustSucceed(factory.Create(ctx, cfg))
			n.Next(node.Context{Context: ctx, MarkChanged: func(int) {}})
			changes := authorityState.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(5)))
		})

		It("Should not re-fire when the var changes without a Reset", func(ctx SpecContext) {
			v := NewVarInput[uint8](1)
			cfg := setAuthority.Config(v, uint32(42))
			v.Set(77)
			n := MustSucceed(factory.Create(ctx, cfg))
			nCtx := node.Context{Context: ctx, MarkChanged: func(int) {}}
			n.Next(nCtx)
			v.Set(33)
			n.Next(nCtx)
			changes := authorityState.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(77)))
		})

		It("Should read the latest var value on re-fire after Reset", func(ctx SpecContext) {
			v := NewVarInput[uint8](1)
			cfg := setAuthority.Config(v, uint32(42))
			v.Set(77)
			n := MustSucceed(factory.Create(ctx, cfg))
			nCtx := node.Context{Context: ctx, MarkChanged: func(int) {}}
			n.Next(nCtx)
			Expect(authorityState.Flush()[0].Authority).To(Equal(uint8(77)))
			n.Reset()
			v.Set(33)
			n.Next(nCtx)
			changes := authorityState.Flush()
			Expect(changes).To(HaveLen(1))
			Expect(changes[0].Authority).To(Equal(uint8(33)))
		})
	})

	Describe("Reset", func() {
		var (
			authorityState *control.ProgramState
			factory        node.Factory
		)
		BeforeEach(func() {
			authorityState = &control.ProgramState{}
			factory = control.NewHost(authorityState)
		})

		DescribeTable("Should allow re-fire after Reset",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				nCtx := node.Context{Context: ctx, MarkChanged: func(int) {}}
				n.Next(nCtx)
				changes := authorityState.Flush()
				Expect(changes).To(HaveLen(1))
				n.Reset()
				n.Next(nCtx)
				changes = authorityState.Flush()
				Expect(changes).To(HaveLen(1))
			},
			Entry("const inputs", uint8(200), uint32(42)),
			Entry("var value", VarOf(uint8(200)), uint32(42)),
		)

		DescribeTable("Should produce same authority on re-fire",
			func(ctx SpecContext, value, channel any) {
				n := MustSucceed(factory.Create(ctx, setAuthority.Config(value, channel)))
				nCtx := node.Context{Context: ctx, MarkChanged: func(int) {}}
				n.Next(nCtx)
				first := authorityState.Flush()
				Expect(first).To(HaveLen(1))
				Expect(first[0].Authority).To(Equal(uint8(200)))
				n.Reset()
				n.Next(nCtx)
				second := authorityState.Flush()
				Expect(second).To(HaveLen(1))
				Expect(second[0].Authority).To(Equal(first[0].Authority))
				Expect(*second[0].Channel).To(Equal(*first[0].Channel))
			},
			Entry("const inputs", uint8(200), uint32(42)),
			Entry("var value", VarOf(uint8(200)), uint32(42)),
		)
	})

	Describe("IsOutputTruthy", func() {
		It("Should always return false", func(ctx SpecContext) {
			factory := control.NewHost(&control.ProgramState{})
			cfg := constConfig(ctx, 200, 42)
			n := MustSucceed(factory.Create(ctx, cfg))
			Expect(n.IsOutputTruthy(0)).To(BeFalse())
			Expect(n.IsOutputTruthy(1)).To(BeFalse())
			Expect(n.IsOutputTruthy(-1)).To(BeFalse())
		})
	})

	Describe("Symbols", func() {
		var root *symbol.Symbol
		BeforeEach(func() { root = symbol.NewRoot(nil, control.NewSymbols()) })
		bare := func(ctx context.Context, name string) *symbol.Symbol {
			return MustSucceed(root.Resolve(ctx, name, symbol.IncludeInternal))
		}
		It("Should expose bare set_authority symbol", func(ctx SpecContext) {
			sym := bare(ctx, "set_authority")
			Expect(sym.Name).To(Equal("set_authority"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should expose qualified control.set_authority symbol", func(ctx SpecContext) {
			mod := MustSucceed(root.Resolve(ctx, "control", symbol.IncludeInternal))
			sym := MustSucceed(mod.Resolve(ctx, "set_authority", symbol.IncludeInternal))
			Expect(sym.Name).To(Equal("set_authority"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})
		It("Should declare unified inputs with an activation trigger", func(ctx SpecContext) {
			sym := bare(ctx, "set_authority")
			Expect(sym.Type.Inputs).To(HaveLen(3))
			Expect(sym.Type.Inputs[0].Name).To(Equal(ir.DefaultOutputParam))
			Expect(sym.Type.Inputs[0].Value).To(Equal(uint8(0)))
			Expect(sym.Type.Inputs[1].Name).To(Equal("value"))
			Expect(sym.Type.Inputs[2].Name).To(Equal("channel"))
			Expect(sym.Trigger).To(Equal(symbol.TriggerInput(ir.DefaultOutputParam)))
		})
	})
})

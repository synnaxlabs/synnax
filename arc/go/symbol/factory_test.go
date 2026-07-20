// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factories", func() {
	Describe("InternalHostFunc", func() {
		It("Should build a KindFunction symbol flagged Internal and Exec=WASM", func() {
			sym := symbol.InternalHostFunc(
				"channels.read",
				types.Params{{Name: "key", Type: types.U32()}},
				types.Params{{Name: "value", Type: types.F64()}},
			)
			Expect(sym.Name).To(Equal("channels.read"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
			Expect(sym.Internal).To(BeTrue())
			Expect(sym.Exec).To(Equal(symbol.ExecWASM))
		})

		It("Should construct the symbol's Type from the given inputs and outputs", func() {
			inputs := types.Params{{Name: "key", Type: types.U32()}}
			outputs := types.Params{{Name: "value", Type: types.F64()}}
			sym := symbol.InternalHostFunc("read", inputs, outputs)
			Expect(sym.Type.Kind).To(Equal(types.KindFunction))
			Expect(sym.Type.FunctionProperties.Inputs).To(Equal(inputs))
			Expect(sym.Type.FunctionProperties.Outputs).To(Equal(outputs))
		})

		It("Should accept empty input and output params", func() {
			sym := symbol.InternalHostFunc("noop", nil, nil)
			Expect(sym.Type.FunctionProperties.Inputs).To(BeEmpty())
			Expect(sym.Type.FunctionProperties.Outputs).To(BeEmpty())
		})
	})

	Describe("QualifiedName", func() {
		It("Should prefix the module name for a module member", func() {
			mod := &symbol.Symbol{Name: "math", Kind: symbol.KindModule}
			mod.AddChild(&symbol.Symbol{Name: "avg", Kind: symbol.KindFunction})
			Expect(mod.FindChild("avg").QualifiedName()).To(Equal("math.avg"))
		})

		It("Should return the bare name for a top-level symbol", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			sym := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(sym.QualifiedName()).To(Equal("x"))
		})

		It("Should return the bare name when the parent is not a module", func(bCtx SpecContext) {
			rootScope := symbol.NewRoot(nil, nil)
			funcScope := MustSucceed(rootScope.Add(
				bCtx,
				symbol.Symbol{Name: "f", Kind: symbol.KindFunction},
			))
			inner := MustSucceed(funcScope.Add(
				bCtx,
				symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()},
			))
			Expect(inner.QualifiedName()).To(Equal("x"))
		})

		It("Should return the bare name for a symbol with no parent", func() {
			lonely := &symbol.Symbol{Name: "orphan"}
			Expect(lonely.QualifiedName()).To(Equal("orphan"))
		})
	})

	Describe("AutoImportModules", func() {
		It("Should install a KindModuleAlias child for each module in the ambient prelude", func() {
			timeMod := &symbol.Symbol{Name: "time", Kind: symbol.KindModule}
			timeMod.AddChild(&symbol.Symbol{Name: "now", Kind: symbol.KindFunction, Type: types.F64()})
			mathMod := &symbol.Symbol{Name: "math", Kind: symbol.KindModule}
			mathMod.AddChild(&symbol.Symbol{Name: "avg", Kind: symbol.KindFunction, Type: types.F64()})
			ambient := &symbol.Symbol{Kind: symbol.KindAmbient}
			ambient.AddChild(timeMod)
			ambient.AddChild(mathMod)
			root := symbol.NewRoot(nil, nil)
			ambient.AddChild(root)

			symbol.AutoImportModules(root)

			timeAlias := root.FindChild("time")
			Expect(timeAlias).ToNot(BeNil())
			Expect(timeAlias.Kind).To(Equal(symbol.KindModuleAlias))
			Expect(timeAlias.Target).To(Equal(timeMod))
			mathAlias := root.FindChild("math")
			Expect(mathAlias).ToNot(BeNil())
			Expect(mathAlias.Target).To(Equal(mathMod))
		})

		It("Should be a no-op when the root has no Parent", func() {
			orphan := &symbol.Symbol{Kind: symbol.KindBlock}
			symbol.AutoImportModules(orphan)
			Expect(orphan.Children()).To(BeEmpty())
		})

		It("Should skip non-module children of the ambient prelude", func() {
			ambient := &symbol.Symbol{Kind: symbol.KindAmbient}
			ambient.AddChild(&symbol.Symbol{Name: "value", Kind: symbol.KindVariable})
			ambient.AddChild(&symbol.Symbol{Name: "time", Kind: symbol.KindModule})
			root := symbol.NewRoot(nil, nil)
			ambient.AddChild(root)

			symbol.AutoImportModules(root)

			Expect(root.FindChild("value")).To(BeNil())
			Expect(root.FindChild("time")).ToNot(BeNil())
		})
	})

	Describe("NewRoot", func() {
		It("Should attach ambient globals as siblings of the root", func() {
			channel := &symbol.Symbol{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())}
			root := symbol.NewRoot(nil, []*symbol.Symbol{channel})
			Expect(root.Parent).ToNot(BeNil())
			Expect(root.Parent.Kind).To(Equal(symbol.KindAmbient))
			Expect(root.Parent.FindChild("sensor")).ToNot(BeNil())
		})

		It("Should shallow-copy each ambient global so callers can reuse them across roots", func() {
			original := &symbol.Symbol{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32())}
			root := symbol.NewRoot(nil, []*symbol.Symbol{original})
			attached := root.Parent.FindChild("sensor")
			Expect(attached.Parent).To(Equal(root.Parent))
			Expect(original.Parent).To(BeNil())
		})

		It("Should isolate Parent assignment between concurrent roots", func() {
			shared := &symbol.Symbol{Name: "shared", Kind: symbol.KindChannel, Type: types.Chan(types.F32())}
			rootA := symbol.NewRoot(nil, []*symbol.Symbol{shared})
			rootB := symbol.NewRoot(nil, []*symbol.Symbol{shared})
			attachedA := rootA.Parent.FindChild("shared")
			attachedB := rootB.Parent.FindChild("shared")
			Expect(attachedA).ToNot(BeIdenticalTo(attachedB))
			Expect(attachedA.Parent).To(BeIdenticalTo(rootA.Parent))
			Expect(attachedB.Parent).To(BeIdenticalTo(rootB.Parent))
		})

		It("Should consult the given dynamic resolver on root.Resolve misses", func(bCtx SpecContext) {
			resolver := &recordingResolver{}
			root := symbol.NewRoot(resolver, nil)
			_, _ = root.Resolve(bCtx, "missing")
			Expect(resolver.resolveCalls).To(Equal(1))
		})
	})
})

var _ = Describe("Value variable predicates", func() {
	// rwChan builds the read+write channel type a bare-channel alias carries.
	rwChan := func(elem types.Type) types.Type {
		t := types.Chan(elem)
		t.ChanDirection = types.ChanDirectionRead | types.ChanDirectionWrite
		return t
	}
	src := 7
	DescribeTable("classify a symbol by kind and data type",
		func(sym symbol.Symbol, valueVar, readWrite, reactive, literal, backsInternal bool) {
			Expect(sym.IsValueVariable()).To(Equal(valueVar))
			Expect(sym.IsChannelReadWrite()).To(Equal(readWrite))
			Expect(sym.IsReactive()).To(Equal(reactive))
			Expect(sym.IsLiteral()).To(Equal(literal))
			Expect(sym.BacksInternalChannel()).To(Equal(backsInternal))
		},
		Entry("literal := variable",
			symbol.Symbol{Kind: symbol.KindVariable, Type: types.I32()},
			true, false, false, true, true),
		Entry("stateful $= variable",
			symbol.Symbol{Kind: symbol.KindStatefulVariable, Type: types.F64()},
			true, false, false, true, true),
		Entry("bare-channel read/write alias",
			symbol.Symbol{Kind: symbol.KindVariable, Type: rwChan(types.F32()), SourceID: &src},
			true, true, false, false, false),
		Entry("reactive channel-read variable",
			symbol.Symbol{Kind: symbol.KindVariable, Type: types.ReadChan(types.F32())},
			true, false, true, false, true),
		Entry("chan-typed variable with unset direction classifies as reactive",
			symbol.Symbol{Kind: symbol.KindVariable, Type: types.Chan(types.F32())},
			true, false, true, false, true),
		Entry("stateful bare-channel read/write alias",
			symbol.Symbol{
				Kind: symbol.KindStatefulVariable, Type: rwChan(types.F32()),
				SourceID: &src,
			},
			true, true, false, false, false),
		Entry("stateful reactive channel-read variable",
			symbol.Symbol{Kind: symbol.KindStatefulVariable, Type: types.ReadChan(types.F32())},
			true, false, true, false, true),
		Entry("loop variable is not a value variable",
			symbol.Symbol{Kind: symbol.KindLoopVariable, Type: types.I32()},
			false, false, false, false, false),
		Entry("channel symbol is not a value variable",
			symbol.Symbol{Kind: symbol.KindChannel, Type: types.Chan(types.F32())},
			false, false, false, false, false),
		Entry("function symbol is not a value variable",
			symbol.Symbol{Kind: symbol.KindFunction},
			false, false, false, false, false),
	)
})

type recordingResolver struct {
	resolveCalls int
	searchCalls  int
}

func (r *recordingResolver) Resolve(_ context.Context, _ string) (*symbol.Symbol, error) {
	r.resolveCalls++
	return nil, errors.Wrap(query.ErrNotFound, "not found")
}

func (r *recordingResolver) Search(_ context.Context, _ string) ([]*symbol.Symbol, error) {
	r.searchCalls++
	return nil, nil
}

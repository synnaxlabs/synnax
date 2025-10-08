package runtime_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/arc/stratifier"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

type MockNode struct {
}

func (m MockNode) Next(ctx context.Context, onChange func(string)) {
	onChange("output")
}

var _ = Describe("Runtime", func() {
	var ctx context.Context = context.Background()
	It("Should run an arc program", func() {
		source := `
		stage double{} (input f32) f32 {
			return input * 2
		}
		sensor -> double{}
		`
		t := MustSucceed(text.Parse(text.Text{Raw: source}))
		Expect(t.AST).ToNot(BeNil())
		resolver := ir.MapResolver{
			"sensor": ir.Symbol{
				Name: "sensor",
				Kind: ir.KindChannel,
				Type: ir.Chan{ValueType: ir.F32{}},
			},
			"output": ir.Symbol{
				Name: "output",
				Kind: ir.KindChannel,
				Type: ir.Chan{ValueType: ir.F32{}},
			},
		}
		inter, res := text.Analyze(ctx, t, resolver)
		Expect(res.Ok()).To(BeTrue(), res.String())
		Expect(inter.Nodes).To(HaveLen(2))
		Expect(inter.Edges).To(HaveLen(1))

		var ok bool
		inter.Strata, ok = stratifier.Stratify(ctx, inter.Nodes, inter.Edges, &res)
		Expect(ok).To(BeTrue(), res.String())

		compilerOutput := MustSucceed(text.Compile(ctx, inter, compiler.DisableHostImport()))
		Expect(compilerOutput.WASM).ToNot(BeEmpty())
		wasmRuntime := wazero.NewRuntime(ctx)
		wasmModule := MustSucceed(wasmRuntime.Instantiate(ctx, compilerOutput.WASM))

		state := &runtime.State{Outputs: map[ir.Handle]telem.Series{}}
		for _, edge := range inter.Edges {
			state.Outputs[edge.Source] = telem.NewSeriesV[float32](0)
		}
		nodes := make(map[string]runtime.Node)
		for _, node := range inter.Nodes {
			s, ok := inter.GetStage(node.Type)
			if ok {
				baseWasmFunc := wasmModule.ExportedFunction(s.Key)
				wrapped := wasm.WrapFunction(
					baseWasmFunc,
					wasmModule.Memory(),
					s.Outputs,
					compilerOutput.OutputMemoryBases[node.Key],
				)

				nodes[node.Key] = &wasm.Node{
					Node: node,
					Wasm: wrapped,
					Inputs: lo.Filter(inter.Edges, func(item ir.Edge, index int) bool {
						return item.Target.Node == node.Key
					}),
					Outputs: lo.Filter(inter.Edges, func(item ir.Edge, index int) bool {
						return item.Source.Node == node.Key
					}),
					State:  state,
					Params: make([]uint64, len(s.Outputs.Keys)),
				}
			}
		}

		nodes["on_0"] = MockNode{}

		s := runtime.NewScheduler(inter, nodes)
		state.Outputs[inter.Edges[0].Source] = telem.NewSeriesV[float32](8)
		h := ir.Handle{Node: "double_0", Param: "output"}
		state.Outputs[h] = telem.NewSeriesV[float32](0)
		c := 1_000_000
		start := telem.Now()
		for range c {
			s.MarkChanged("on_0")
			s.Next(ctx)
		}
		fmt.Println(telem.Since(start) / telem.TimeSpan(c))
	})
})

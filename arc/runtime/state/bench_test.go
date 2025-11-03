package state_test

import (
	"testing"

	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

func BenchmarkRefreshInputsSingleInput(b *testing.B) {
	g := graph.Graph{
		Nodes: graph.Nodes{
			{Key: "source", Type: "source"},
			{Key: "target", Type: "target"},
		},
		Functions: []graph.Function{
			{
				Key: "source",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.F32()},
				},
			},
			{
				Key: "target",
				Inputs: types.Params{
					{Name: ir.DefaultInputParam, Type: types.F32()},
				},
			},
		},
		Edges: []ir.Edge{
			{
				Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "target", Param: ir.DefaultInputParam},
			},
		},
	}
	inter, diagnostics := graph.Analyze(ctx, g, nil)
	if !diagnostics.Ok() {
		b.Fatalf("Failed to analyze graph: %s", diagnostics.String())
	}
	cfg := state.Config{IR: inter}
	s := state.New(cfg)
	sourceNode := s.Node("source")
	targetNode := s.Node("target")
	*sourceNode.Output(0) = telem.NewSeriesV[float32](0)
	*sourceNode.OutputTime(0) = telem.NewSeriesSecondsTSV(1)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telem.SetValueAt[float32](*sourceNode.Output(0), 0, float32(i))
		telem.SetValueAt[telem.TimeStamp](
			*sourceNode.OutputTime(0),
			0,
			telem.TimeStamp(i+1)*telem.SecondTS,
		)
		if !targetNode.RefreshInputs() {
			b.Fatal("Failed to refresh inputs")
		}
	}
}

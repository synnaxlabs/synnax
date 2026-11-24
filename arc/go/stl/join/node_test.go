// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

const nodeType = "join_spec"

// joinSetup is a join node of two f64 inputs, a and b, whose primary is a.
type joinSetup struct {
	state *node.ProgramState
	n     node.Node
	// self counts the calls to MarkSelfChanged.
	self int
}

// openJoin opens a join of a and b with a as the primary, and b pairing by arrival when
// arrival is set.
func openJoin(ctx SpecContext, arrival bool) *joinSetup {
	GinkgoHelper()
	inputs := types.Params{
		{Name: join.InputParam(0), Type: types.F64()},
		{Name: join.InputParam(1), Type: types.F64()},
	}
	if arrival {
		inputs = append(inputs, types.Param{
			Name:  join.ArrivalParam(1),
			Type:  types.U8(),
			Value: uint8(1),
		})
	}
	inputs = append(inputs, types.Param{
		Name:  join.PrimaryParam,
		Type:  types.I64(),
		Value: int64(0),
	})
	sym := &symbol.Symbol{
		Name: nodeType,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(types.FunctionProperties{
			Inputs: inputs,
			Outputs: types.Params{
				{Name: join.OutputParam(0), Type: types.F64()},
				{Name: join.OutputParam(1), Type: types.F64()},
			},
		}),
	}
	g := graph.Graph{
		Nodes: []graph.Node{{Key: "a"}, {Key: "b"}, {Key: "join"}},
		Inputs: map[string]msgpack.EncodedJSON{
			"a":    {"type": "a"},
			"b":    {"type": "b"},
			"join": {"type": nodeType},
		},
		Edges: graph.Edges{
			{
				Source: ir.Handle{Node: "a", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "join", Param: join.InputParam(0)},
			},
			{
				Source: ir.Handle{Node: "b", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "join", Param: join.InputParam(1)},
			},
		},
		Functions: []ir.Function{
			{
				Key:     "a",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			},
			{
				Key:     "b",
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.F64()}},
			},
		},
	}
	analyzed, diagnostics := graph.Analyze(ctx, g, NewGraphRoot(nil, *sym))
	Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
	s := node.New(analyzed)
	h := join.NewHost(nodeType, telem.Now)
	n := MustSucceed(h.Create(node.Config{
		Node:    ir.Node{Key: "join", Type: nodeType, Inputs: inputs},
		State:   s.Node("join"),
		Program: program.Program{IR: analyzed},
	}))
	return &joinSetup{state: s, n: n}
}

// push sets the output of source to values at stamps, in seconds, with alignment.
func (j *joinSetup) push(
	source string,
	alignment telem.Alignment,
	values []float64,
	stamps ...telem.TimeStamp,
) {
	src := j.state.Node(source)
	data := telem.NewSeries(values)
	t := telem.NewSeriesSecondsTSV(stamps...)
	data.Alignment, t.Alignment = alignment, alignment
	*src.Output(0), *src.OutputTime(0) = data, t
	src.MarkFresh(0)
}

// next runs the join and returns the outputs it marked changed.
func (j *joinSetup) next(ctx SpecContext) []int {
	var changed []int
	j.n.Next(node.Context{
		Context:         ctx,
		MarkChanged:     func(i int) { changed = append(changed, i) },
		MarkSelfChanged: func() { j.self++ },
		ReportError:     func(err error) { Fail(err.Error()) },
	})
	return changed
}

// output returns output i of the join and its stamps.
func (j *joinSetup) output(i int) (telem.Series, telem.Series) {
	n := j.state.Node("join")
	return *n.Output(i), *n.OutputTime(i)
}

var _ = Describe("Node", func() {
	It("Should refuse a node type that is not its own", func() {
		h := join.NewHost(nodeType, telem.Now)
		Expect(h.Create(node.Config{Node: ir.Node{Type: "other"}})).Error().
			To(MatchError(query.ErrNotFound))
	})

	It("Should give every output the primary's stamps and alignment", func(
		ctx SpecContext,
	) {
		j := openJoin(ctx, false)
		j.push("b", 50, []float64{10, 20}, 1, 3)
		j.push("a", 100, []float64{1, 2, 3}, 2, 3, 4)
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		a, aTime := j.output(0)
		b, bTime := j.output(1)
		Expect(a).To(telem.MatchSeriesDataV[float64](1, 2, 3))
		Expect(b).To(telem.MatchSeriesDataV[float64](10, 20, 20))
		for _, s := range []telem.Series{a, aTime, b, bTime} {
			Expect(s.Alignment).To(Equal(telem.Alignment(100)))
			Expect(s.TimeRange).To(Equal(telem.TimeRange{
				Start: 2 * telem.SecondTS,
				End:   4*telem.SecondTS + 1,
			}))
		}
		Expect(bTime).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(2, 3, 4)))
		Expect(j.self).To(BeZero())
		Expect(j.n.(*join.Node).Dropped()).To(BeZero())
	})

	It("Should emit each run after a gap on its own step", func(ctx SpecContext) {
		j := openJoin(ctx, false)
		j.push("b", 0, []float64{10, 11, 12, 20}, 0, 1, 2, 20)
		j.push("a", 0, []float64{1, 2, 3, 4, 5}, 1, 2, 10, 11, 21)
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		Expect(j.self).To(Equal(1))
		first, firstTime := j.output(0)
		Expect(first).To(telem.MatchSeriesDataV[float64](1, 2))
		Expect(firstTime).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(1, 2)))
		Expect(first.Alignment).To(Equal(telem.Alignment(0)))
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		Expect(j.self).To(Equal(1))
		second, _ := j.output(0)
		b, _ := j.output(1)
		Expect(second).To(telem.MatchSeriesDataV[float64](5))
		Expect(b).To(telem.MatchSeriesDataV[float64](20))
		Expect(second.Alignment).To(Equal(telem.Alignment(4)))
		Expect(j.n.(*join.Node).Dropped()).To(Equal(2))
	})

	It("Should count the primary stamps that get no output", func(ctx SpecContext) {
		j := openJoin(ctx, false)
		j.push("b", 0, []float64{10}, 5)
		j.push("a", 0, []float64{1, 2}, 2, 3)
		Expect(j.next(ctx)).To(BeEmpty())
		Expect(j.n.(*join.Node).Dropped()).To(Equal(2))
		Expect(j.n.(*join.Node).Dropped()).To(BeZero())
	})

	It("Should pair an arrival input whatever stamps it carries", func(
		ctx SpecContext,
	) {
		j := openJoin(ctx, true)
		j.push("b", 0, []float64{10}, 9)
		j.push("a", 0, []float64{1, 2}, 2, 3)
		Expect(j.next(ctx)).To(ConsistOf(0, 1))
		b, _ := j.output(1)
		Expect(b).To(telem.MatchSeriesDataV[float64](10, 10))
	})
})

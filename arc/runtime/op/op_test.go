package op_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OP", func() {
	ctx := context.Background()
	DescribeTable("Binary Operators", func(
		opName string,
		a, b, expectedOutput telem.Series,
	) {
		f := op.NewFactory()
		s := &state.State{Outputs: map[ir.Handle]state.Output{}}
		lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
		rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
		outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}
		s.Outputs[lhsSourceHandle] = state.Output{Data: a}
		s.Outputs[rhsSourceHandle] = state.Output{Data: b}
		s.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: expectedOutput.DataType}}
		inter := ir.IR{
			Edges: []ir.Edge{
				{
					Source: lhsSourceHandle,
					Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
				},
				{
					Source: rhsSourceHandle,
					Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
				},
				{
					Source: ir.Handle{Node: "op", Param: ir.DefaultOutputParam},
					Target: outputHandle,
				},
			},
		}
		irNode := ir.Node{Key: "op", Type: opName}
		runtimeNode := MustSucceed(f.Create(ctx, node.Config{
			State:  s,
			Node:   irNode,
			Module: module.Module{IR: inter},
		}))
		changed := ""
		runtimeNode.Next(ctx, func(output string) {
			changed = output
		})
		Expect(changed).To(Equal(ir.DefaultOutputParam))
		res := s.Outputs[outputHandle].Data
		Expect(res).To(telem.MatchSeries(expectedOutput))
	},
		// Greater Than (gt)
		Entry("gt - float64 - all true",
			"gt",
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("gt - float64 - all false",
			"gt",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[uint8](0, 0, 0),
		),
		Entry("gt - float64 - mixed",
			"gt",
			telem.NewSeriesV[float64](1.0, 3.0, 2.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			telem.NewSeriesV[uint8](0, 1, 0),
		),
		Entry("gt - int64",
			"gt",
			telem.NewSeriesV[int64](5, 10, 3),
			telem.NewSeriesV[int64](3, 8, 3),
			telem.NewSeriesV[uint8](1, 1, 0),
		),
		Entry("gt - uint32",
			"gt",
			telem.NewSeriesV[uint32](100, 50, 75),
			telem.NewSeriesV[uint32](50, 60, 80),
			telem.NewSeriesV[uint8](1, 0, 0),
		),

		// Greater Than Or Equal (ge)
		Entry("ge - float64 - all true",
			"ge",
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("ge - float64 - with equal values",
			"ge",
			telem.NewSeriesV[float64](2.0, 3.0, 2.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("ge - int32 - mixed",
			"ge",
			telem.NewSeriesV[int32](5, 10, 3),
			telem.NewSeriesV[int32](5, 8, 4),
			telem.NewSeriesV[uint8](1, 1, 0),
		),

		// Less Than (lt)
		Entry("lt - float64 - all true",
			"lt",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("lt - float64 - all false",
			"lt",
			telem.NewSeriesV[float64](5.0, 6.0, 7.0),
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[uint8](0, 0, 0),
		),
		Entry("lt - int16 - mixed",
			"lt",
			telem.NewSeriesV[int16](1, 5, 10),
			telem.NewSeriesV[int16](2, 5, 9),
			telem.NewSeriesV[uint8](1, 0, 0),
		),

		// Less Than Or Equal (le)
		Entry("le - float32 - all true",
			"le",
			telem.NewSeriesV[float32](1.0, 2.0, 3.0),
			telem.NewSeriesV[float32](3.0, 4.0, 5.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("le - float32 - with equal values",
			"le",
			telem.NewSeriesV[float32](2.0, 3.0, 2.0),
			telem.NewSeriesV[float32](2.0, 2.0, 2.0),
			telem.NewSeriesV[uint8](1, 0, 1),
		),
		Entry("le - uint64 - mixed",
			"le",
			telem.NewSeriesV[uint64](5, 10, 3),
			telem.NewSeriesV[uint64](5, 8, 4),
			telem.NewSeriesV[uint8](1, 0, 1),
		),

		// Equal (eq)
		Entry("eq - float64 - all equal",
			"eq",
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			telem.NewSeriesV[float64](2.0, 2.0, 2.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("eq - float64 - none equal",
			"eq",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			telem.NewSeriesV[uint8](0, 0, 0),
		),
		Entry("eq - int8 - mixed",
			"eq",
			telem.NewSeriesV[int8](1, 2, 3),
			telem.NewSeriesV[int8](1, 5, 3),
			telem.NewSeriesV[uint8](1, 0, 1),
		),

		// Not Equal (ne)
		Entry("ne - float64 - all not equal",
			"ne",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			telem.NewSeriesV[uint8](1, 1, 1),
		),
		Entry("ne - float64 - some equal",
			"ne",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](1.0, 5.0, 3.0),
			telem.NewSeriesV[uint8](0, 1, 0),
		),
		Entry("ne - uint8 - mixed",
			"ne",
			telem.NewSeriesV[uint8](5, 10, 15),
			telem.NewSeriesV[uint8](5, 20, 30),
			telem.NewSeriesV[uint8](0, 1, 1),
		),

		// Addition (add)
		Entry("add - float64 - simple",
			"add",
			telem.NewSeriesV[float64](1.0, 2.0, 3.0),
			telem.NewSeriesV[float64](4.0, 5.0, 6.0),
			telem.NewSeriesV[float64](5.0, 7.0, 9.0),
		),
		Entry("add - int64 - positive",
			"add",
			telem.NewSeriesV[int64](10, 20, 30),
			telem.NewSeriesV[int64](5, 15, 25),
			telem.NewSeriesV[int64](15, 35, 55),
		),
		Entry("add - int32 - with negatives",
			"add",
			telem.NewSeriesV[int32](-5, 10, -3),
			telem.NewSeriesV[int32](8, -2, 7),
			telem.NewSeriesV[int32](3, 8, 4),
		),
		Entry("add - uint16 - simple",
			"add",
			telem.NewSeriesV[uint16](100, 200, 300),
			telem.NewSeriesV[uint16](50, 75, 125),
			telem.NewSeriesV[uint16](150, 275, 425),
		),

		// Subtraction (sub)
		Entry("sub - float64 - simple",
			"sub",
			telem.NewSeriesV[float64](10.0, 20.0, 30.0),
			telem.NewSeriesV[float64](3.0, 5.0, 10.0),
			telem.NewSeriesV[float64](7.0, 15.0, 20.0),
		),
		Entry("sub - int64 - with negatives",
			"sub",
			telem.NewSeriesV[int64](5, 10, 15),
			telem.NewSeriesV[int64](10, 3, 20),
			telem.NewSeriesV[int64](-5, 7, -5),
		),
		Entry("sub - float32 - decimals",
			"sub",
			telem.NewSeriesV[float32](5.5, 10.2, 15.9),
			telem.NewSeriesV[float32](2.3, 4.1, 8.9),
			telem.NewSeriesV[float32](3.2, 6.1, 7),
		),

		// Multiplication (mul)
		Entry("mul - float64 - simple",
			"mul",
			telem.NewSeriesV[float64](2.0, 3.0, 4.0),
			telem.NewSeriesV[float64](5.0, 6.0, 7.0),
			telem.NewSeriesV[float64](10.0, 18.0, 28.0),
		),
		Entry("mul - int64 - with zero",
			"mul",
			telem.NewSeriesV[int64](5, 0, 10),
			telem.NewSeriesV[int64](3, 7, 2),
			telem.NewSeriesV[int64](15, 0, 20),
		),
		Entry("mul - int32 - with negatives",
			"mul",
			telem.NewSeriesV[int32](-2, 4, -5),
			telem.NewSeriesV[int32](3, -3, 2),
			telem.NewSeriesV[int32](-6, -12, -10),
		),

		// Division (div)
		Entry("div - float64 - simple",
			"div",
			telem.NewSeriesV[float64](10.0, 20.0, 30.0),
			telem.NewSeriesV[float64](2.0, 4.0, 5.0),
			telem.NewSeriesV[float64](5.0, 5.0, 6.0),
		),
		Entry("div - int64 - integer division",
			"div",
			telem.NewSeriesV[int64](15, 20, 7),
			telem.NewSeriesV[int64](3, 4, 2),
			telem.NewSeriesV[int64](5, 5, 3),
		),
		Entry("div - float32 - decimals",
			"div",
			telem.NewSeriesV[float32](10.0, 15.0, 20.0),
			telem.NewSeriesV[float32](4.0, 3.0, 8.0),
			telem.NewSeriesV[float32](2.5, 5.0, 2.5),
		),

		// Test different length inputs (broadcasting behavior)
		Entry("gt - different lengths - a longer",
			"gt",
			telem.NewSeriesV[float64](5.0, 6.0, 7.0, 8.0),
			telem.NewSeriesV[float64](4.0, 5.0),
			telem.NewSeriesV[uint8](1, 1, 1, 1),
		),
		Entry("add - different lengths - b longer",
			"add",
			telem.NewSeriesV[int64](10, 20),
			telem.NewSeriesV[int64](1, 2, 3, 4),
			telem.NewSeriesV[int64](11, 22, 23, 24),
		),
		Entry("mul - different lengths - scalar-like",
			"mul",
			telem.NewSeriesV[float64](2.0),
			telem.NewSeriesV[float64](3.0, 4.0, 5.0),
			telem.NewSeriesV[float64](6.0, 8.0, 10.0),
		),
	)

	Describe("Time Propagation", func() {
		It("Should use LHS time when LHS is longer", func() {
			f := op.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}
			lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
			rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}

			lhsData := telem.NewSeriesV[float64](1.0, 2.0, 3.0, 4.0)
			lhsTime := telem.NewSeriesV[telem.TimeStamp](100, 200, 300, 400)
			rhsData := telem.NewSeriesV[float64](5.0, 6.0)
			rhsTime := telem.NewSeriesV[telem.TimeStamp](150, 250)

			s.Outputs[lhsSourceHandle] = state.Output{Data: lhsData, Time: lhsTime}
			s.Outputs[rhsSourceHandle] = state.Output{Data: rhsData, Time: rhsTime}
			s.Outputs[outputHandle] = state.Output{
				Data: telem.Series{DataType: telem.Float64T},
				Time: telem.Series{DataType: telem.TimeStampT},
			}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: lhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: rhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
			}

			irNode := ir.Node{Key: "op", Type: "add"}
			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changed := ""
			runtimeNode.Next(ctx, func(output string) {
				changed = output
			})

			Expect(changed).To(Equal(ir.DefaultOutputParam))
			res := s.Outputs[outputHandle]
			Expect(res.Data).To(telem.MatchSeries(telem.NewSeriesV[float64](6.0, 8.0, 9.0, 10.0)))
			Expect(res.Time).To(telem.MatchSeries(lhsTime))
		})

		It("Should use RHS time when RHS is longer", func() {
			f := op.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}
			lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
			rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}

			lhsData := telem.NewSeriesV[float64](1.0, 2.0)
			lhsTime := telem.NewSeriesV[telem.TimeStamp](100, 200)
			rhsData := telem.NewSeriesV[float64](5.0, 6.0, 7.0, 8.0)
			rhsTime := telem.NewSeriesV[telem.TimeStamp](150, 250, 350, 450)

			s.Outputs[lhsSourceHandle] = state.Output{Data: lhsData, Time: lhsTime}
			s.Outputs[rhsSourceHandle] = state.Output{Data: rhsData, Time: rhsTime}
			s.Outputs[outputHandle] = state.Output{
				Data: telem.Series{DataType: telem.Float64T},
				Time: telem.Series{DataType: telem.TimeStampT},
			}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: lhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: rhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
			}

			irNode := ir.Node{Key: "op", Type: "add"}
			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changed := ""
			runtimeNode.Next(ctx, func(output string) {
				changed = output
			})

			Expect(changed).To(Equal(ir.DefaultOutputParam))
			res := s.Outputs[outputHandle]
			Expect(res.Data).To(telem.MatchSeries(telem.NewSeriesV[float64](6.0, 8.0, 9.0, 10.0)))
			Expect(res.Time).To(telem.MatchSeries(rhsTime))
		})

		It("Should use LHS time when lengths are equal", func() {
			f := op.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}
			lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
			rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}

			lhsData := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
			lhsTime := telem.NewSeriesV[telem.TimeStamp](100, 200, 300)
			rhsData := telem.NewSeriesV[float64](4.0, 5.0, 6.0)
			rhsTime := telem.NewSeriesV[telem.TimeStamp](150, 250, 350)

			s.Outputs[lhsSourceHandle] = state.Output{Data: lhsData, Time: lhsTime}
			s.Outputs[rhsSourceHandle] = state.Output{Data: rhsData, Time: rhsTime}
			s.Outputs[outputHandle] = state.Output{
				Data: telem.Series{DataType: telem.Uint8T},
				Time: telem.Series{DataType: telem.TimeStampT},
			}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: lhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: rhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
			}

			irNode := ir.Node{Key: "op", Type: "gt"}
			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changed := ""
			runtimeNode.Next(ctx, func(output string) {
				changed = output
			})

			Expect(changed).To(Equal(ir.DefaultOutputParam))
			res := s.Outputs[outputHandle]
			Expect(res.Data).To(telem.MatchSeries(telem.NewSeriesV[uint8](0, 0, 0)))
			Expect(res.Time).To(telem.MatchSeries(lhsTime))
		})

		It("Should not produce output when LHS is empty", func() {
			f := op.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}
			lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
			rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}

			lhsData := telem.NewSeriesV[float64]()
			lhsTime := telem.NewSeriesV[telem.TimeStamp]()
			rhsData := telem.NewSeriesV[float64](5.0, 6.0)
			rhsTime := telem.NewSeriesV[telem.TimeStamp](150, 250)

			s.Outputs[lhsSourceHandle] = state.Output{Data: lhsData, Time: lhsTime}
			s.Outputs[rhsSourceHandle] = state.Output{Data: rhsData, Time: rhsTime}
			s.Outputs[outputHandle] = state.Output{
				Data: telem.Series{DataType: telem.Float64T},
				Time: telem.Series{DataType: telem.TimeStampT},
			}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: lhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: rhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
			}

			irNode := ir.Node{Key: "op", Type: "add"}
			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changed := ""
			runtimeNode.Next(ctx, func(output string) {
				changed = output
			})

			Expect(changed).To(Equal(""))
			res := s.Outputs[outputHandle]
			Expect(res.Data.Len()).To(Equal(int64(0)))
			Expect(res.Time.Len()).To(Equal(int64(0)))
		})

		It("Should not produce output when RHS is empty", func() {
			f := op.NewFactory()
			s := &state.State{Outputs: map[ir.Handle]state.Output{}}
			lhsSourceHandle := ir.Handle{Node: "lhsSource", Param: ir.DefaultOutputParam}
			rhsSourceHandle := ir.Handle{Node: "rhsSource", Param: ir.DefaultOutputParam}
			outputHandle := ir.Handle{Node: "op", Param: ir.DefaultOutputParam}

			lhsData := telem.NewSeriesV[float64](1.0, 2.0)
			lhsTime := telem.NewSeriesV[telem.TimeStamp](100, 200)
			rhsData := telem.NewSeriesV[float64]()
			rhsTime := telem.NewSeriesV[telem.TimeStamp]()

			s.Outputs[lhsSourceHandle] = state.Output{Data: lhsData, Time: lhsTime}
			s.Outputs[rhsSourceHandle] = state.Output{Data: rhsData, Time: rhsTime}
			s.Outputs[outputHandle] = state.Output{
				Data: telem.Series{DataType: telem.Float64T},
				Time: telem.Series{DataType: telem.TimeStampT},
			}

			inter := ir.IR{
				Edges: []ir.Edge{
					{
						Source: lhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.LHSInputParam},
					},
					{
						Source: rhsSourceHandle,
						Target: ir.Handle{Node: "op", Param: ir.RHSInputParam},
					},
				},
			}

			irNode := ir.Node{Key: "op", Type: "mul"}
			runtimeNode := MustSucceed(f.Create(ctx, node.Config{
				State:  s,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			changed := ""
			runtimeNode.Next(ctx, func(output string) {
				changed = output
			})

			Expect(changed).To(Equal(""))
			res := s.Outputs[outputHandle]
			Expect(res.Data.Len()).To(Equal(int64(0)))
			Expect(res.Time.Len()).To(Equal(int64(0)))
		})
	})
})

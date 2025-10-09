package selector_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	std "github.com/synnaxlabs/arc/runtime/select"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Select", func() {
	ctx := context.Background()
	DescribeTable("Select Node", func(
		input telem.Series,
		expectedTrue, expectedFalse telem.Series,
		expectedOutputs []string,
	) {
		f := std.NewFactory()
		s := &state.State{Outputs: map[ir.Handle]telem.Series{}}

		inputSourceHandle := ir.Handle{Node: "inputSource", Param: ir.DefaultOutputParam}
		trueOutputHandle := ir.Handle{Node: "select", Param: "true"}
		falseOutputHandle := ir.Handle{Node: "select", Param: "false"}

		s.Outputs[inputSourceHandle] = input
		s.Outputs[trueOutputHandle] = expectedTrue
		s.Outputs[falseOutputHandle] = expectedFalse

		inter := ir.IR{
			Edges: []ir.Edge{
				{
					Source: inputSourceHandle,
					Target: ir.Handle{Node: "select", Param: ir.DefaultInputParam},
				},
			},
		}

		irNode := ir.Node{Key: "select", Type: "select"}
		runtimeNode := MustSucceed(f.Create(node.Config{
			State:  s,
			Node:   irNode,
			Module: module.Module{IR: inter},
		}))

		changedOutputs := []string{}
		runtimeNode.Next(ctx, func(output string) {
			changedOutputs = append(changedOutputs, output)
		})

		Expect(changedOutputs).To(ConsistOf(expectedOutputs))

		trueResult := s.Outputs[trueOutputHandle]
		Expect(trueResult).To(telem.MatchSeries(expectedTrue))

		falseResult := s.Outputs[falseOutputHandle]
		Expect(falseResult).To(telem.MatchSeries(expectedFalse))
	},
		Entry("all true",
			telem.NewSeriesV[uint8](1, 1, 1, 1),
			telem.NewSeriesV[uint8](1, 1, 1, 1),
			telem.NewSeriesV[uint8](),
			[]string{"true"},
		),
		Entry("all false",
			telem.NewSeriesV[uint8](0, 0, 0, 0),
			telem.NewSeriesV[uint8](),
			telem.NewSeriesV[uint8](0, 0, 0, 0),
			[]string{"false"},
		),
		Entry("mixed - more true",
			telem.NewSeriesV[uint8](1, 1, 0, 1),
			telem.NewSeriesV[uint8](1, 1, 1),
			telem.NewSeriesV[uint8](0),
			[]string{"true", "false"},
		),
		Entry("mixed - more false",
			telem.NewSeriesV[uint8](0, 1, 0, 0),
			telem.NewSeriesV[uint8](1),
			telem.NewSeriesV[uint8](0, 0, 0),
			[]string{"true", "false"},
		),
		Entry("mixed - equal split",
			telem.NewSeriesV[uint8](1, 0, 1, 0),
			telem.NewSeriesV[uint8](1, 1),
			telem.NewSeriesV[uint8](0, 0),
			[]string{"true", "false"},
		),
		Entry("single true",
			telem.NewSeriesV[uint8](1),
			telem.NewSeriesV[uint8](1),
			telem.NewSeriesV[uint8](),
			[]string{"true"},
		),
		Entry("single false",
			telem.NewSeriesV[uint8](0),
			telem.NewSeriesV[uint8](),
			telem.NewSeriesV[uint8](0),
			[]string{"false"},
		),
		Entry("empty input",
			telem.NewSeriesV[uint8](),
			telem.NewSeriesV[uint8](),
			telem.NewSeriesV[uint8](),
			[]string{},
		),
		Entry("alternating pattern",
			telem.NewSeriesV[uint8](1, 0, 1, 0, 1, 0),
			telem.NewSeriesV[uint8](1, 1, 1),
			telem.NewSeriesV[uint8](0, 0, 0),
			[]string{"true", "false"},
		),
		Entry("consecutive true then false",
			telem.NewSeriesV[uint8](1, 1, 1, 0, 0, 0),
			telem.NewSeriesV[uint8](1, 1, 1),
			telem.NewSeriesV[uint8](0, 0, 0),
			[]string{"true", "false"},
		),
		Entry("consecutive false then true",
			telem.NewSeriesV[uint8](0, 0, 0, 1, 1, 1),
			telem.NewSeriesV[uint8](1, 1, 1),
			telem.NewSeriesV[uint8](0, 0, 0),
			[]string{"true", "false"},
		),
		Entry("mostly true with one false",
			telem.NewSeriesV[uint8](1, 1, 1, 1, 1, 0),
			telem.NewSeriesV[uint8](1, 1, 1, 1, 1),
			telem.NewSeriesV[uint8](0),
			[]string{"true", "false"},
		),
		Entry("mostly false with one true",
			telem.NewSeriesV[uint8](0, 0, 0, 0, 0, 1),
			telem.NewSeriesV[uint8](1),
			telem.NewSeriesV[uint8](0, 0, 0, 0, 0),
			[]string{"true", "false"},
		),
	)
})

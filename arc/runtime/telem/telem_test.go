// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	runtelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Telem", func() {
	ctx := context.Background()

	Describe("State", func() {
		It("should initialize with empty maps", func() {
			s := runtelem.NewState()
			Expect(s.Data).NotTo(BeNil())
			Expect(s.Writes).NotTo(BeNil())
			Expect(s.Readers).NotTo(BeNil())
			Expect(s.Data).To(HaveLen(0))
			Expect(s.Writes).To(HaveLen(0))
			Expect(s.Readers).To(HaveLen(0))
		})

		It("should ingest frames and mark dependencies dirty", func() {
			s := runtelem.NewState()
			dirtyNodes := []string{}

			// Register a dependency
			s.Readers[1] = []string{"node1", "node2"}

			// Create a frame with data
			fr := telem.MultiFrame[uint32](
				[]uint32{1},
				[]telem.Series{telem.NewSeriesV[float64](1.0, 2.0, 3.0)},
			)

			// Ingest the frame
			s.Ingest(fr, func(nodeKey string) {
				dirtyNodes = append(dirtyNodes, nodeKey)
			})

			// Should mark both dependencies as dirty
			Expect(dirtyNodes).To(ConsistOf("node1", "node2"))

			// Data should be stored
			Expect(s.Data[1].Len()).To(Equal(int64(3)))
		})

		It("should append data to existing series", func() {
			s := runtelem.NewState()

			// First ingestion
			fr1 := telem.MultiFrame[uint32](
				[]uint32{1},
				[]telem.Series{telem.NewSeriesV[float64](1.0, 2.0)},
			)
			s.Ingest(fr1, func(nodeKey string) {})

			// Second ingestion - should append
			fr2 := telem.MultiFrame[uint32](
				[]uint32{1},
				[]telem.Series{telem.NewSeriesV[float64](3.0, 4.0)},
			)
			s.Ingest(fr2, func(nodeKey string) {})

			// Total length should be 4
			Expect(s.Data[1].Len()).To(Equal(int64(4)))
		})
	})

	Describe("Source Node", func() {
		It("should read from telem state and update outputs", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(1)
			outputHandle := ir.Handle{Node: "source", Param: ir.DefaultOutputParam}
			runtimeState.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Float64T}}

			irNode := ir.Node{
				Key:      "source",
				Type:     "on",
				Channels: ir.Channels{Read: set.FromSlice([]uint32{channelKey})},
			}

			f := runtelem.NewTelemFactory(telemState)
			sourceNode := MustSucceed(f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			}))

			// Initialize the node
			sourceNode.Init(ctx, func(output string) {})

			// Add data to telem state with timestamp
			series := telem.NewSeriesV[float64](10.0, 20.0, 30.0)
			series.TimeRange = telem.TimeRange{
				Start: 0,
				End:   telem.TimeStamp(3),
			}
			telemState.Data[channelKey] = runtelem.Data{
				MultiSeries: telem.MultiSeries{Series: []telem.Series{series}},
				IndexKey:    channelKey,
			}

			// Call Next - should update output
			changedOutputs := []string{}
			sourceNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})

			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
			Expect(runtimeState.Outputs[outputHandle].Data).To(telem.MatchSeries(series))
		})

		It("should track high water mark and only emit new data", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(1)
			outputHandle := ir.Handle{Node: "source", Param: ir.DefaultOutputParam}
			runtimeState.Outputs[outputHandle] = state.Output{Data: telem.Series{DataType: telem.Float64T}}

			irNode := ir.Node{
				Key:      "source",
				Type:     "on",
				Channels: ir.Channels{Read: set.FromSlice([]uint32{channelKey})},
			}

			f := runtelem.NewTelemFactory(telemState)
			sourceNode := MustSucceed(f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			}))

			sourceNode.Init(ctx, func(output string) {})

			// First data
			series1 := telem.NewSeriesV[float64](10.0)
			series1.Alignment = 0
			telemState.Data[channelKey] = runtelem.Data{
				MultiSeries: telem.MultiSeries{Series: []telem.Series{series1}},
				IndexKey:    channelKey,
			}

			sourceNode.Next(ctx, func(output string) {})

			// Same data - should not emit
			changedOutputs := []string{}
			sourceNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(BeEmpty())

			// New data with higher timestamp
			series2 := telem.NewSeriesV[float64](20.0)
			series2.Alignment = 1
			telemState.Data[channelKey] = runtelem.Data{
				MultiSeries: telem.MultiSeries{Series: []telem.Series{series2}},
				IndexKey:    channelKey,
			}

			changedOutputs = []string{}
			sourceNode.Next(ctx, func(output string) {
				changedOutputs = append(changedOutputs, output)
			})
			Expect(changedOutputs).To(ConsistOf(ir.DefaultOutputParam))
		})
	})

	Describe("Sink Node", func() {
		It("should write from inputs to telem state", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(2)
			sourceHandle := ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam}

			// Set up input data
			inputData := telem.NewSeriesV[float64](100.0, 200.0, 300.0)
			runtimeState.Outputs[sourceHandle] = state.Output{Data: inputData}

			irNode := ir.Node{
				Key:      "sink",
				Type:     "write",
				Channels: ir.Channels{Write: set.FromSlice([]uint32{channelKey})},
			}

			// Create edge from upstream to sink
			inputEdge := ir.Edge{
				Source: sourceHandle,
				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			}

			inter := ir.IR{Edges: []ir.Edge{inputEdge}}

			f := runtelem.NewTelemFactory(telemState)
			sinkNode := MustSucceed(f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			// Initialize the node
			sinkNode.Init(ctx, func(output string) {})

			// Call Next - should write data
			sinkNode.Next(ctx, func(output string) {})

			// Check that data was written
			Expect(telemState.Writes[channelKey]).To(telem.MatchSeries(inputData))
		})

		It("should update writes on each Next call", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(3)
			sourceHandle := ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam}

			irNode := ir.Node{
				Key:      "sink",
				Type:     "write",
				Channels: ir.Channels{Write: set.FromSlice([]uint32{channelKey})},
			}

			inputEdge := ir.Edge{
				Source: sourceHandle,
				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			}

			inter := ir.IR{Edges: []ir.Edge{inputEdge}}

			f := runtelem.NewTelemFactory(telemState)
			sinkNode := MustSucceed(f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: inter},
			}))

			sinkNode.Init(ctx, func(output string) {})

			// First write
			runtimeState.Outputs[sourceHandle] = state.Output{Data: telem.NewSeriesV[float64](1.0, 2.0)}
			sinkNode.Next(ctx, func(output string) {})
			Expect(telemState.Writes[channelKey]).To(telem.MatchSeries(telem.NewSeriesV[float64](1.0, 2.0)))

			// Second write with different data
			runtimeState.Outputs[sourceHandle] = state.Output{Data: telem.NewSeriesV[float64](3.0, 4.0, 5.0)}
			sinkNode.Next(ctx, func(output string) {})
			Expect(telemState.Writes[channelKey]).To(telem.MatchSeries(telem.NewSeriesV[float64](3.0, 4.0, 5.0)))
		})
	})

	Describe("Telem Factory", func() {
		It("should return NotFound for wrong node type", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			irNode := ir.Node{
				Key:  "wrong",
				Type: "unknown",
			}

			f := runtelem.NewTelemFactory(telemState)
			_, err := f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			})

			Expect(err).To(HaveOccurred())
		})

		It("should create source node for 'on' type", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			irNode := ir.Node{
				Key:      "source",
				Type:     "on",
				Channels: ir.Channels{Read: set.FromSlice([]uint32{1})},
			}

			f := runtelem.NewTelemFactory(telemState)
			n, err := f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(n).NotTo(BeNil())
		})

		It("should create sink node for 'write' type", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			irNode := ir.Node{
				Key:      "sink",
				Type:     "write",
				Channels: ir.Channels{Write: set.FromSlice([]uint32{2})},
			}

			inputEdge := ir.Edge{
				Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			}

			f := runtelem.NewTelemFactory(telemState)
			n, err := f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{Edges: []ir.Edge{inputEdge}}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(n).NotTo(BeNil())
		})

		It("should register source channel dependencies", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(1)
			irNode := ir.Node{
				Key:      "source",
				Type:     "on",
				Channels: ir.Channels{Read: set.FromSlice([]uint32{channelKey})},
			}

			f := runtelem.NewTelemFactory(telemState)
			_, err := f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(telemState.Readers[channelKey]).To(ContainElement("source"))
		})

		It("should register sink channel writers", func() {
			telemState := runtelem.NewState()
			runtimeState := &state.State{Outputs: map[ir.Handle]state.Output{}}

			channelKey := uint32(2)
			irNode := ir.Node{
				Key:      "sink",
				Type:     "write",
				Channels: ir.Channels{Write: set.FromSlice([]uint32{channelKey})},
			}

			inputEdge := ir.Edge{
				Source: ir.Handle{Node: "upstream", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
			}

			f := runtelem.NewTelemFactory(telemState)
			_, err := f.Create(ctx, node.Config{
				State:  runtimeState,
				Node:   irNode,
				Module: module.Module{IR: ir.IR{Edges: []ir.Edge{inputEdge}}},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(telemState.Writers[channelKey]).To(ContainElement("sink"))
		})
	})
})

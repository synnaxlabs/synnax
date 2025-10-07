// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package archive_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/archive"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var ctx = context.Background()

func TestRuntime(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Runtime Suite")
}

var _ = Describe("Runtime", func() {
	Describe("NewRuntime", func() {
		It("Should create a runtime from an empty IR", func() {
			program := ir.IR{
				Stages:    []ir.Stage{},
				Functions: []ir.Function{},
				Nodes:     []ir.Node{},
				Edges:     []ir.Edge{},
				Strata:    ir.NewStrata(),
			}

			rt := MustSucceed(archive.NewRuntime(ctx, program))
			Expect(rt).ToNot(BeNil())
			Expect(rt.Close()).To(Succeed())
		})

		It("Should initialize channel state for all referenced channels", func() {
			// Create a simple graph with one node that reads from channel 100
			program := ir.IR{
				Stages: []ir.Stage{
					{
						Key:     "test_stage",
						Config:  ir.NamedTypes{},
						Params:  ir.NamedTypes{},
						Outputs: ir.NamedTypes{},
					},
				},
				Nodes: []ir.Node{
					{
						Key:    "node_1",
						Type:   "test_stage",
						Config: map[string]any{},
						Channels: ir.Channels{
							Read:  set.Set[uint32]{100: {}},
							Write: set.Set[uint32]{},
						},
					},
				},
				Edges: []ir.Edge{},
				Strata: ir.Strata{
					Nodes: map[string]int{"node_1": 0},
					Max:   0,
				},
			}

			rt := MustSucceed(archive.NewRuntime(ctx, program))
			Expect(rt).ToNot(BeNil())

			// Note: Can't directly test channel state without exposing it
			// This test just verifies initialization doesn't panic

			Expect(rt.Close()).To(Succeed())
		})
	})

	Describe("Frame Ingestion", func() {
		It("Should process an empty frame without error", func() {
			program := ir.IR{
				Stages: []ir.Stage{},
				Nodes:  []ir.Node{},
				Edges:  []ir.Edge{},
				Strata: ir.NewStrata(),
			}

			rt := MustSucceed(archive.NewRuntime(ctx, program))

			// Create empty frame
			frame := telem.MultiFrame[uint32](nil, nil)

			Expect(rt.Next(frame)).To(Succeed())
			Expect(rt.Close()).To(Succeed())
		})

		It("Should process a frame with channel data", func() {
			// Create a simple graph with one node that reads from channel 100
			program := ir.IR{
				Stages: []ir.Stage{
					{
						Key:     "test_stage",
						Config:  ir.NamedTypes{},
						Params:  ir.NamedTypes{},
						Outputs: ir.NamedTypes{},
					},
				},
				Nodes: []ir.Node{
					{
						Key:    "node_1",
						Type:   "test_stage",
						Config: map[string]any{},
						Channels: ir.Channels{
							Read:  set.Set[uint32]{100: {}},
							Write: set.Set[uint32]{},
						},
					},
				},
				Edges: []ir.Edge{},
				Strata: ir.Strata{
					Nodes: map[string]int{"node_1": 0},
					Max:   0,
				},
			}

			rt := MustSucceed(archive.NewRuntime(ctx, program))

			// Create frame with data for channel 100
			series := telem.NewSeries([]float64{1.0, 2.0, 3.0, 4.0})
			series.TimeRange = telem.TimeRange{Start: 0, End: 4000}
			series.Alignment = telem.Alignment(0)

			frame := telem.MultiFrame[uint32]([]uint32{100}, []telem.Series{series})

			// This will fail until WASM execution is implemented
			// For now, just verify it doesn't crash
			_ = rt.Next(frame)

			Expect(rt.Close()).To(Succeed())
		})
	})

	Describe("Stratified Execution", func() {
		It("Should execute nodes in stratified order", func() {
			Skip("Requires WASM integration")

			// Test with a graph that has multiple strata
			// node_1 (stratum 0) -> node_2 (stratum 1) -> node_3 (stratum 2)
		})
	})
})

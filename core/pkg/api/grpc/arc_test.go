// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"testing"

	arcir "github.com/synnaxlabs/arc/ir"
)

// TestTranslateEdgeToPB_PreservesKind is a regression test ensuring that edge Kind is
// preserved during protobuf serialization. This bug was discovered when one-shot edges
// (=>) were being serialized as continuous edges because the Kind field was not being set.
func TestTranslateEdgeToPB_PreservesKind(t *testing.T) {
	tests := []struct {
		name string
		kind arcir.EdgeKind
	}{
		{"Continuous edge", arcir.Continuous},
		{"OneShot edge", arcir.OneShot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := arcir.Edge{
				Source: arcir.Handle{Node: "node1", Param: "output"},
				Target: arcir.Handle{Node: "node2", Param: "input"},
				Kind:   tt.kind,
			}

			pb := translateEdgeToPB(edge)

			if arcir.PBEdgeKind(pb.Kind) != arcir.PBEdgeKind(tt.kind) {
				t.Errorf("translateEdgeToPB: expected Kind=%v, got %v", tt.kind, pb.Kind)
			}
		})
	}
}

// TestTranslateEdgeFromPB_PreservesKind is a regression test ensuring that edge Kind is
// preserved when deserializing from protobuf.
func TestTranslateEdgeFromPB_PreservesKind(t *testing.T) {
	tests := []struct {
		name string
		kind arcir.PBEdgeKind
	}{
		{"Continuous edge", arcir.PBEdgeKind_CONTINUOUS},
		{"OneShot edge", arcir.PBEdgeKind_ONE_SHOT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &arcir.PBEdge{
				Source: &arcir.PBHandle{Node: "node1", Param: "output"},
				Target: &arcir.PBHandle{Node: "node2", Param: "input"},
				Kind:   tt.kind,
			}

			edge := translateEdgeFromPB(pb)

			if arcir.PBEdgeKind(edge.Kind) != tt.kind {
				t.Errorf("translateEdgeFromPB: expected Kind=%v, got %v", tt.kind, edge.Kind)
			}
		})
	}
}

// TestTranslateEdgeRoundTrip tests that an edge can be serialized to protobuf and back
// while preserving all fields including Kind.
func TestTranslateEdgeRoundTrip(t *testing.T) {
	original := arcir.Edge{
		Source: arcir.Handle{Node: "sensor", Param: "output"},
		Target: arcir.Handle{Node: "stage_entry", Param: "activate"},
		Kind:   arcir.OneShot,
	}

	pb := translateEdgeToPB(original)
	result := translateEdgeFromPB(pb)

	if result.Source != original.Source {
		t.Errorf("Source mismatch: expected %v, got %v", original.Source, result.Source)
	}
	if result.Target != original.Target {
		t.Errorf("Target mismatch: expected %v, got %v", original.Target, result.Target)
	}
	if result.Kind != original.Kind {
		t.Errorf("Kind mismatch: expected %v, got %v", original.Kind, result.Kind)
	}
}

// TestTranslateIRToPB_IncludesSequences is a regression test ensuring that sequences are
// included in the IR protobuf serialization.
func TestTranslateIRToPB_IncludesSequences(t *testing.T) {
	ir := arcir.IR{
		Sequences: arcir.Sequences{
			{
				Key: "main",
				Stages: []arcir.Stage{
					{Key: "ignite", Nodes: []string{"entry_main_ignite", "valve_0"}},
					{Key: "pressurize", Nodes: []string{"entry_main_pressurize", "pump_0"}},
				},
			},
			{
				Key: "abort",
				Stages: []arcir.Stage{
					{Key: "shutdown", Nodes: []string{"entry_abort_shutdown", "kill_0"}},
				},
			},
		},
	}

	pb, err := translateIRToPB(ir)
	if err != nil {
		t.Fatalf("translateIRToPB failed: %v", err)
	}

	if len(pb.Sequences) != 2 {
		t.Errorf("Expected 2 sequences, got %d", len(pb.Sequences))
	}

	// Verify first sequence
	if pb.Sequences[0].Key != "main" {
		t.Errorf("Expected sequence key 'main', got '%s'", pb.Sequences[0].Key)
	}
	if len(pb.Sequences[0].Stages) != 2 {
		t.Errorf("Expected 2 stages in 'main', got %d", len(pb.Sequences[0].Stages))
	}

	// Verify second sequence
	if pb.Sequences[1].Key != "abort" {
		t.Errorf("Expected sequence key 'abort', got '%s'", pb.Sequences[1].Key)
	}
}

// TestTranslateSequenceRoundTrip tests that sequences can be serialized and deserialized.
func TestTranslateSequenceRoundTrip(t *testing.T) {
	original := arcir.Sequence{
		Key: "test_sequence",
		Stages: []arcir.Stage{
			{Key: "stage1", Nodes: []string{"node1", "node2"}},
			{Key: "stage2", Nodes: []string{"node3"}},
		},
	}

	pb := translateSequenceToPB(original)
	result := translateSequenceFromPB(pb)

	if result.Key != original.Key {
		t.Errorf("Key mismatch: expected %s, got %s", original.Key, result.Key)
	}
	if len(result.Stages) != len(original.Stages) {
		t.Errorf("Stages length mismatch: expected %d, got %d",
			len(original.Stages), len(result.Stages))
	}
	for i := range original.Stages {
		if result.Stages[i].Key != original.Stages[i].Key {
			t.Errorf("Stage[%d] key mismatch: expected %s, got %s",
				i, original.Stages[i].Key, result.Stages[i].Key)
		}
		if len(result.Stages[i].Nodes) != len(original.Stages[i].Nodes) {
			t.Errorf("Stage[%d] nodes length mismatch", i)
		}
	}
}

// TestTranslateIRRoundTrip_WithOneShotEdges tests full IR round-trip with one-shot edges.
func TestTranslateIRRoundTrip_WithOneShotEdges(t *testing.T) {
	original := arcir.IR{
		Edges: arcir.Edges{
			{
				Source: arcir.Handle{Node: "timer", Param: "output"},
				Target: arcir.Handle{Node: "controller", Param: "input"},
				Kind:   arcir.Continuous,
			},
			{
				Source: arcir.Handle{Node: "sensor", Param: "output"},
				Target: arcir.Handle{Node: "stage_entry", Param: "activate"},
				Kind:   arcir.OneShot,
			},
		},
		Sequences: arcir.Sequences{
			{
				Key: "main",
				Stages: []arcir.Stage{
					{Key: "run", Nodes: []string{"stage_entry", "valve_0"}},
				},
			},
		},
	}

	pb, err := translateIRToPB(original)
	if err != nil {
		t.Fatalf("translateIRToPB failed: %v", err)
	}

	result, err := translateIRFromPB(pb)
	if err != nil {
		t.Fatalf("translateIRFromPB failed: %v", err)
	}

	// Verify edges
	if len(result.Edges) != 2 {
		t.Fatalf("Expected 2 edges, got %d", len(result.Edges))
	}
	if result.Edges[0].Kind != arcir.Continuous {
		t.Errorf("Edge[0] expected Continuous, got %v", result.Edges[0].Kind)
	}
	if result.Edges[1].Kind != arcir.OneShot {
		t.Errorf("Edge[1] expected OneShot, got %v", result.Edges[1].Kind)
	}

	// Verify sequences
	if len(result.Sequences) != 1 {
		t.Fatalf("Expected 1 sequence, got %d", len(result.Sequences))
	}
	if result.Sequences[0].Key != "main" {
		t.Errorf("Expected sequence key 'main', got '%s'", result.Sequences[0].Key)
	}
}

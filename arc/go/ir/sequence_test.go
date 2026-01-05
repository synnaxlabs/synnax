// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stage", func() {
	Describe("String", func() {
		It("Should format stage with nodes", func() {
			stage := ir.Stage{
				Key:   "pressurization",
				Nodes: []string{"timer_1", "controller_1"},
			}
			Expect(stage.String()).To(Equal("pressurization: [timer_1, controller_1]"))
		})

		It("Should format stage with empty nodes", func() {
			stage := ir.Stage{Key: "terminal", Nodes: nil}
			Expect(stage.String()).To(Equal("terminal: []"))
		})

		It("Should format stage with strata", func() {
			stage := ir.Stage{
				Key:    "main",
				Nodes:  []string{"a", "b"},
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			str := stage.String()
			Expect(str).To(ContainSubstring("main: [a, b]"))
			Expect(str).To(ContainSubstring("[0]: a"))
			Expect(str).To(ContainSubstring("[1]: b"))
		})
	})

	Describe("JSON Serialization", func() {
		It("Should marshal and unmarshal stage with nodes", func() {
			stage := ir.Stage{
				Key:   "pressurization",
				Nodes: []string{"timer_1", "controller_1", "condition_1"},
			}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("pressurization"))
			Expect(restored.Nodes).To(Equal([]string{"timer_1", "controller_1", "condition_1"}))
		})

		It("Should handle empty nodes list", func() {
			stage := ir.Stage{Key: "terminal"}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("terminal"))
			Expect(restored.Nodes).To(BeNil())
		})

		It("Should marshal nodes as null when nil, not empty array", func() {
			stage := ir.Stage{Key: "empty"}
			data := MustSucceed(json.Marshal(stage))
			// Should be {"key":"empty","nodes":null} not {"key":"empty","nodes":[]}
			Expect(string(data)).To(ContainSubstring(`"nodes":null`))
		})

		It("Should preserve order of nodes", func() {
			stage := ir.Stage{
				Key:   "ordered",
				Nodes: []string{"first", "second", "third", "fourth"},
			}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Nodes[0]).To(Equal("first"))
			Expect(restored.Nodes[1]).To(Equal("second"))
			Expect(restored.Nodes[2]).To(Equal("third"))
			Expect(restored.Nodes[3]).To(Equal("fourth"))
		})
	})
})

var _ = Describe("Sequence", func() {
	var seq ir.Sequence

	BeforeEach(func() {
		seq = ir.Sequence{
			Key: "main",
			Stages: []ir.Stage{
				{Key: "precheck", Nodes: []string{"check_1", "check_2"}},
				{Key: "pressurization", Nodes: []string{"timer_1", "ctrl_1"}},
				{Key: "ignition", Nodes: []string{"igniter_1"}},
				{Key: "complete", Nodes: nil},
			},
		}
	})

	Describe("String", func() {
		It("Should format sequence with tree structure", func() {
			str := seq.String()
			Expect(str).To(HavePrefix("main\n"))
			Expect(str).To(ContainSubstring("├── precheck:"))
			Expect(str).To(ContainSubstring("├── pressurization:"))
			Expect(str).To(ContainSubstring("├── ignition:"))
			Expect(str).To(ContainSubstring("└── complete:"))
		})

		It("Should format single-stage sequence", func() {
			single := ir.Sequence{
				Key:    "single",
				Stages: []ir.Stage{{Key: "only", Nodes: []string{"node_1"}}},
			}
			str := single.String()
			Expect(str).To(HavePrefix("single\n"))
			Expect(str).To(ContainSubstring("└── only: [node_1]"))
		})
	})

	Describe("Entry", func() {
		It("Should return first stage as entry point", func() {
			entry := seq.Entry()
			Expect(entry.Key).To(Equal("precheck"))
			Expect(entry.Nodes).To(Equal([]string{"check_1", "check_2"}))
		})

		It("Should panic on empty sequence", func() {
			empty := ir.Sequence{Key: "empty", Stages: nil}
			Expect(func() { empty.Entry() }).To(Panic())
		})

		It("Should panic on sequence with empty stages slice", func() {
			empty := ir.Sequence{Key: "empty", Stages: []ir.Stage{}}
			Expect(func() { empty.Entry() }).To(Panic())
		})

		It("Should work with single-stage sequence", func() {
			single := ir.Sequence{
				Key:    "single",
				Stages: []ir.Stage{{Key: "only", Nodes: []string{"node_1"}}},
			}
			entry := single.Entry()
			Expect(entry.Key).To(Equal("only"))
		})
	})

	Describe("NextStage", func() {
		It("Should return next stage in order", func() {
			next, ok := seq.NextStage("precheck")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("pressurization"))
		})

		It("Should return next for middle stage", func() {
			next, ok := seq.NextStage("pressurization")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("ignition"))
		})

		It("Should chain through all stages", func() {
			current := "precheck"
			expected := []string{"pressurization", "ignition", "complete"}
			for i := 0; i < 3; i++ {
				next, ok := seq.NextStage(current)
				Expect(ok).To(BeTrue())
				Expect(next.Key).To(Equal(expected[i]))
				current = next.Key
			}
		})

		It("Should return false for last stage", func() {
			_, ok := seq.NextStage("complete")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for unknown stage", func() {
			_, ok := seq.NextStage("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for empty string key", func() {
			_, ok := seq.NextStage("")
			Expect(ok).To(BeFalse())
		})

		It("Should work correctly with single-stage sequence", func() {
			single := ir.Sequence{
				Key:    "single",
				Stages: []ir.Stage{{Key: "only"}},
			}
			_, ok := single.NextStage("only")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("FindStage", func() {
		It("Should find existing stage", func() {
			stage, ok := seq.FindStage("ignition")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("ignition"))
			Expect(stage.Nodes).To(Equal([]string{"igniter_1"}))
		})

		It("Should find first stage", func() {
			stage, ok := seq.FindStage("precheck")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("precheck"))
		})

		It("Should find last stage", func() {
			stage, ok := seq.FindStage("complete")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("complete"))
		})

		It("Should return false for nonexistent stage", func() {
			_, ok := seq.FindStage("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for empty key", func() {
			_, ok := seq.FindStage("")
			Expect(ok).To(BeFalse())
		})

		It("Should handle empty sequence", func() {
			empty := ir.Sequence{Key: "empty"}
			_, ok := empty.FindStage("any")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("JSON Serialization", func() {
		It("Should preserve stage ordering", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Stages).To(HaveLen(4))
			Expect(restored.Stages[0].Key).To(Equal("precheck"))
			Expect(restored.Stages[1].Key).To(Equal("pressurization"))
			Expect(restored.Stages[2].Key).To(Equal("ignition"))
			Expect(restored.Stages[3].Key).To(Equal("complete"))
		})

		It("Should preserve nodes within stages", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Stages[0].Nodes).To(Equal([]string{"check_1", "check_2"}))
			Expect(restored.Stages[1].Nodes).To(Equal([]string{"timer_1", "ctrl_1"}))
			Expect(restored.Stages[2].Nodes).To(Equal([]string{"igniter_1"}))
			Expect(restored.Stages[3].Nodes).To(BeNil())
		})

		It("Should handle empty stages slice", func() {
			empty := ir.Sequence{Key: "empty", Stages: []ir.Stage{}}
			data := MustSucceed(json.Marshal(empty))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("empty"))
			Expect(restored.Stages).To(HaveLen(0))
		})

		It("Should round-trip Entry() result correctly", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			entry := restored.Entry()
			Expect(entry.Key).To(Equal("precheck"))
		})

		It("Should round-trip NextStage() correctly", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			next, ok := restored.NextStage("precheck")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("pressurization"))
		})
	})
})

var _ = Describe("Sequences", func() {
	var sequences ir.Sequences

	BeforeEach(func() {
		sequences = ir.Sequences{
			{
				Key: "main",
				Stages: []ir.Stage{
					{Key: "start", Nodes: []string{"node_1"}},
					{Key: "end", Nodes: []string{"node_2"}},
				},
			},
			{
				Key: "abort",
				Stages: []ir.Stage{
					{Key: "safing", Nodes: []string{"abort_1", "abort_2"}},
					{Key: "safed", Nodes: nil},
				},
			},
			{
				Key: "recovery",
				Stages: []ir.Stage{
					{Key: "assess", Nodes: []string{"assess_1"}},
					{Key: "restart", Nodes: []string{"restart_1"}},
					{Key: "complete", Nodes: nil},
				},
			},
		}
	})

	Describe("Find", func() {
		It("Should find existing sequence", func() {
			seq, ok := sequences.Find("main")
			Expect(ok).To(BeTrue())
			Expect(seq.Key).To(Equal("main"))
			Expect(seq.Stages).To(HaveLen(2))
		})

		It("Should find sequence by key", func() {
			seq, ok := sequences.Find("abort")
			Expect(ok).To(BeTrue())
			Expect(seq.Key).To(Equal("abort"))
		})

		It("Should return false for missing sequence", func() {
			_, ok := sequences.Find("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for empty key", func() {
			_, ok := sequences.Find("")
			Expect(ok).To(BeFalse())
		})

		It("Should handle empty collection", func() {
			empty := ir.Sequences{}
			_, ok := empty.Find("main")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("Should get existing sequence", func() {
			seq := sequences.Get("main")
			Expect(seq.Key).To(Equal("main"))
		})

		It("Should panic for missing sequence", func() {
			Expect(func() {
				_ = sequences.Get("nonexistent")
			}).To(Panic())
		})
	})

	Describe("FindStage", func() {
		It("Should find stage and its parent sequence", func() {
			stage, seq, ok := sequences.FindStage("safing")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("safing"))
			Expect(stage.Nodes).To(Equal([]string{"abort_1", "abort_2"}))
			Expect(seq.Key).To(Equal("abort"))
		})

		It("Should find stage in first sequence", func() {
			stage, seq, ok := sequences.FindStage("start")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("start"))
			Expect(seq.Key).To(Equal("main"))
		})

		It("Should find stage in last sequence", func() {
			stage, seq, ok := sequences.FindStage("complete")
			Expect(ok).To(BeTrue())
			Expect(stage.Key).To(Equal("complete"))
			Expect(seq.Key).To(Equal("recovery"))
		})

		It("Should return false for missing stage", func() {
			_, _, ok := sequences.FindStage("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return first match when stage names collide", func() {
			// Stage names only need to be unique within sequence
			seqs := ir.Sequences{
				{Key: "seq1", Stages: []ir.Stage{{Key: "init", Nodes: []string{"s1_init"}}}},
				{Key: "seq2", Stages: []ir.Stage{{Key: "init", Nodes: []string{"s2_init"}}}},
			}
			stage, seq, ok := seqs.FindStage("init")
			Expect(ok).To(BeTrue())
			Expect(seq.Key).To(Equal("seq1")) // First match
			Expect(stage.Key).To(Equal("init"))
			Expect(stage.Nodes).To(Equal([]string{"s1_init"}))
		})

		It("Should handle empty collection", func() {
			empty := ir.Sequences{}
			_, _, ok := empty.FindStage("any")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("JSON Serialization", func() {
		It("Should marshal and unmarshal sequences", func() {
			data := MustSucceed(json.Marshal(sequences))

			var restored ir.Sequences
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored).To(HaveLen(3))
			Expect(restored[0].Key).To(Equal("main"))
			Expect(restored[1].Key).To(Equal("abort"))
			Expect(restored[2].Key).To(Equal("recovery"))
		})

		It("Should preserve nested stages", func() {
			data := MustSucceed(json.Marshal(sequences))

			var restored ir.Sequences
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			// Check main sequence
			main, ok := restored.Find("main")
			Expect(ok).To(BeTrue())
			Expect(main.Stages).To(HaveLen(2))
			Expect(main.Entry().Key).To(Equal("start"))

			// Check abort sequence
			abort, ok := restored.Find("abort")
			Expect(ok).To(BeTrue())
			Expect(abort.Stages).To(HaveLen(2))
			Expect(abort.Entry().Key).To(Equal("safing"))

			// Check recovery sequence
			recovery, ok := restored.Find("recovery")
			Expect(ok).To(BeTrue())
			Expect(recovery.Stages).To(HaveLen(3))
			Expect(recovery.Entry().Key).To(Equal("assess"))
		})

		It("Should handle empty sequences", func() {
			empty := ir.Sequences{}
			data := MustSucceed(json.Marshal(empty))

			var restored ir.Sequences
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored).To(BeEmpty())
		})
	})

	Describe("Multiple Sequence Operations", func() {
		It("Should support independent Entry() calls", func() {
			main := sequences.Get("main")
			abort := sequences.Get("abort")
			recovery := sequences.Get("recovery")

			Expect(main.Entry().Key).To(Equal("start"))
			Expect(abort.Entry().Key).To(Equal("safing"))
			Expect(recovery.Entry().Key).To(Equal("assess"))
		})

		It("Should support independent NextStage() calls", func() {
			main := sequences.Get("main")
			abort := sequences.Get("abort")

			mainNext, _ := main.NextStage("start")
			abortNext, _ := abort.NextStage("safing")

			Expect(mainNext.Key).To(Equal("end"))
			Expect(abortNext.Key).To(Equal("safed"))
		})

		It("Should support FindStage across all sequences", func() {
			// Find stages from different sequences
			stage1, seq1, ok1 := sequences.FindStage("start")
			stage2, seq2, ok2 := sequences.FindStage("safing")
			stage3, seq3, ok3 := sequences.FindStage("restart")

			Expect(ok1).To(BeTrue())
			Expect(ok2).To(BeTrue())
			Expect(ok3).To(BeTrue())

			Expect(seq1.Key).To(Equal("main"))
			Expect(seq2.Key).To(Equal("abort"))
			Expect(seq3.Key).To(Equal("recovery"))

			Expect(stage1.Nodes).To(Equal([]string{"node_1"}))
			Expect(stage2.Nodes).To(Equal([]string{"abort_1", "abort_2"}))
			Expect(stage3.Nodes).To(Equal([]string{"restart_1"}))
		})
	})
})

var _ = Describe("Integration: Sequence with Edge Kinds", func() {
	It("Should represent a complete sequence state machine with edges", func() {
		// Build a realistic sequence with both Continuous and OneShot edges
		sequences := ir.Sequences{
			{
				Key: "main",
				Stages: []ir.Stage{
					{Key: "precheck", Nodes: []string{"timer_1", "check_1", "precheck_entry"}},
					{Key: "pressurize", Nodes: []string{"valve_ctrl", "pressure_monitor", "pressurize_entry"}},
					{Key: "complete", Nodes: []string{"complete_entry"}},
				},
			},
		}

		edges := ir.Edges{
			// Continuous dataflow within precheck stage
			{
				Source: ir.Handle{Node: "timer_1", Param: "output"},
				Target: ir.Handle{Node: "check_1", Param: "input"},
				Kind:   ir.Continuous,
			},
			// OneShot transition: precheck -> pressurize
			{
				Source: ir.Handle{Node: "check_1", Param: "output"},
				Target: ir.Handle{Node: "pressurize_entry", Param: "activate"},
				Kind:   ir.OneShot,
			},
			// Continuous dataflow within pressurize stage
			{
				Source: ir.Handle{Node: "valve_ctrl", Param: "output"},
				Target: ir.Handle{Node: "pressure_monitor", Param: "input"},
				Kind:   ir.Continuous,
			},
			// OneShot transition: pressurize -> complete
			{
				Source: ir.Handle{Node: "pressure_monitor", Param: "threshold_met"},
				Target: ir.Handle{Node: "complete_entry", Param: "activate"},
				Kind:   ir.OneShot,
			},
		}

		// Verify structure
		Expect(sequences).To(HaveLen(1))
		main := sequences.Get("main")
		Expect(main.Stages).To(HaveLen(3))

		// Verify entry point
		entry := main.Entry()
		Expect(entry.Key).To(Equal("precheck"))

		// Verify stage navigation
		next, ok := main.NextStage("precheck")
		Expect(ok).To(BeTrue())
		Expect(next.Key).To(Equal("pressurize"))

		// Verify edge kinds
		continuous := edges.GetByKind(ir.Continuous)
		oneShot := edges.GetByKind(ir.OneShot)

		Expect(continuous).To(HaveLen(2))
		Expect(oneShot).To(HaveLen(2))

		// Verify OneShot edges target entry nodes
		for _, e := range oneShot {
			Expect(e.Target.Node).To(ContainSubstring("_entry"))
			Expect(e.Target.Param).To(Equal("activate"))
		}
	})
})

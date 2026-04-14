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
				Key:    "pressurization",
				Strata: ir.Strata{{"timer_1", "controller_1"}},
			}
			Expect(stage.String()).To(HavePrefix("pressurization: [timer_1, controller_1]"))
		})

		It("Should format stage with empty nodes", func() {
			stage := ir.Stage{Key: "terminal"}
			Expect(stage.String()).To(Equal("terminal: []"))
		})

		It("Should format stage with strata", func() {
			stage := ir.Stage{
				Key:    "main",
				Strata: ir.Strata{{"a"}, {"b"}},
			}
			str := stage.String()
			Expect(str).To(ContainSubstring("main: [a, b]"))
			Expect(str).To(ContainSubstring("[0]: a"))
			Expect(str).To(ContainSubstring("[1]: b"))
		})
	})

	Describe("JSON Serialization", func() {
		It("Should marshal and unmarshal stage with strata", func() {
			stage := ir.Stage{
				Key:    "pressurization",
				Strata: ir.Strata{{"timer_1", "controller_1", "condition_1"}},
			}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("pressurization"))
			Expect(restored.Strata.Flatten()).To(Equal([]string{"timer_1", "controller_1", "condition_1"}))
		})

		It("Should handle empty strata", func() {
			stage := ir.Stage{Key: "terminal"}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("terminal"))
			Expect(restored.Strata).To(BeNil())
		})

		It("Should preserve stratum order", func() {
			stage := ir.Stage{
				Key:    "ordered",
				Strata: ir.Strata{{"first", "second"}, {"third", "fourth"}},
			}
			data := MustSucceed(json.Marshal(stage))

			var restored ir.Stage
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			flat := restored.Strata.Flatten()
			Expect(flat).To(Equal([]string{"first", "second", "third", "fourth"}))
		})
	})
})

func stageStep(key string, nodes []string) ir.Step {
	stage := &ir.Stage{Key: key}
	if nodes != nil {
		stage.Strata = ir.Strata{nodes}
	}
	return ir.Step{Key: key, Stage: stage}
}

var _ = Describe("Sequence", func() {
	var seq ir.Sequence

	BeforeEach(func() {
		seq = ir.Sequence{
			Key: "main",
			Steps: []ir.Step{
				stageStep("precheck", []string{"check_1", "check_2"}),
				stageStep("pressurization", []string{"timer_1", "ctrl_1"}),
				stageStep("ignition", []string{"igniter_1"}),
				stageStep("complete", nil),
			},
		}
	})

	Describe("String", func() {
		It("Should format sequence with tree structure", func() {
			str := seq.String()
			Expect(str).To(HavePrefix("main\n"))
			Expect(str).To(ContainSubstring("├── precheck"))
			Expect(str).To(ContainSubstring("├── pressurization"))
			Expect(str).To(ContainSubstring("├── ignition"))
			Expect(str).To(ContainSubstring("└── complete"))
		})

		It("Should format single-step sequence", func() {
			single := ir.Sequence{
				Key:   "single",
				Steps: []ir.Step{stageStep("only", []string{"node_1"})},
			}
			str := single.String()
			Expect(str).To(HavePrefix("single\n"))
			Expect(str).To(ContainSubstring("└── only"))
		})
	})

	Describe("Entry", func() {
		It("Should return first step as entry point", func() {
			entry := seq.Entry()
			Expect(entry.Key).To(Equal("precheck"))
			Expect(entry.StageNodes()).To(Equal([]string{"check_1", "check_2"}))
		})

		It("Should panic on empty sequence", func() {
			empty := ir.Sequence{Key: "empty", Steps: nil}
			Expect(func() { empty.Entry() }).To(Panic())
		})

		It("Should panic on sequence with empty steps slice", func() {
			empty := ir.Sequence{Key: "empty", Steps: []ir.Step{}}
			Expect(func() { empty.Entry() }).To(Panic())
		})

		It("Should work with single-step sequence", func() {
			single := ir.Sequence{
				Key:   "single",
				Steps: []ir.Step{stageStep("only", []string{"node_1"})},
			}
			entry := single.Entry()
			Expect(entry.Key).To(Equal("only"))
		})
	})

	Describe("NextStep", func() {
		It("Should return next step in order", func() {
			next, ok := seq.NextStep("precheck")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("pressurization"))
		})

		It("Should return next for middle step", func() {
			next, ok := seq.NextStep("pressurization")
			Expect(ok).To(BeTrue())
			Expect(next.Key).To(Equal("ignition"))
		})

		It("Should chain through all steps", func() {
			current := "precheck"
			expected := []string{"pressurization", "ignition", "complete"}
			for i := range 3 {
				next, ok := seq.NextStep(current)
				Expect(ok).To(BeTrue())
				Expect(next.Key).To(Equal(expected[i]))
				current = next.Key
			}
		})

		It("Should return false for last step", func() {
			_, ok := seq.NextStep("complete")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for unknown step", func() {
			_, ok := seq.NextStep("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for empty string key", func() {
			_, ok := seq.NextStep("")
			Expect(ok).To(BeFalse())
		})

		It("Should work correctly with single-step sequence", func() {
			single := ir.Sequence{
				Key:   "single",
				Steps: []ir.Step{stageStep("only", nil)},
			}
			_, ok := single.NextStep("only")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("FindStep", func() {
		It("Should find existing step", func() {
			step, ok := seq.FindStep("ignition")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("ignition"))
			Expect(step.StageNodes()).To(Equal([]string{"igniter_1"}))
		})

		It("Should find first step", func() {
			step, ok := seq.FindStep("precheck")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("precheck"))
		})

		It("Should find last step", func() {
			step, ok := seq.FindStep("complete")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("complete"))
		})

		It("Should return false for nonexistent step", func() {
			_, ok := seq.FindStep("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return false for empty key", func() {
			_, ok := seq.FindStep("")
			Expect(ok).To(BeFalse())
		})

		It("Should handle empty sequence", func() {
			empty := ir.Sequence{Key: "empty"}
			_, ok := empty.FindStep("any")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("JSON Serialization", func() {
		It("Should preserve step ordering", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Steps).To(HaveLen(4))
			Expect(restored.Steps[0].Key).To(Equal("precheck"))
			Expect(restored.Steps[1].Key).To(Equal("pressurization"))
			Expect(restored.Steps[2].Key).To(Equal("ignition"))
			Expect(restored.Steps[3].Key).To(Equal("complete"))
		})

		It("Should preserve nodes within stage steps", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			Expect(restored.Steps[0].StageNodes()).To(Equal([]string{"check_1", "check_2"}))
			Expect(restored.Steps[1].StageNodes()).To(Equal([]string{"timer_1", "ctrl_1"}))
			Expect(restored.Steps[2].StageNodes()).To(Equal([]string{"igniter_1"}))
			Expect(restored.Steps[3].StageNodes()).To(BeNil())
		})

		It("Should handle empty steps slice", func() {
			empty := ir.Sequence{Key: "empty", Steps: []ir.Step{}}
			data := MustSucceed(json.Marshal(empty))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())
			Expect(restored.Key).To(Equal("empty"))
			Expect(restored.Steps).To(HaveLen(0))
		})

		It("Should round-trip Entry() result correctly", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			entry := restored.Entry()
			Expect(entry.Key).To(Equal("precheck"))
		})

		It("Should round-trip NextStep() correctly", func() {
			data := MustSucceed(json.Marshal(seq))

			var restored ir.Sequence
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			next, ok := restored.NextStep("precheck")
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
				Steps: []ir.Step{
					stageStep("start", []string{"node_1"}),
					stageStep("end", []string{"node_2"}),
				},
			},
			{
				Key: "abort",
				Steps: []ir.Step{
					stageStep("safing", []string{"abort_1", "abort_2"}),
					stageStep("safed", nil),
				},
			},
			{
				Key: "recovery",
				Steps: []ir.Step{
					stageStep("assess", []string{"assess_1"}),
					stageStep("restart", []string{"restart_1"}),
					stageStep("complete", nil),
				},
			},
		}
	})

	Describe("Find", func() {
		It("Should find existing sequence", func() {
			seq, ok := sequences.Find("main")
			Expect(ok).To(BeTrue())
			Expect(seq.Key).To(Equal("main"))
			Expect(seq.Steps).To(HaveLen(2))
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

	Describe("FindStep", func() {
		It("Should find step and its parent sequence", func() {
			step, seq, ok := sequences.FindStep("safing")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("safing"))
			Expect(step.StageNodes()).To(Equal([]string{"abort_1", "abort_2"}))
			Expect(seq.Key).To(Equal("abort"))
		})

		It("Should find step in first sequence", func() {
			step, seq, ok := sequences.FindStep("start")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("start"))
			Expect(seq.Key).To(Equal("main"))
		})

		It("Should find step in last sequence", func() {
			step, seq, ok := sequences.FindStep("complete")
			Expect(ok).To(BeTrue())
			Expect(step.Key).To(Equal("complete"))
			Expect(seq.Key).To(Equal("recovery"))
		})

		It("Should return false for missing step", func() {
			_, _, ok := sequences.FindStep("nonexistent")
			Expect(ok).To(BeFalse())
		})

		It("Should return first match when step names collide", func() {
			seqs := ir.Sequences{
				{Key: "seq1", Steps: []ir.Step{stageStep("init", []string{"s1_init"})}},
				{Key: "seq2", Steps: []ir.Step{stageStep("init", []string{"s2_init"})}},
			}
			step, seq, ok := seqs.FindStep("init")
			Expect(ok).To(BeTrue())
			Expect(seq.Key).To(Equal("seq1"))
			Expect(step.Key).To(Equal("init"))
			Expect(step.StageNodes()).To(Equal([]string{"s1_init"}))
		})

		It("Should handle empty collection", func() {
			empty := ir.Sequences{}
			_, _, ok := empty.FindStep("any")
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

		It("Should preserve nested steps", func() {
			data := MustSucceed(json.Marshal(sequences))

			var restored ir.Sequences
			Expect(json.Unmarshal(data, &restored)).To(Succeed())

			main, ok := restored.Find("main")
			Expect(ok).To(BeTrue())
			Expect(main.Steps).To(HaveLen(2))
			Expect(main.Entry().Key).To(Equal("start"))

			abort, ok := restored.Find("abort")
			Expect(ok).To(BeTrue())
			Expect(abort.Steps).To(HaveLen(2))
			Expect(abort.Entry().Key).To(Equal("safing"))

			recovery, ok := restored.Find("recovery")
			Expect(ok).To(BeTrue())
			Expect(recovery.Steps).To(HaveLen(3))
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

		It("Should support independent NextStep() calls", func() {
			main := sequences.Get("main")
			abort := sequences.Get("abort")

			mainNext, _ := main.NextStep("start")
			abortNext, _ := abort.NextStep("safing")

			Expect(mainNext.Key).To(Equal("end"))
			Expect(abortNext.Key).To(Equal("safed"))
		})

		It("Should support FindStep across all sequences", func() {
			step1, seq1, ok1 := sequences.FindStep("start")
			step2, seq2, ok2 := sequences.FindStep("safing")
			step3, seq3, ok3 := sequences.FindStep("restart")

			Expect(ok1).To(BeTrue())
			Expect(ok2).To(BeTrue())
			Expect(ok3).To(BeTrue())

			Expect(seq1.Key).To(Equal("main"))
			Expect(seq2.Key).To(Equal("abort"))
			Expect(seq3.Key).To(Equal("recovery"))

			Expect(step1.StageNodes()).To(Equal([]string{"node_1"}))
			Expect(step2.StageNodes()).To(Equal([]string{"abort_1", "abort_2"}))
			Expect(step3.StageNodes()).To(Equal([]string{"restart_1"}))
		})
	})
})

var _ = Describe("Integration: Sequence with Edge Kinds", func() {
	It("Should represent a complete sequence state machine with edges", func() {
		sequences := ir.Sequences{
			{
				Key: "main",
				Steps: []ir.Step{
					stageStep("precheck", []string{"timer_1", "check_1", "precheck_entry"}),
					stageStep("pressurize", []string{"valve_ctrl", "pressure_monitor", "pressurize_entry"}),
					stageStep("complete", []string{"complete_entry"}),
				},
			},
		}

		edges := ir.Edges{
			{
				Source: ir.Handle{Node: "timer_1", Param: "output"},
				Target: ir.Handle{Node: "check_1", Param: "input"},
				Kind:   ir.EdgeKindContinuous,
			},
			{
				Source: ir.Handle{Node: "check_1", Param: "output"},
				Target: ir.Handle{Node: "pressurize_entry", Param: "activate"},
				Kind:   ir.EdgeKindConditional,
			},
			{
				Source: ir.Handle{Node: "valve_ctrl", Param: "output"},
				Target: ir.Handle{Node: "pressure_monitor", Param: "input"},
				Kind:   ir.EdgeKindContinuous,
			},
			{
				Source: ir.Handle{Node: "pressure_monitor", Param: "threshold_met"},
				Target: ir.Handle{Node: "complete_entry", Param: "activate"},
				Kind:   ir.EdgeKindConditional,
			},
		}

		Expect(sequences).To(HaveLen(1))
		main := sequences.Get("main")
		Expect(main.Steps).To(HaveLen(3))

		entry := main.Entry()
		Expect(entry.Key).To(Equal("precheck"))

		next, ok := main.NextStep("precheck")
		Expect(ok).To(BeTrue())
		Expect(next.Key).To(Equal("pressurize"))

		continuous := edges.GetByKind(ir.EdgeKindContinuous)
		conditional := edges.GetByKind(ir.EdgeKindConditional)

		Expect(continuous).To(HaveLen(2))
		Expect(conditional).To(HaveLen(2))

		// Verify EdgeKindConditional edges target entry nodes
		for _, e := range conditional {
			Expect(e.Target.Node).To(ContainSubstring("_entry"))
			Expect(e.Target.Param).To(Equal("activate"))
		}
	})
})

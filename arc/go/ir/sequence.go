// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"fmt"
	"strings"

	"github.com/samber/lo"
)

// Entry returns the entry step of the sequence (always Steps[0]).
// Panics if the sequence has no steps.
func (s Sequence) Entry() Step {
	if len(s.Steps) == 0 {
		panic("sequence has no steps")
	}
	return s.Steps[0]
}

// NextStep returns the step that follows the given step in definition order.
// Returns the step and true if found, or zero value and false if the given
// step is the last step or not found.
func (s Sequence) NextStep(stepKey string) (Step, bool) {
	for i, step := range s.Steps {
		if step.Key == stepKey {
			if i+1 < len(s.Steps) {
				return s.Steps[i+1], true
			}
			return Step{}, false
		}
	}
	return Step{}, false
}

// FindStep searches for a step by key within this sequence.
// Returns the step and true if found, or zero value and false otherwise.
func (s Sequence) FindStep(stepKey string) (Step, bool) {
	return lo.Find(s.Steps, func(step Step) bool { return step.Key == stepKey })
}

// Find searches for a sequence by key. Returns the sequence and true if found,
// or zero value and false otherwise.
func (s Sequences) Find(key string) (Sequence, bool) {
	return lo.Find(s, func(seq Sequence) bool { return seq.Key == key })
}

// Get returns the sequence with the given key. Panics if not found.
func (s Sequences) Get(key string) Sequence { return lo.Must(s.Find(key)) }

// FindStep searches for a step across all sequences. Returns the step,
// its parent sequence, and true if found. Returns zero values and false otherwise.
// If multiple sequences have steps with the same key, returns the first match.
func (s Sequences) FindStep(stepKey string) (Step, Sequence, bool) {
	for _, seq := range s {
		if step, ok := seq.FindStep(stepKey); ok {
			return step, seq, true
		}
	}
	return Step{}, Sequence{}, false
}

// IsFlow returns true if this step is a flow (leaf) step.
func (s Step) IsFlow() bool { return s.Flow != nil }

// IsStage returns true if this step is a stage (parallel) step.
func (s Step) IsStage() bool { return s.Stage != nil }

// IsSequence returns true if this step is a sequence (sequential) step.
func (s Step) IsSequence() bool { return s.Sequence != nil }

// StageNodes returns the node keys for a stage step in stratum order, or nil
// for other kinds.
func (s Step) StageNodes() []string {
	if s.Stage != nil {
		return s.Stage.Strata.Flatten()
	}
	return nil
}

// FlowNodes returns the node keys for a flow step, or nil for other kinds.
func (s Step) FlowNodes() []string {
	if s.Flow != nil {
		return s.Flow.Nodes
	}
	return nil
}

// String returns the string representation of the step.
func (s Step) String() string {
	return s.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (s Step) stringWithPrefix(prefix string) string {
	var b strings.Builder
	switch {
	case s.Flow != nil:
		lo.Must(fmt.Fprintf(&b, "%s (flow): [%s]\n",
			s.displayKey(), strings.Join(s.Flow.Nodes, ", ")))
	case s.Stage != nil:
		lo.Must(fmt.Fprintf(&b, "%s (stage): [%s]\n",
			s.displayKey(), strings.Join(s.Stage.Strata.Flatten(), ", ")))
		if len(s.Stage.Strata) > 0 {
			b.WriteString(s.Stage.Strata.stringWithPrefix(prefix))
		}
	case s.Sequence != nil:
		lo.Must(fmt.Fprintf(&b, "%s (sequence)\n", s.displayKey()))
		b.WriteString(s.Sequence.stringWithPrefix(prefix))
	}
	return b.String()
}

func (s Step) displayKey() string {
	if s.Key != "" {
		return s.Key
	}
	return "(anonymous)"
}

// String returns the string representation of the stage.
// Format: "key: [node1, node2, ...]"
func (s Stage) String() string {
	return s.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (s Stage) stringWithPrefix(prefix string) string {
	var b strings.Builder
	lo.Must(fmt.Fprintf(&b, "%s: [%s]", s.Key, strings.Join(s.Strata.Flatten(), ", ")))
	if len(s.Strata) > 0 {
		lo.Must(fmt.Fprintf(&b, "\n%s", s.Strata.stringWithPrefix(prefix)))
	}
	return b.String()
}

// String returns the string representation of the sequence.
func (s Sequence) String() string {
	return s.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (s Sequence) stringWithPrefix(prefix string) string {
	var b strings.Builder
	b.WriteString(s.Key)
	b.WriteString("\n")
	for i, step := range s.Steps {
		isLast := i == len(s.Steps)-1
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		stepChildPrefix := prefix + treeIndent(isLast)
		b.WriteString(step.stringWithPrefix(stepChildPrefix))
	}
	return b.String()
}

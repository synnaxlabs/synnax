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

// Entry returns the entry stage of the sequence (always Stages[0]).
// Panics if the sequence has no stages.
func (s Sequence) Entry() Stage {
	if len(s.Stages) == 0 {
		panic("sequence has no stages")
	}
	return s.Stages[0]
}

// NextStage returns the stage that follows the given stage in definition order.
// Returns the stage and true if found, or zero value and false if the given
// stage is the last stage or not found.
func (s Sequence) NextStage(stageKey string) (Stage, bool) {
	for i, stage := range s.Stages {
		if stage.Key == stageKey {
			if i+1 < len(s.Stages) {
				return s.Stages[i+1], true
			}
			return Stage{}, false
		}
	}
	return Stage{}, false
}

// FindStage searches for a stage by key within this sequence.
// Returns the stage and true if found, or zero value and false otherwise.
func (s Sequence) FindStage(stageKey string) (Stage, bool) {
	return lo.Find(s.Stages, func(stage Stage) bool { return stage.Key == stageKey })
}

// Find searches for a sequence by key. Returns the sequence and true if found,
// or zero value and false otherwise.
func (s Sequences) Find(key string) (Sequence, bool) {
	return lo.Find(s, func(seq Sequence) bool { return seq.Key == key })
}

// Get returns the sequence with the given key. Panics if not found.
func (s Sequences) Get(key string) Sequence { return lo.Must(s.Find(key)) }

// FindStage searches for a stage across all sequences. Returns the stage,
// its parent sequence, and true if found. Returns zero values and false otherwise.
// If multiple sequences have stages with the same key, returns the first match.
func (s Sequences) FindStage(stageKey string) (Stage, Sequence, bool) {
	for _, seq := range s {
		if stage, ok := seq.FindStage(stageKey); ok {
			return stage, seq, true
		}
	}
	return Stage{}, Sequence{}, false
}

// String returns the string representation of the stage.
// Format: "key: [node1, node2, ...]"
func (s Stage) String() string {
	return s.stringWithPrefix("")
}

// stringWithPrefix returns the string representation with tree formatting.
func (s Stage) stringWithPrefix(prefix string) string {
	var b strings.Builder
	lo.Must(fmt.Fprintf(&b, "%s: [%s]", s.Key, strings.Join(s.Nodes, ", ")))
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
	for i, stage := range s.Stages {
		isLast := i == len(s.Stages)-1
		b.WriteString(prefix)
		b.WriteString(treePrefix(isLast))
		stageChildPrefix := prefix + treeIndent(isLast)
		b.WriteString(stage.stringWithPrefix(stageChildPrefix))
		if len(stage.Strata) == 0 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

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

// Stage represents a state within a sequence. Nodes listed in the stage are
// active only when this stage is active. The runtime determines which nodes
// to execute based on the active stage.
type Stage struct {
	// Key is the unique identifier for this stage within its sequence.
	Key string `json:"key"`
	// Nodes contains the keys of nodes that belong to this stage.
	// These nodes are active only when this stage is active.
	Nodes []string `json:"nodes"`
}

// Sequence represents a state machine containing ordered stages. A sequence defines
// the structure of a sequential automation workflow. The entry point is always
// Stages[0], and the order of stages in the slice determines `next` resolution.
type Sequence struct {
	// Key is the unique identifier for this sequence (the sequence name).
	Key string `json:"key"`
	// Stages contains the stages in definition order. This order determines
	// what `next` resolves to for each stage. Entry point is always Stages[0].
	Stages []Stage `json:"stages"`
}

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

// Sequences is a collection of sequence definitions.
type Sequences []Sequence

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
	return fmt.Sprintf("%s: [%s]", s.Key, strings.Join(s.Nodes, ", "))
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
		b.WriteString(stage.String())
		b.WriteString("\n")
	}
	return b.String()
}

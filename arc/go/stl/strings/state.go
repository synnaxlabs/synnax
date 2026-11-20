// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings

import "github.com/synnaxlabs/x/errors"

const (
	// MaxBytes is the most bytes of transient strings a ProgramState holds between two
	// calls to Clear. Each handle also counts HandleBytes, which keeps transient
	// handles below the literal handle base.
	MaxBytes = 16 << 20
	// HandleBytes is the cost that each transient handle adds to MaxBytes.
	HandleBytes = 64
)

// ErrLimit is the panic value of Create and CreateView when a string would pass
// MaxBytes. A host function that panics traps the guest call that made it.
var ErrLimit = errors.New("program made too many strings in one cycle")

// literalHandleBase is the starting value for literal string handles.
// Literal string handles are stable for the ProgramState lifetime and are never
// cleared by Clear. Using a high base value ensures they cannot collide with
// transient string handles, which start at 1 and reset back to 1 on every
// Clear call.
const literalHandleBase uint32 = 1 << 24

// ProgramState manages transient and literal string handles.
// Transient handles are short-lived references used within a single execution
// cycle and cleared on each flush. Literal handles persist for the ProgramState
// lifetime.
type ProgramState struct {
	strings              map[uint32]string
	counter              uint32
	bytes                int
	literalStrings       map[uint32]string
	literalStringCounter uint32
}

// NewProgramState creates a new strings ProgramState.
//
// ProgramState is program-scoped (one instance per running Arc program). It
// follows the codebase convention of using ProgramState for program-level
// runtime data, distinct from node-level State in runtime/node.
func NewProgramState() *ProgramState {
	return &ProgramState{
		strings:              make(map[uint32]string),
		counter:              1,
		literalStrings:       make(map[uint32]string),
		literalStringCounter: literalHandleBase,
	}
}

// Create stores a string and returns a transient handle for later retrieval. It panics
// with ErrLimit when the string would pass MaxBytes.
func (s *ProgramState) Create(str string) uint32 {
	return s.create(str, HandleBytes+len(str))
}

// CreateView stores a string that is held elsewhere, such as a stateful variable, and
// returns a transient handle for it. Only its handle counts toward MaxBytes. It panics
// with ErrLimit when the handle would pass MaxBytes.
func (s *ProgramState) CreateView(str string) uint32 {
	return s.create(str, HandleBytes)
}

func (s *ProgramState) create(str string, cost int) uint32 {
	if str == "" {
		return 0
	}
	if s.bytes+cost > MaxBytes {
		panic(ErrLimit)
	}
	handle := s.counter
	s.counter++
	s.bytes += cost
	s.strings[handle] = str
	return handle
}

// CreateLiteral stores a string and returns a stable handle that persists
// for the lifetime of the ProgramState and is never cleared by Clear.
// Use this for literal param strings whose handles are baked into node args
// at configure time.
func (s *ProgramState) CreateLiteral(str string) uint32 {
	if str == "" {
		return 0
	}
	handle := s.literalStringCounter
	s.literalStringCounter++
	s.literalStrings[handle] = str
	return handle
}

// Get retrieves a string by its handle.
// Checks transient strings first, then persistent literal strings.
func (s *ProgramState) Get(handle uint32) (string, bool) {
	if handle == 0 {
		return "", true
	}
	if str, ok := s.strings[handle]; ok {
		return str, true
	}
	str, ok := s.literalStrings[handle]
	return str, ok
}

// Clear removes transient strings and resets the transient counter.
// Literal strings are preserved.
func (s *ProgramState) Clear() {
	clear(s.strings)
	s.counter = 1
	s.bytes = 0
}

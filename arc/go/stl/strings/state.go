// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings

// configHandleBase is the starting value for config string handles.
// Config string handles are stable for the ProgramState lifetime and are never
// cleared by Clear. Using a high base value ensures they cannot collide with
// transient string handles, which start at 1 and reset back to 1 on every
// Clear call.
const configHandleBase uint32 = 1 << 24

// ProgramState manages transient and config string handles.
// Transient handles are short-lived references used within a single execution
// cycle and cleared on each flush. Config handles persist for the ProgramState
// lifetime.
type ProgramState struct {
	strings             map[uint32]string
	counter             uint32
	configStrings       map[uint32]string
	configStringCounter uint32
}

// NewProgramState creates a new ProgramState.
func NewProgramState() *ProgramState {
	return &ProgramState{
		strings:             make(map[uint32]string),
		counter:             1,
		configStrings:       make(map[uint32]string),
		configStringCounter: configHandleBase,
	}
}

// Create stores a string and returns a transient handle for later retrieval.
func (s *ProgramState) Create(str string) uint32 {
	handle := s.counter
	s.counter++
	s.strings[handle] = str
	return handle
}

// CreateConfig stores a string and returns a stable handle that persists
// for the lifetime of the ProgramState and is never cleared by Clear.
// Use this for config param strings whose handles are baked into node args
// at configure time.
func (s *ProgramState) CreateConfig(str string) uint32 {
	handle := s.configStringCounter
	s.configStringCounter++
	s.configStrings[handle] = str
	return handle
}

// Get retrieves a string by its handle.
// Checks transient strings first, then persistent config strings.
func (s *ProgramState) Get(handle uint32) (string, bool) {
	if str, ok := s.strings[handle]; ok {
		return str, true
	}
	str, ok := s.configStrings[handle]
	return str, ok
}

// Clear removes transient strings and resets the transient counter.
// Config strings are preserved.
func (s *ProgramState) Clear() {
	clear(s.strings)
	s.counter = 1
}

// Reset removes all strings including config strings.
func (s *ProgramState) Reset() {
	s.Clear()
	clear(s.configStrings)
	s.configStringCounter = configHandleBase
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package series

import "github.com/synnaxlabs/x/telem"

// ProgramState manages transient series handles.
// Handles are short-lived references used within a single execution cycle
// and cleared on each flush.
type ProgramState struct {
	series  map[uint32]telem.Series
	counter uint32
}

// NewProgramState creates a new ProgramState.
func NewProgramState() *ProgramState {
	return &ProgramState{
		series:  make(map[uint32]telem.Series),
		counter: 1,
	}
}

// Store stores a series and returns a handle for later retrieval.
func (s *ProgramState) Store(series telem.Series) uint32 {
	handle := s.counter
	s.counter++
	s.series[handle] = series
	return handle
}

// Get retrieves a series by its handle.
func (s *ProgramState) Get(handle uint32) (telem.Series, bool) {
	series, ok := s.series[handle]
	return series, ok
}

// Clear removes all stored series and resets the counter.
func (s *ProgramState) Clear() {
	clear(s.series)
	s.counter = 1
}

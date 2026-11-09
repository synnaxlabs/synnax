// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package series

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const (
	// MaxHandles is the most series a ProgramState holds between two calls to Clear.
	MaxHandles = 1 << 16
	// MaxBytes is the most bytes of series data a ProgramState holds between two calls
	// to Clear.
	MaxBytes = 64 << 20
)

// ErrLimit is the panic value of Store when a series would pass MaxHandles or
// MaxBytes. A host function that panics traps the guest call that made it.
var ErrLimit = errors.New("program made too many series in one cycle")

// ProgramState manages transient series handles.
// Handles are short-lived references used within a single execution cycle
// and cleared on each flush.
type ProgramState struct {
	series  map[uint32]telem.Series
	counter uint32
	bytes   int64
}

// NewProgramState creates a new ProgramState.
func NewProgramState() *ProgramState {
	return &ProgramState{
		series:  make(map[uint32]telem.Series),
		counter: 1,
	}
}

// Store stores a series and returns a handle for later retrieval. It panics with
// ErrLimit when the series would pass MaxHandles or MaxBytes.
func (s *ProgramState) Store(series telem.Series) uint32 {
	size := int64(series.Size())
	if s.counter > MaxHandles || s.bytes+size > MaxBytes {
		panic(ErrLimit)
	}
	handle := s.counter
	s.counter++
	s.bytes += size
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
	s.bytes = 0
}

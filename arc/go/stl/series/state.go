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
	// MaxBytes is the most bytes of series a ProgramState holds between two calls to
	// Clear. Each handle also counts HandleBytes.
	MaxBytes = 64 << 20
	// HandleBytes is the cost that each handle adds to MaxBytes.
	HandleBytes = 64
)

// ErrLimit is the panic value of Store and StoreView when a series would pass
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

// Store stores a series that the caller allocated for it and returns a handle for
// later retrieval. It panics with ErrLimit when the series would pass MaxBytes.
func (s *ProgramState) Store(series telem.Series) uint32 {
	return s.store(series, HandleBytes+int64(series.Size()))
}

// StoreView stores a series that shares its data with memory held elsewhere, such as
// a stateful variable. Only its handle counts toward MaxBytes.
func (s *ProgramState) StoreView(series telem.Series) uint32 {
	return s.store(series, HandleBytes)
}

func (s *ProgramState) store(series telem.Series, cost int64) uint32 {
	if s.bytes+cost > MaxBytes {
		panic(ErrLimit)
	}
	handle := s.counter
	s.counter++
	s.bytes += cost
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

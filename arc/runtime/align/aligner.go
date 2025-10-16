// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package align

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
)

// entry stores accumulated data for a single input
type entry struct {
	Data      telem.MultiSeries // Accumulated data series
	Time      telem.MultiSeries // Accumulated time series (parallel to Data)
	Watermark telem.TimeStamp   // Highest timestamp processed
}

// AlignedInput represents aligned data ready for computation
type AlignedInput struct {
	Data      telem.Series
	Time      telem.Series
	Alignment telem.AlignmentBounds
}

// Operation represents a complete set of aligned inputs ready to process
type Operation struct {
	Inputs map[string]AlignedInput
}

// Aligner manages multi-input alignment using CombineLatest semantics.
// It accumulates data from multiple inputs and produces aligned operations
// when all inputs have data available.
type Aligner struct {
	entries    map[string]*entry
	inputKeys  []string // Track input order
	readyCheck func(map[string]*entry) bool
}

// NewAligner creates a new Aligner for the given input keys.
// inputKeys must contain all expected inputs (e.g., ["lhs", "rhs"] for binary ops)
func NewAligner(inputKeys []string) *Aligner {
	entries := make(map[string]*entry)
	for _, key := range inputKeys {
		entries[key] = &entry{
			Data:      telem.MultiSeries{},
			Time:      telem.MultiSeries{},
			Watermark: 0,
		}
	}
	return &Aligner{
		entries:   entries,
		inputKeys: inputKeys,
		readyCheck: func(entries map[string]*entry) bool {
			// CombineLatest: all inputs must have at least one series
			for _, e := range entries {
				if len(e.Data.Series) == 0 {
					return false
				}
			}
			return true
		},
	}
}

// Add ingests new data and time series for the given input key.
// Data and Time must have matching lengths (parallel series).
func (a *Aligner) Add(key string, data telem.Series, time telem.Series) error {
	e, ok := a.entries[key]
	if !ok {
		return errors.Newf("[aligner] unknown input key: %s", key)
	}

	if data.Len() != time.Len() {
		return errors.Newf(
			"[aligner] data and time length mismatch: data=%d, time=%d",
			data.Len(),
			time.Len(),
		)
	}

	// Append to accumulated series
	e.Data.Series = append(e.Data.Series, data)
	e.Time.Series = append(e.Time.Series, time)

	return nil
}

// Next produces the next aligned operation using CombineLatest semantics.
// Returns (Operation, true) if an operation is ready, or (Operation{}, false) if not.
//
// CombineLatest semantics:
// - Waits until ALL inputs have data
// - Processes series in chronological order (earliest unprocessed timestamp first)
// - Combines earliest new value with latest values from other inputs
// - Preserves latest series from each input for reuse with future inputs
func (a *Aligner) Next() (Operation, bool) {
	// Readiness check: all inputs must have data
	if !a.readyCheck(a.entries) {
		return Operation{}, false
	}

	// Find the input with the earliest unprocessed timestamp
	var triggerKey string
	var triggerTimestamp telem.TimeStamp
	var triggerIdx int = -1

	for key, e := range a.entries {
		for i, timeSeries := range e.Time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)

			// Only consider timestamps that haven't been processed yet
			if ts > e.Watermark {
				// First valid timestamp, or earlier than current trigger
				if triggerIdx == -1 || ts < triggerTimestamp {
					triggerKey = key
					triggerTimestamp = ts
					triggerIdx = i
				}
			}
		}
	}

	// No unprocessed data available
	if triggerIdx == -1 {
		return Operation{}, false
	}

	// Build operation: use earliest unprocessed from trigger input,
	// latest available from all other inputs
	op := Operation{Inputs: make(map[string]AlignedInput)}

	for key, e := range a.entries {
		var dataSeries telem.Series
		var timeSeries telem.Series

		if key == triggerKey {
			// Use the earliest unprocessed series (the trigger)
			dataSeries = e.Data.Series[triggerIdx]
			timeSeries = e.Time.Series[triggerIdx]
		} else {
			// Use the latest available series (may be already processed)
			// This implements "combine with latest" semantics
			latestIdx := len(e.Data.Series) - 1
			dataSeries = e.Data.Series[latestIdx]
			timeSeries = e.Time.Series[latestIdx]
		}

		op.Inputs[key] = AlignedInput{
			Data:      dataSeries,
			Time:      timeSeries,
			Alignment: dataSeries.AlignmentBounds(),
		}
	}

	// Update watermark only for the triggered input
	a.entries[triggerKey].Watermark = triggerTimestamp

	// Cleanup: remove processed series, but keep at least the latest series
	// per input for reuse with future operations (CombineLatest semantics)
	for _, e := range a.entries {
		newData := []telem.Series{}
		newTime := []telem.Series{}

		// Keep all series with timestamp > watermark
		for i, timeSeries := range e.Time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > e.Watermark {
				newData = append(newData, e.Data.Series[i])
				newTime = append(newTime, timeSeries)
			}
		}

		// If all series were processed, keep the latest one for reuse
		if len(newData) == 0 && len(e.Data.Series) > 0 {
			lastIdx := len(e.Data.Series) - 1
			newData = []telem.Series{e.Data.Series[lastIdx]}
			newTime = []telem.Series{e.Time.Series[lastIdx]}
		}

		e.Data.Series = newData
		e.Time.Series = newTime
	}

	return op, true
}

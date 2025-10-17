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

type entry struct {
	Data      telem.MultiSeries
	Time      telem.MultiSeries
	Watermark telem.TimeStamp
}

type AlignedInput struct {
	Data      telem.Series
	Time      telem.Series
	Alignment telem.AlignmentBounds
}

type Operation struct {
	Inputs map[string]AlignedInput
}

type Aligner struct {
	entries    map[string]*entry
	inputKeys  []string // Track input order
	readyCheck func(map[string]*entry) bool
}

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
	e.Data.Series = append(e.Data.Series, data)
	e.Time.Series = append(e.Time.Series, time)
	return nil
}

func (a *Aligner) Next() (Operation, bool) {
	if !a.readyCheck(a.entries) {
		return Operation{}, false
	}

	var (
		triggerKey       string
		triggerTimestamp telem.TimeStamp
		triggerIdx       int = -1
	)
	for key, e := range a.entries {
		for i, timeSeries := range e.Time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > e.Watermark {
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

	op := Operation{Inputs: make(map[string]AlignedInput)}

	for key, e := range a.entries {
		var dataSeries telem.Series
		var timeSeries telem.Series

		if key == triggerKey {
			dataSeries = e.Data.Series[triggerIdx]
			timeSeries = e.Time.Series[triggerIdx]
		} else {
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

	a.entries[triggerKey].Watermark = triggerTimestamp

	for _, e := range a.entries {
		var (
			newData []telem.Series
			newTime []telem.Series
		)
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

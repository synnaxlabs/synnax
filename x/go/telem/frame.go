// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"fmt"
	"iter"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/bit"
)

// Frame is a performance optimized record of comparable keys to series.
type Frame[K comparable] struct {
	// keys are the keys of the frame record. keys is guaranteed to have the same length
	// as series. Note that keys are not guaranteed to be unique.
	keys []K
	// series is the series of the frame record. series is guaranteed to have the same length
	// as keys.
	series []Series
	// mask is used as a high-performance filter for removing entries from the frame.
	// it allows for an alternative to filtering by creating copies of the key and
	// series slices. Mask can only handle frames with up to 128 entries. If the
	// mask is not nil, it is enabled and will be used to filter the frame.
	mask *bit.Mask128
}

// UnaryFrame creates a new frame with a single key and series.
func UnaryFrame[T comparable](key T, series Series) Frame[T] {
	return Frame[T]{keys: []T{key}, series: []Series{series}}
}

func MultiFrame[T comparable](keys []T, series []Series) Frame[T] {
	if len(keys) != len(series) {
		panic("keys and series must be of the same length")
	}
	return Frame[T]{keys: keys, series: series}
}

// String returns a nicely formatted string representation of the frame. This function
// should not be used in performance critical paths.
func (f Frame[K]) String() string {
	if f.Empty() {
		return "Frame{}"
	}
	var b strings.Builder
	b.WriteString("Frame{\n")
	for key, series := range f.Entries() {
		b.WriteString(fmt.Sprintf(" %s: %v", fmt.Sprintf("%v", key), series))
		b.WriteString(",\n")
	}
	b.WriteString("}")
	return b.String()
}

// shouldExclude returns true if the entry at index i should be excluded from frame
// operations.
func (f Frame[K]) shouldExclude(i int) bool {
	return f.mask != nil && f.mask.Get(i)
}

// KeysSlice returns the slice of keys in the frame. If FilterKeys has been called
// on the frame, this function will have a considerable performance impact on the
// hot path. If not, the performance impact is negligible.
func (f Frame[K]) KeysSlice() []K {
	if f.mask == nil {
		return f.keys
	}
	keys := make([]K, 0, len(f.keys))
	for k := range f.Keys() {
		keys = append(keys, k)
	}
	return keys
}

// SeriesSlice returns the slice of series in the frame. If FilterKeys has been called
// on the frame, this function will have a considerable performance impact on the
// hot path. If not, the performance impact is negligible.
func (f Frame[K]) SeriesSlice() []Series {
	if f.mask == nil {
		return f.series
	}
	series := make([]Series, 0, len(f.series))
	for s := range f.Series() {
		series = append(series, s)
	}
	return series
}

// Series returns an iterable sequence of series in the frame.
func (f Frame[K]) Series() iter.Seq[Series] {
	return func(yield func(Series) bool) {
		for i, s := range f.series {
			if f.shouldExclude(i) {
				continue
			}
			if !yield(s) {
				return
			}
		}
	}
}

// Keys returns an iterable sequence of keys in the frame.
func (f Frame[K]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for i, k := range f.keys {
			if f.shouldExclude(i) {
				continue
			}
			if !yield(k) {
				return
			}
		}
	}
}

// Entries returns an iterable sequence of key-series pairs in the frame.
func (f Frame[K]) Entries() iter.Seq2[K, Series] {
	return func(yield func(K, Series) bool) {
		for i, k := range f.keys {
			if f.shouldExclude(i) {
				continue
			}
			if !yield(k, f.series[i]) {
				return
			}
		}
	}
}

// Get gets all series in the frame matching the given key.
func (f Frame[K]) Get(key K) []Series {
	series := make([]Series, 0, len(f.keys))
	for i, k := range f.keys {
		if k == key && !f.shouldExclude(i) {
			series = append(series, f.series[i])
		}
	}
	return series
}

// Append adds a new key-series pair to the frame, returning the updated frame.
func (f Frame[K]) Append(key K, series Series) Frame[K] {
	f.keys = append(f.keys, key)
	f.series = append(f.series, series)
	return f
}

// Empty returns true if the frame has no keys or series.
func (f Frame[K]) Empty() bool {
	if f.mask != nil {
		return len(f.series)-f.mask.TrueCount() == 0
	}
	return len(f.series) == 0
}

func (f Frame[K]) Vertical() bool {
	uniqueKeys := lo.Uniq(f.KeysSlice())
	return len(uniqueKeys) == len(f.SeriesSlice())
}

func (f Frame[K]) Even() bool {
	if f.Empty() {
		return true
	}
	var first *Series
	for s := range f.Series() {
		if first == nil {
			first = &s
		} else if s.Len() != first.Len() || s.TimeRange != first.TimeRange {
			return false
		}
	}
	return true
}

func (f Frame[K]) Len() int64 {
	for i, s := range f.series {
		if !f.shouldExclude(i) {
			return s.Len()
		}
	}
	return 0
}

// FilterKeys filters the frame to only include the keys in the given slice, returning
// a shallow copy of the filtered frame.
func (f Frame[K]) FilterKeys(keys []K) Frame[K] {
	if len(f.keys) < f.mask.Size() {
		if f.mask == nil {
			f.mask = &bit.Mask128{}
		}
		for i, key := range f.keys {
			if !lo.Contains(keys, key) {
				f.mask.Set(i, true)
			}
		}
		return f
	}
	var (
		fKeys   = make([]K, 0, len(keys))
		fArrays = make([]Series, 0, len(keys))
	)
	for key, ser := range f.Entries() {
		if lo.Contains(keys, key) {
			fKeys = append(fKeys, key)
			fArrays = append(fArrays, ser)
		}
	}
	return Frame[K]{keys: fKeys, series: fArrays}
}

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
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/bit"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/vmihailenco/msgpack/v5"
)

// Frame is a performance-optimized record of numeric keys to series.
// size = 24 + 24 + 1 + 16 + (7 buffer) = 72 bytes.
type Frame[K types.SizedNumeric] struct {
	// keys are the keys of the frame record. the keys slice is guaranteed to have the
	// same length as the series slice. Note that keys are not guaranteed to be unique.
	// 24 bytes
	keys []K
	// series is the series of the frame record. the series slice is guaranteed to have
	// the same length as the keys slice.
	series []Series
	// 24 bytes
	// mask is used as a high-performance filter for removing entries from the frame.
	// it allows for an alternative to filtering by creating copies of the key and
	// series slices. Mask can only handle frames with up to 128 entries. If enabled
	// is true, the mask is enabled and will be used to filter the frame.
	mask struct {
		// 1 byte
		enabled bool
		// 16 bytes
		bit.Mask128
	}
}

// UnsafeReinterpretFrameKeysAs reinterprets the keys of the frame as a different type. This
// method performs no static type checking and is unsafe. Caveat emptor.
func UnsafeReinterpretFrameKeysAs[I, O types.SizedNumeric](f Frame[I]) Frame[O] {
	return Frame[O]{
		keys:   unsafe.ReinterpretSlice[I, O](f.keys),
		series: f.series,
		mask:   f.mask,
	}
}

// UnaryFrame creates a new frame with a single key and series.
func UnaryFrame[K types.SizedNumeric](key K, series Series) Frame[K] {
	return Frame[K]{keys: []K{key}, series: []Series{series}}
}

// MultiFrame creates a new Frame with multiple keys and series. The keys and series
// slices must be the same length, or this function will panic.
func MultiFrame[K types.SizedNumeric](keys []K, series []Series) Frame[K] {
	if len(keys) != len(series) {
		panic("keys and series must be of the same length")
	}
	return Frame[K]{keys: keys, series: series}
}

// AllocFrame allocates a new frame with the given capacity. The capacity is the maximum
// number of entries that can be added to the frame before it needs to be resized. The
// frame is not initialized, so the keys and series slices will be empty. This function
// is useful for creating a frame with a known number of entries
func AllocFrame[K types.SizedNumeric](cap int) Frame[K] {
	return Frame[K]{keys: make([]K, 0, cap), series: make([]Series, 0, cap)}
}

// String returns a nicely formatted string representation of the frame. This function
// should not be used in performance-critical paths.
func (f Frame[K]) String() string {
	if f.Empty() {
		return "Frame{}"
	}
	var b strings.Builder
	b.WriteString("Frame{\n")
	for key, series := range f.Entries() {
		b.WriteString(fmt.Sprintf(" %v: %v, \n", key, series))
	}
	b.WriteString("}")
	return b.String()
}

// ShouldExcludeRaw returns true if the entry at index i is excluded from the frame
// due to filtering rules.
func (f Frame[K]) ShouldExcludeRaw(rawIndex int) bool {
	return f.mask.enabled && f.mask.Get(rawIndex)
}

// KeysSlice returns the slice of keys in the frame.
//
// If the frame has been filtered via KeepKeys or ExcludeKeys, this function will
// result in a deep copy of the keys slice, and may have a considerable performance
// impact. If the frame has not been filtered, the performance impact is negligible.
func (f Frame[K]) KeysSlice() []K {
	if !f.mask.enabled {
		return f.keys
	}
	keys := make([]K, 0, len(f.keys))
	for k := range f.Keys() {
		keys = append(keys, k)
	}
	return keys
}

// SeriesSlice returns the slice of series in the frame.
//
// If the frame has been filtered via KeepKeys or ExcludeKeys, this function will
// result in a deep copy of the keys slice, and may have a considerable performance
// impact. If the frame has not been filtered, the performance impact is negligible.
func (f Frame[K]) SeriesSlice() []Series {
	if !f.mask.enabled {
		return f.series
	}
	series := make([]Series, 0, len(f.series))
	for s := range f.Series() {
		series = append(series, s)
	}
	return series
}

// RawSeries returns the raw slice of series in the frame. This includes any series that
// have been filtered out by KeepKeys or ExcludeKeys. To check whether an index in
// this slice has been filtered out, use ShouldExcludeRaw.
//
// It is generally recommended to avoid using this function except for
// performance-critical paths where the overhead of allocating returned closures
// through Series() and SeriesI() is too high.
//
// It is not safe to modify the contents of the returned slice.
func (f Frame[K]) RawSeries() []Series { return f.series }

// RawKeys returns the raw slice of keys in teh frame. This includes any keys that have
// been filtered out by KeepKeys or ExcludeKeys. To check whether an index in this
// slice has been filtered out, use ShouldExcludeRaw.
//
// It is not safe to modify the contents of the returned slice.
func (f Frame[K]) RawKeys() []K { return f.keys }

func (f Frame[K]) normalizeIndex(index int) int {
	length := len(f.series) - f.mask.TrueCount()
	if index < 0 {
		index = length + index
	}
	if index < 0 || index >= length {
		panic("index out of range")
	}
	if !f.mask.enabled {
		return index
	}
	for i := 0; i <= index; i++ {
		if f.mask.Get(i) {
			index++
		}
	}
	return index
}

// SeriesAt returns the series at the given index in the frame.
func (f Frame[K]) SeriesAt(i int) Series {
	return f.series[f.normalizeIndex(i)]
}

// RawSeriesAt returns the series at the given index in the frame without checking for
// filtering rules. This function is unsafe and should only be used in conjunction with
// indexes generated by RawSeries or RawKeys. To set the series at the given
// index, use SetRawSeriesAt.
func (f Frame[K]) RawSeriesAt(i int) Series {
	return f.series[i]
}

// SetSeriesAt sets the series at the given index in the frame.
func (f Frame[K]) SetSeriesAt(i int, s Series) {
	i = f.normalizeIndex(i)
	f.series[i] = s
}

// SetRawSeriesAt sets the series at the given index in the frame without checking for
// filtering rules. This function is unsafe and should only be used in conjunction with
// indexes generated by using RawSeriesAt, RawSeries, or RawKeys.
func (f Frame[K]) SetRawSeriesAt(i int, s Series) { f.series[i] = s }

// At returns the key:series pair at the given index in the frame.
func (f Frame[K]) At(i int) (K, Series) {
	i = f.normalizeIndex(i)
	return f.keys[i], f.series[i]
}

// Series returns an iterable sequence of series in the frame.
func (f Frame[K]) Series() iter.Seq[Series] {
	return func(yield func(Series) bool) {
		for i, s := range f.series {
			if !f.ShouldExcludeRaw(i) && !yield(s) {
				return
			}
		}
	}
}

// SeriesI returns an iterable sequence of series in the frame with their indices.
func (f Frame[K]) SeriesI() iter.Seq2[int, Series] {
	offset := 0
	return func(yield func(int, Series) bool) {
		for i, s := range f.series {
			if f.ShouldExcludeRaw(i) {
				offset++
				continue
			}
			if !yield(i-offset, s) {
				return
			}
		}
	}
}

// Entries returns an iterable sequence of key-series pairs in the frame.
func (f Frame[K]) Entries() iter.Seq2[K, Series] {
	return func(yield func(K, Series) bool) {
		for i, k := range f.keys {
			if !f.ShouldExcludeRaw(i) && !yield(k, f.series[i]) {
				return
			}
		}
	}
}

// HasData returns true if there are any samples in the frame.
func (f Frame[K]) HasData() bool {
	for i, s := range f.series {
		if !f.ShouldExcludeRaw(i) && s.Len() > 0 {
			return true
		}
	}
	return false
}

// Keys returns an iterable sequence of keys in the frame.
func (f Frame[K]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for i, k := range f.keys {
			if !f.ShouldExcludeRaw(i) && !yield(k) {
				return
			}
		}
	}
}

// RawKeyAt returns the series at the given index in the frame without checking for
// filtering rules. This function is unsafe and should only be used in conjunction
// with indexes generated by SeriesAt, RawSeries, or RawKeys. To set the series at the
// given index, use SetRawSeriesAt.
func (f Frame[K]) RawKeyAt(i int) K {
	return f.keys[i]
}

var (
	_ json.Marshaler        = Frame[int8]{}
	_ json.Unmarshaler      = &Frame[int8]{}
	_ msgpack.CustomEncoder = Frame[int8]{}
	_ msgpack.CustomDecoder = &Frame[int8]{}
)

// serializableFrame is a helper type for serializing the Frame to encoded
// representations.
type serializableFrame[K comparable] struct {
	Keys   []K      `json:"keys" msgpack:"keys"`
	Series []Series `json:"series" msgpack:"series"`
}

// MarshalJSON implements json.Marshaler to handle data masking.
func (f Frame[K]) MarshalJSON() ([]byte, error) {
	if !f.mask.enabled {
		return json.Marshal(serializableFrame[K]{Keys: f.keys, Series: f.series})
	}
	var (
		buf      bytes.Buffer
		anyAdded bool
	)
	buf.WriteString(`{"keys":[`)
	for k := range f.Keys() {
		if anyAdded {
			buf.WriteString(",")
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		anyAdded = true
	}
	buf.WriteString(`],"series":[`)
	anyAdded = false
	for s := range f.Series() {
		if anyAdded {
			buf.WriteString(",")
		}
		seriesBytes, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		buf.Write(seriesBytes)
		anyAdded = true
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

// UnmarshalJSON implements json.Marshaler to handle data masking.
func (f *Frame[K]) UnmarshalJSON(data []byte) error {
	var frame serializableFrame[K]
	if err := json.Unmarshal(data, &frame); err != nil {
		return err
	}
	f.keys = frame.Keys
	f.series = frame.Series
	return nil
}

// EncodeMsgpack implements msgpack.CustomEncoder to handle data masking.
func (f Frame[K]) EncodeMsgpack(enc *msgpack.Encoder) error {
	if !f.mask.enabled {
		return enc.Encode(serializableFrame[K]{Keys: f.keys, Series: f.series})
	}
	count := f.Count()
	if err := enc.EncodeMapLen(2); err != nil {
		return err
	}
	if err := enc.EncodeString("keys"); err != nil {
		return err
	}
	if err := enc.EncodeArrayLen(count); err != nil {
		return err
	}
	for k := range f.Keys() {
		if err := enc.Encode(k); err != nil {
			return err
		}
	}
	if err := enc.EncodeString("series"); err != nil {
		return err
	}
	if err := enc.EncodeArrayLen(count); err != nil {
		return err
	}
	for s := range f.Series() {
		if err := enc.Encode(s); err != nil {
			return err
		}
	}
	return nil
}

// DecodeMsgpack can continue using serializableFrame
func (f *Frame[K]) DecodeMsgpack(dec *msgpack.Decoder) error {
	var frame serializableFrame[K]
	err := dec.Decode(&frame)
	f.keys = frame.Keys
	f.series = frame.Series
	return err
}

// Get gets all series in the frame matching the given key.
func (f Frame[K]) Get(key K) MultiSeries {
	series := make([]Series, 0, len(f.keys))
	for i, k := range f.keys {
		if k == key && !f.ShouldExcludeRaw(i) {
			series = append(series, f.series[i])
		}
	}
	return MultiSeries{Series: series}
}

// Append adds a new key-series pair to the end of teh frame, returning the updated
// frame.
func (f Frame[K]) Append(key K, series Series) Frame[K] {
	f.keys = append(f.keys, key)
	f.series = append(f.series, series)
	return f
}

// Prepend adds a new key-series pair to the beginning of the frame, returning the
// updated frame.
func (f Frame[K]) Prepend(key K, series Series) Frame[K] {
	f.keys = append([]K{key}, f.keys...)
	f.series = append([]Series{series}, f.series...)
	return f
}

// Empty returns true if the frame has no keys or series.
func (f Frame[K]) Empty() bool {
	return f.Count() == 0
}

// Len returns the largest total number of samples in all series for a particular key.
func (f Frame[K]) Len() int64 {
	var (
		length  int64
		lengths = make(map[K]int64)
	)
	for k, s := range f.Entries() {
		v := lengths[k] + s.Len()
		lengths[k] = v
		if v > length {
			length = v
		}
	}
	return length
}

// Count returns the number of keys/series in the frame.
func (f Frame[K]) Count() int {
	if f.mask.enabled {
		return len(f.series) - f.mask.TrueCount()
	}
	return len(f.series)
}

// Extend extends the frame by appending the keys and series from another frame to it.
func (f Frame[K]) Extend(frames ...Frame[K]) Frame[K] {
	for _, fr := range frames {
		f.keys = append(f.keys, fr.KeysSlice()...)
		f.series = append(f.series, fr.SeriesSlice()...)
	}
	return f
}

// ShallowCopy returns a shallow copy of the frame, i.e., the keys and series slices
// are copied, but the series themselves are not.
func (f Frame[K]) ShallowCopy() Frame[K] {
	keys := make([]K, len(f.keys))
	copy(keys, f.keys)
	series := make([]Series, len(f.series))
	copy(series, f.series)
	return Frame[K]{keys: keys, series: series, mask: f.mask}
}

// KeepKeys filters the frame to only include the keys in the given slice, returning
// a shallow copy of the filtered frame.
func (f Frame[K]) KeepKeys(keys []K) Frame[K] {
	return f.filter(keys, lo.Contains)
}

// notContain is defined statically so that it doesn't accidentally escape to the heap,
// applying more pressure on GC
func notContains[T comparable](s []T, e T) bool { return !lo.Contains(s, e) }

// ExcludeKeys filters the frame to include any keys that are NOT in the given slice,
// returning a shallow copy of the filtered frame.
func (f Frame[K]) ExcludeKeys(keys []K) Frame[K] {
	if len(keys) == 0 {
		return f
	}
	return f.filter(keys, notContains)
}

func (f Frame[K]) filter(keys []K, keep func([]K, K) bool) Frame[K] {
	if len(f.keys) < f.mask.Cap() {
		f.mask.enabled = true
		for i, key := range f.keys {
			if !keep(keys, key) {
				f.mask.Mask128 = f.mask.Set(i, true)
			}
		}
		return f
	}
	var (
		fKeys   = make([]K, 0, len(keys))
		fSeries = make([]Series, 0, len(keys))
	)
	for k, s := range f.Entries() {
		if keep(keys, k) {
			fKeys = append(fKeys, k)
			fSeries = append(fSeries, s)
		}
	}
	return Frame[K]{keys: fKeys, series: fSeries}
}

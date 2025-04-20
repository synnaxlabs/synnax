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
	"github.com/synnaxlabs/x/unsafe"
	"github.com/vmihailenco/msgpack/v5"
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

func ReinterpretKeysAs[I, O comparable](f Frame[I]) Frame[O] {
	return Frame[O]{
		keys:   unsafe.ReinterpretSlice[I, O](f.keys),
		series: f.series,
		mask:   f.mask.Copy(),
	}
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

func (f Frame[K]) normalizeIndex(index int) int {
	length := len(f.series)
	if index < 0 {
		index = length + index
	}
	if index < 0 || index >= length {
		panic("index out of range")
	}
	if f.mask == nil {
		return index
	}
	for i := 0; i < index; i++ {
		if f.mask.Get(i) {
			index++
		}
	}
	return index
}

func (f Frame[K]) SeriesAt(index int) Series {
	return f.series[f.normalizeIndex(index)]
}

func (f Frame[K]) SetSeriesAt(index int, series Series) {
	index = f.normalizeIndex(index)
	f.series[index] = series
}

func (f Frame[K]) At(index int) (K, Series) {
	index = f.normalizeIndex(index)
	return f.keys[index], f.series[index]
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

func (f Frame[K]) SeriesI() iter.Seq2[int, Series] {
	return func(yield func(int, Series) bool) {
		for i, s := range f.series {
			if f.shouldExclude(i) {
				continue
			}
			if !yield(i, s) {
				return
			}
		}
	}
}

func (f Frame[K]) HasData() bool {
	return len(f.series) > 0
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

var (
	_ json.Marshaler        = Frame[int]{}
	_ json.Unmarshaler      = &Frame[int]{}
	_ msgpack.CustomEncoder = Frame[int]{}
	_ msgpack.CustomDecoder = &Frame[int]{}
)

type frameRepr[K comparable] struct {
	Keys   []K      `json:"keys" msgpack:"keys"`
	Series []Series `json:"series" msgpack:"series"`
}

// MarshalJSON implements json.Marshaler to handle data masking.
func (f Frame[K]) MarshalJSON() ([]byte, error) {
	if f.mask == nil {
		return json.Marshal(frameRepr[K]{Keys: f.keys, Series: f.series})
	}

	var buf bytes.Buffer
	buf.WriteString(`{"keys":[`)
	var addedCount int
	for k := range f.Keys() {
		if addedCount > 0 {
			buf.WriteString(",")
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		addedCount++
	}
	buf.WriteString(`],"series":[`)
	addedCount = 0
	for s := range f.Series() {
		if addedCount > 0 {
			buf.WriteString(",")
		}
		seriesBytes, err := json.Marshal(s)
		if err != nil {
			return nil, err
		}
		buf.Write(seriesBytes)
		addedCount++
	}
	buf.WriteString(`]}`)
	return buf.Bytes(), nil
}

// UnmarshalJSON can continue using frameRepr
func (f *Frame[K]) UnmarshalJSON(data []byte) error {
	var frame frameRepr[K]
	if err := json.Unmarshal(data, &frame); err != nil {
		return err
	}
	f.keys = frame.Keys
	f.series = frame.Series
	return nil
}

// EncodeMsgpack implements msgpack.CustomEncoder to handle data masking.
func (f Frame[K]) EncodeMsgpack(enc *msgpack.Encoder) error {
	// If no mask, encode directly
	if f.mask == nil {
		return enc.Encode(frameRepr[K]{Keys: f.keys, Series: f.series})
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

// DecodeMsgpack can continue using frameRepr
func (f *Frame[K]) DecodeMsgpack(dec *msgpack.Decoder) error {
	var frame frameRepr[K]
	if err := dec.Decode(&frame); err != nil {
		return err
	}
	f.keys = frame.Keys
	f.series = frame.Series
	return nil
}

func (f Frame[K]) KeysI() iter.Seq2[int, K] {
	return func(yield func(int, K) bool) {
		for i, k := range f.keys {
			if f.shouldExclude(i) {
				continue
			}
			if !yield(i, k) {
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
func (f Frame[K]) Get(key K) MultiSeries {
	series := make([]Series, 0, len(f.keys))
	for i, k := range f.keys {
		if k == key && !f.shouldExclude(i) {
			series = append(series, f.series[i])
		}
	}
	return MultiSeries{Series: series}
}

// Append adds a new key-series pair to the frame, returning the updated frame.
func (f Frame[K]) Append(key K, series Series) Frame[K] {
	f.keys = append(f.keys, key)
	f.series = append(f.series, series)
	return f
}

func (f Frame[K]) Prepend(key K, series Series) Frame[K] {
	f.keys = append([]K{key}, f.keys...)
	f.series = append([]Series{series}, f.series...)
	return f
}

// Empty returns true if the frame has no keys or series.
func (f Frame[K]) Empty() bool {
	return f.Count() == 0
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

func (f Frame[K]) Len() (len int64) {
	var firstKey *K
	for k, s := range f.Entries() {
		if firstKey == nil {
			firstKey = &k
		}
		if k == *firstKey {
			len += s.Len()
		}
	}
	return
}

func (f Frame[K]) Count() int {
	if f.mask != nil {
		return len(f.series) - f.mask.TrueCount()
	}
	return len(f.series)
}

func (f Frame[K]) Extend(fr Frame[K]) Frame[K] {
	if f.mask != nil {
		f.mask = nil
	}
	f.keys = append(f.keys, fr.KeysSlice()...)
	f.series = append(f.series, fr.SeriesSlice()...)
	return f
}

func (f Frame[K]) ShallowCopy() Frame[K] {
	keys := make([]K, len(f.keys))
	copy(keys, f.keys)
	series := make([]Series, len(f.series))
	copy(series, f.series)
	return Frame[K]{keys: keys, series: series, mask: f.mask.Copy()}
}

// FilterKeys filters the frame to only include the keys in the given slice, returning
// a shallow copy of the filtered frame.
func (f Frame[K]) FilterKeys(keys []K) Frame[K] {
	//if len(f.keys) < f.mask.Size() {
	//	if f.mask == nil {
	//		f.mask = &bit.Mask128{}
	//	} else {
	//		f.mask = f.mask.Copy()
	//	}
	//	for i, key := range f.keys {
	//		if !lo.Contains(keys, key) {
	//			f.mask.Set(i, true)
	//		}
	//	}
	//	if f.mask.TrueCount() == len(f.keys) && len(f.keys) == 3 {
	//		fmt.Println("COND")
	//	}
	//	return f
	//}
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

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/exp/slices"
)

type Frame struct {
	Keys   []ChannelKey
	Series []telem.Series
}

func NewFrame(keys []ChannelKey, series []telem.Series) Frame {
	if len(keys) != len(series) {
		panic("keys and telemetry series in a frame must be of the same length")
	}
	kf := Frame{Keys: keys, Series: series}
	return kf
}

func (f Frame) UniqueKeys() []ChannelKey { return lo.Uniq(f.Keys) }

func (f Frame) Key(i int) ChannelKey { return f.Keys[i] }

func (f Frame) Append(key ChannelKey, series telem.Series) Frame {
	return NewFrame(append(f.Keys, key), append(f.Series, series))
}

func (f Frame) Get(key ChannelKey) []telem.Series {
	return lo.Filter(f.Series, func(_ telem.Series, i int) bool {
		return f.Keys[i] == key
	})
}

func (f Frame) Prepend(key ChannelKey, series telem.Series) Frame {
	return NewFrame(append([]uint32{key}, f.Keys...), append([]telem.Series{series}, f.Series...))
}

func (f Frame) AppendMany(keys []ChannelKey, series []telem.Series) Frame {
	return NewFrame(append(f.Keys, keys...), append(f.Series, series...))
}

func (f Frame) PrependMany(keys []ChannelKey, series []telem.Series) Frame {
	return NewFrame(append(keys, f.Keys...), append(series, f.Series...))
}

func (f Frame) AppendFrame(frame Frame) Frame { return f.AppendMany(frame.Keys, frame.Series) }

func (f Frame) FilterKeys(keys []ChannelKey) Frame {
	if slices.Equal(keys, f.Keys) {
		return f
	}
	var (
		filteredKeys   = make([]ChannelKey, 0, len(keys))
		filteredArrays = make([]telem.Series, 0, len(keys))
	)
	for i, key := range f.Keys {
		if lo.Contains(keys, key) {
			filteredKeys = append(filteredKeys, key)
			filteredArrays = append(filteredArrays, f.Series[i])
		}
	}
	return NewFrame(filteredKeys, filteredArrays)
}

func (f Frame) Unary() bool { return len(f.Keys) == len(f.UniqueKeys()) }

func (f Frame) Even() bool {
	for i := 1; i < len(f.Series); i++ {
		if f.Series[i].Len() != f.Series[0].Len() {
			return false
		}
		if f.Series[i].TimeRange != f.Series[0].TimeRange {
			return false
		}
	}
	return true
}

// SquashSameKeyData is meant for testing use only. It is NOT optimized.
func (f Frame) SquashSameKeyData(key ChannelKey) (data []byte) {
	for i := 0; i < len(f.Keys); i++ {
		if f.Keys[i] == key {
			data = append(data, f.Series[i].Data...)
		}
	}

	return
}

// Len returns the length of all series in the frame,
// if a series has a different length, Len panics.
func (f Frame) Len() int64 {
	f.assertEven("Len")
	if len(f.Series) == 0 {
		return 0
	}
	return f.Series[0].Len()
}

func (f Frame) Empty() bool {
	return len(f.Series) == 0
}

func (f Frame) assertEven(method string) {
	if !f.Even() {
		panic("[telem] - cannot call " + method + " on uneven frame")
	}
}

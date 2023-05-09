// Copyright 2023 Synnax Labs, Inc.
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
)

type Frame struct {
	Keys   []string
	Arrays []telem.Array
}

func NewFrame(keys []string, arrays []telem.Array) Frame {
	if len(keys) != len(arrays) {
		panic("[cesium] - Keys and telemetry arrays in a frame must be of the same length")
	}
	kf := Frame{Keys: keys, Arrays: arrays}
	return kf
}

func (f Frame) UniqueKeys() []string { return lo.Uniq(f.Keys) }

func (f Frame) Key(i int) string { return f.Keys[i] }

func (f Frame) Append(key string, arr telem.Array) Frame {
	return NewFrame(append(f.Keys, key), append(f.Arrays, arr))
}

func (f Frame) Prepend(key string, arr telem.Array) Frame {
	return NewFrame(append([]string{key}, f.Keys...), append([]telem.Array{arr}, f.Arrays...))
}

func (f Frame) AppendMany(keys []string, arrays []telem.Array) Frame {
	return NewFrame(append(f.Keys, keys...), append(f.Arrays, arrays...))
}

func (f Frame) PrependMany(keys []string, arrays []telem.Array) Frame {
	return NewFrame(append(keys, f.Keys...), append(arrays, f.Arrays...))
}

func (f Frame) AppendFrame(frame Frame) Frame { return f.AppendMany(frame.Keys, frame.Arrays) }

func (f Frame) FilterKeys(keys []string) Frame {
	var (
		filteredKeys   = make([]string, 0, len(keys))
		filteredArrays = make([]telem.Array, 0, len(keys))
	)
	for i, key := range f.Keys {
		if lo.Contains(keys, key) {
			filteredKeys = append(filteredKeys, key)
			filteredArrays = append(filteredArrays, f.Arrays[i])
		}
	}
	return NewFrame(filteredKeys, filteredArrays)
}

func (f Frame) Unary() bool { return len(f.Keys) == len(f.UniqueKeys()) }

func (f Frame) Even() bool {
	for i := 1; i < len(f.Arrays); i++ {
		if f.Arrays[i].Len() != f.Arrays[0].Len() {
			return false
		}
		if f.Arrays[i].TimeRange != f.Arrays[0].TimeRange {
			return false
		}
	}
	return true
}

func (f Frame) Len() int64 {
	f.assertEven("Len")
	return f.Arrays[0].Len()
}

func (f Frame) assertEven(method string) {
	if !f.Even() {
		panic("[telem] - cannot call " + method + " on uneven frame")
	}
}

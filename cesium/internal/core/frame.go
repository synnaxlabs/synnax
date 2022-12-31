// Copyright 2022 Synnax Labs, Inc.
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
	keys []string
	telem.Frame
}

func NewFrame(keys []string, arrays []telem.Array) Frame {
	if len(keys) != len(arrays) {
		panic("[cesium] - keys and telemetry arrays in a frame must be of the same length")
	}
	kf := Frame{keys: keys}
	kf.Arrays = arrays
	return kf
}

func (f Frame) Keys() []string { return f.keys }

func (f Frame) UniqueKeys() []string { return lo.Uniq(f.keys) }

func (f Frame) Unary() bool { return len(f.keys) == len(f.UniqueKeys()) }

func (f Frame) Key(i int) string { return f.keys[i] }

func (f Frame) Append(key string, arr telem.Array) Frame {
	return NewFrame(append(f.keys, key), append(f.Arrays, arr))
}

func (f Frame) Prepend(key string, arr telem.Array) Frame {
	return NewFrame(append([]string{key}, f.keys...), append([]telem.Array{arr}, f.Arrays...))
}

func (f Frame) AppendMany(keys []string, arrays []telem.Array) Frame {
	return NewFrame(append(f.keys, keys...), append(f.Arrays, arrays...))
}

func (f Frame) PrependMany(keys []string, arrays []telem.Array) Frame {
	return NewFrame(append(keys, f.keys...), append(arrays, f.Arrays...))
}

func (f Frame) AppendFrame(frame Frame) Frame { return f.AppendMany(frame.keys, frame.Arrays) }

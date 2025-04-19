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

type Frame = telem.Frame[ChannelKey]

func UnaryFrame(keys []ChannelKey, series []telem.Series) Frame {
	if len(keys) != len(series) {
		panic("keys and telemetry series in a frame must be of the same length")
	}
	kf := Frame{Keys: keys, Series: series}
	return kf
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

func (f Frame) FilterKeys(keys []ChannelKey) Frame {
	return Frame{f.Frame.FilterKeys(keys)}
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

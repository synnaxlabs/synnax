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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

// Frame is an extension of telem.Frame that provides a distribution layer specific
// API.
type Frame struct{ telem.Frame[channel.Key] }

// Append appends a series to the frame with the given key. For more details, see the
// telem.Frame.Append implementation.
func (f Frame) Append(key channel.Key, series telem.Series) Frame {
	return Frame{f.Frame.Append(key, series)}
}

func UnaryFrame(key channel.Key, series telem.Series) Frame {
	return Frame{telem.UnaryFrame(key, series)}
}

func MultiFrame(keys []channel.Key, series []telem.Series) Frame {
	return Frame{telem.MultiFrame(keys, series)}
}

func AllocFrame(cap int) Frame {
	return Frame{telem.AllocFrame[channel.Key](cap)}
}

func (f Frame) SplitByLeaseholder() map[core.NodeKey]Frame {
	frames := make(map[core.NodeKey]Frame)
	for key, ser := range f.Entries() {
		nodeKey := key.Leaseholder()
		frames[nodeKey] = frames[nodeKey].Append(key, ser)
	}
	return frames
}

func (f *Frame) Sort() { f.Frame.Sort() }

func (f Frame) SplitByHost(host core.NodeKey) (local Frame, remote Frame, free Frame) {
	for key, series := range f.Entries() {
		if key.Leaseholder() == host {
			local = local.Append(key, series)
		} else if key.Leaseholder().IsFree() {
			free = free.Append(key, series)
		} else {
			remote = remote.Append(key, series)
		}
	}
	return local, remote, free
}

func (f Frame) ToStorage() (fr ts.Frame) {
	return telem.MultiFrame[cesium.ChannelKey](channel.Keys(f.KeysSlice()).Storage(), f.SeriesSlice())
}

func (f Frame) FilterKeys(keys channel.Keys) Frame {
	return Frame{f.Frame.FilterKeys(keys)}
}

func (f Frame) Extend(frame Frame) Frame {
	return Frame{f.Frame.Extend(frame.Frame)}
}

func (f Frame) ShallowCopy() Frame {
	return Frame{f.Frame.ShallowCopy()}
}

func MergeFrames(frames []Frame) (f Frame) {
	if len(frames) == 0 {
		return f
	}
	if len(frames) == 1 {
		return frames[0]
	}
	for _, frame := range frames {
		for key, series := range frame.Entries() {
			f = f.Append(key, series)
		}
	}
	return f
}

func NewFrameFromStorage(frame ts.Frame) Frame {
	return Frame{telem.UnsafeReinterpretKeysAs[cesium.ChannelKey, channel.Key](frame)}
}

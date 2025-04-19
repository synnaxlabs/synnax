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

type Frame struct{ telem.Frame[channel.Key] }

func (f Frame) Append(key channel.Key, series telem.Series) Frame {
	return Frame{f.Frame.Append(key, series)}
}

func UnaryFrame(key channel.Key, series telem.Series) Frame {
	return Frame{telem.UnaryFrame(key, series)}
}

func MultiFrame(keys []channel.Key, series []telem.Series) Frame {
	return Frame{telem.MultiFrame(keys, series)}
}

func (f Frame) SplitByLeaseholder() map[core.NodeKey]Frame {
	frames := make(map[core.NodeKey]Frame)
	for key, ser := range f.Entries() {
		nodeKey := key.Leaseholder()
		frames[nodeKey] = frames[nodeKey].Append(key, ser)
	}
	return frames
}

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
	return cesium.NewFrame(channel.Keys(f.KeysSlice()).Storage(), f.SeriesSlice())
}

func (f Frame) FilterKeys(keys channel.Keys) Frame {
	return Frame{f.Frame.FilterKeys(keys)}
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
	return MultiFrame(channel.KeysFromUint32(frame.KeysSlice()), frame.SeriesSlice())
}

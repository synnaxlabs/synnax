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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/telem"
)

type Frame struct {
	keys channel.Keys
	telem.Frame
}

func (f Frame) Keys() channel.Keys { return f.keys }

func (f Frame) Vertical() bool { return len(f.keys.Unique()) == len(f.Arrays) }

func (f Frame) SplitByNodeKey() map[core.NodeKey]Frame {
	frames := make(map[core.NodeKey]Frame)
	for i, key := range f.keys {
		nodeKey := key.NodeKey()
		nf, ok := frames[nodeKey]
		if !ok {
			frames[nodeKey] = NewFrame([]channel.Key{key}, []telem.Array{f.Arrays[i]})
		} else {
			nf.keys = append(nf.keys, key)
			nf.Arrays = append(nf.Arrays, f.Arrays[i])
			frames[nodeKey] = nf
		}
	}
	return frames
}

func (f Frame) SplitByHost(host core.NodeKey) (local Frame, remote Frame) {
	for i, key := range f.keys {
		if key.NodeKey() == host {
			local.keys = append(local.keys, key)
			local.Arrays = append(local.Arrays, f.Arrays[i])
		} else {
			remote.keys = append(remote.keys, key)
			remote.Arrays = append(remote.Arrays, f.Arrays[i])
		}
	}
	return local, remote
}

func (f Frame) FilterKeys(keys channel.Keys) Frame {
	var (
		filteredKeys   = make(channel.Keys, 0, len(f.keys))
		filteredArrays = make([]telem.Array, 0, len(f.Arrays))
	)
	for i, key := range f.keys {
		if lo.Contains(keys, key) {
			filteredKeys = append(filteredKeys, key)
			filteredArrays = append(filteredArrays, f.Arrays[i])
		}
	}
	return NewFrame(filteredKeys, filteredArrays)
}

func (f Frame) ToStorage() storage.Frame { return cesium.NewFrame(f.keys.Strings(), f.Arrays) }

func NewFrame(keys channel.Keys, arrays []telem.Array) Frame {
	return Frame{
		keys:  keys,
		Frame: telem.Frame{Arrays: arrays},
	}
}

func MergeFrames(frames []Frame) (f Frame) {
	if len(frames) == 0 {
		return f
	}
	if len(frames) == 1 {
		return frames[0]
	}
	for _, frame := range frames {
		f.keys = append(f.keys, frame.keys...)
		f.Arrays = append(f.Arrays, frame.Arrays...)
	}
	return f
}

func NewFrameFromStorage(frame storage.Frame) Frame {
	keys := make(channel.Keys, len(frame.Arrays))
	for i := range frame.Arrays {
		keys[i] = channel.MustParseKey(frame.Key(i))
	}
	return NewFrame(keys, frame.Arrays)
}

func UnaryFrame(key channel.Key, array telem.Array) Frame {
	return NewFrame(channel.Keys{key}, []telem.Array{array})
}

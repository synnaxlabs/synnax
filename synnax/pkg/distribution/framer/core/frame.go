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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

type Frame struct {
	Keys   channel.Keys  `json:"keys" msgpack:"keys"`
	Arrays []telem.Array `json:"arrays" msgpack:"arrays"`
}

func (f Frame) Vertical() bool { return len(f.Keys.Unique()) == len(f.Arrays) }

func (f Frame) SplitByNodeKey() map[core.NodeKey]Frame {
	frames := make(map[core.NodeKey]Frame)
	for i, key := range f.Keys {
		nodeKey := key.Leaseholder()
		nf, ok := frames[nodeKey]
		if !ok {
			frames[nodeKey] = Frame{
				Keys:   channel.Keys{key},
				Arrays: []telem.Array{f.Arrays[i]},
			}
		} else {
			nf.Keys = append(nf.Keys, key)
			nf.Arrays = append(nf.Arrays, f.Arrays[i])
			frames[nodeKey] = nf
		}
	}
	return frames
}

func (f Frame) SplitByHost(host core.NodeKey) (local Frame, remote Frame) {
	for i, key := range f.Keys {
		if key.Leaseholder() == host {
			local.Keys = append(local.Keys, key)
			local.Arrays = append(local.Arrays, f.Arrays[i])
		} else {
			remote.Keys = append(remote.Keys, key)
			remote.Arrays = append(remote.Arrays, f.Arrays[i])
		}
	}
	return local, remote
}

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

func (f Frame) ToStorage() (fr ts.Frame) {
	fr.Arrays = f.Arrays
	fr.Keys = f.Keys.Storage()
	return fr
}

func (f Frame) FilterKeys(keys channel.Keys) Frame {
	var (
		fKeys   = make(channel.Keys, 0, len(keys))
		fArrays = make([]telem.Array, 0, len(keys))
	)
	for i, key := range f.Keys {
		if keys.Contains(key) {
			fKeys = append(fKeys, key)
			fArrays = append(fArrays, f.Arrays[i])
		}
	}
	return Frame{Keys: fKeys, Arrays: fArrays}
}

func MergeFrames(frames []Frame) (f Frame) {
	if len(frames) == 0 {
		return f
	}
	if len(frames) == 1 {
		return frames[0]
	}
	for _, frame := range frames {
		f.Keys = append(f.Keys, frame.Keys...)
		f.Arrays = append(f.Arrays, frame.Arrays...)
	}
	return f
}

func NewFrameFromStorage(frame ts.Frame) Frame {
	keys := make(channel.Keys, len(frame.Arrays))
	for i := range frame.Arrays {
		keys[i] = channel.MustParseKey(frame.Keys[i])
	}
	return Frame{Keys: keys, Arrays: frame.Arrays}
}

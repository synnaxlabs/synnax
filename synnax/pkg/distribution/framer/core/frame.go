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
	Keys   channel.Keys   `json:"keys" msgpack:"keys"`
	Series []telem.Series `json:"series" msgpack:"series"`
}

func (f Frame) Vertical() bool { return len(f.Keys.Unique()) == len(f.Series) }

func (f Frame) SplitByNodeKey() map[core.NodeKey]Frame {
	frames := make(map[core.NodeKey]Frame)
	for i, key := range f.Keys {
		nodeKey := key.Leaseholder()
		nf, ok := frames[nodeKey]
		if !ok {
			frames[nodeKey] = Frame{
				Keys:   channel.Keys{key},
				Series: []telem.Series{f.Series[i]},
			}
		} else {
			nf.Keys = append(nf.Keys, key)
			nf.Series = append(nf.Series, f.Series[i])
			frames[nodeKey] = nf
		}
	}
	return frames
}

func (f Frame) SplitByHost(host core.NodeKey) (local Frame, remote Frame, free Frame) {
	for i, key := range f.Keys {
		if key.Leaseholder() == host {
			local.Keys = append(local.Keys, key)
			local.Series = append(local.Series, f.Series[i])
		} else if key.Leaseholder().IsFree() {
			free.Keys = append(free.Keys, key)
			free.Series = append(free.Series, f.Series[i])
		} else {
			remote.Keys = append(remote.Keys, key)
			remote.Series = append(remote.Series, f.Series[i])
		}
	}
	return local, remote, free
}

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

func (f Frame) ToStorage() (fr ts.Frame) {
	fr.Series = f.Series
	fr.Keys = f.Keys.Storage()
	return fr
}

func (f Frame) FilterKeys(keys channel.Keys) Frame {
	var (
		fKeys   = make(channel.Keys, 0, len(keys))
		fArrays = make([]telem.Series, 0, len(keys))
	)
	for i, key := range f.Keys {
		if keys.Contains(key) {
			fKeys = append(fKeys, key)
			fArrays = append(fArrays, f.Series[i])
		}
	}
	return Frame{Keys: fKeys, Series: fArrays}
}

func (f Frame) Get(key channel.Key) (series []telem.Series) {
	for i, k := range f.Keys {
		if k == key {
			series = append(series, f.Series[i])
		}
	}
	return series
}

func (f Frame) Extend(fr Frame) Frame {
	f.Keys = append(f.Keys, fr.Keys...)
	f.Series = append(f.Series, fr.Series...)
	return f
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
		f.Series = append(f.Series, frame.Series...)
	}
	return f
}

func NewFrameFromStorage(frame ts.Frame) Frame {
	keys := make(channel.Keys, len(frame.Series))
	for i := range frame.Series {
		keys[i] = channel.Key(frame.Keys[i])
	}
	return Frame{Keys: keys, Series: frame.Series}
}

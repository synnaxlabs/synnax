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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/telem"
)

// Frame is a record that maps channel keys to telemetry series.
// It wraps the base telem.Frame type with channel.Key as the key type, providing
// distribution-specific functionality for handling telemetry data across nodes.
type Frame struct{ telem.Frame[channel.Key] }

// Append adds a new key-series pair to the frame and returns the updated frame.
// The key must be a valid channel.Key and the series must be a valid telem.Series.
func (f Frame) Append(key channel.Key, series telem.Series) Frame {
	return Frame{f.Frame.Append(key, series)}
}

// UnaryFrame creates a new frame containing a single key-series pair.
// This is useful for creating frames with a single channel's data.
func UnaryFrame(key channel.Key, series telem.Series) Frame {
	return Frame{telem.UnaryFrame(key, series)}
}

// MultiFrame creates a new frame containing multiple key-series pairs.
// The keys and series slices must be of equal length, or MultiFrame will panic.
func MultiFrame(keys []channel.Key, series []telem.Series) Frame {
	return Frame{telem.MultiFrame(keys, series)}
}

// AllocFrame allocates a new frame with a capacity that can hold up to the specified
// number of series before a re-allocation, This is useful for pre-allocating frames
// when the expected number of series is known.
func AllocFrame(cap int) Frame {
	return Frame{telem.AllocFrame[channel.Key](cap)}
}

// SplitByLeaseholder splits the frame into multiple frames based on the leaseholder
// node of each channel. Returns a map where each key is a node key and the value is a
// frame containing all series for channels leased by that node.
func (f Frame) SplitByLeaseholder() map[cluster.NodeKey]Frame {
	frames := make(map[cluster.NodeKey]Frame)
	for key, ser := range f.Entries() {
		nodeKey := key.Leaseholder()
		frames[nodeKey] = frames[nodeKey].Append(key, ser)
	}
	return frames
}

// SplitByHost splits the frame into three frames based on the leaseholder of each channel:
// - local: contains series for channels leased by the specified host
// - remote: contains series for channels leased by other hosts
// - free: contains series for channels that are not leased by any host
func (f Frame) SplitByHost(host cluster.NodeKey) (local Frame, remote Frame, free Frame) {
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

func (f Frame) Extend(frame Frame) Frame {
	return Frame{f.Frame.Extend(frame.Frame)}
}

// ToStorage converts the frame to the storage layer frame format.
// This is used when persisting the frame to storage.
func (f Frame) ToStorage() ts.Frame {
	return telem.UnsafeReinterpretFrameKeysAs[channel.Key, cesium.ChannelKey](f.Frame)
}

// KeepKeys returns a new frame containing only the series for the specified keys.
// The original frame is not modified.
func (f Frame) KeepKeys(keys channel.Keys) Frame {
	return Frame{Frame: f.Frame.KeepKeys(keys)}
}

func (f Frame) ExcludeKeys(keys channel.Keys) Frame {
	return Frame{Frame: f.Frame.ExcludeKeys(keys)}
}

// ShallowCopy creates a shallow copy of the frame.
// The keys and series slices are copied, but the series data itself is not duplicated.
func (f Frame) ShallowCopy() Frame {
	return Frame{Frame: f.Frame.ShallowCopy()}
}

// MergeFrames combines multiple frames into a single frame.
func MergeFrames(frames []Frame) (f Frame) {
	if len(frames) == 0 {
		return f
	}
	f = frames[0]
	for _, frame := range frames[1:] {
		f = f.Extend(frame)
	}
	return f
}

// NewFrameFromStorage creates a new distribution layer frame from a storage layer frame.
func NewFrameFromStorage(frame ts.Frame) Frame {
	return Frame{telem.UnsafeReinterpretFrameKeysAs[cesium.ChannelKey, channel.Key](frame)}
}

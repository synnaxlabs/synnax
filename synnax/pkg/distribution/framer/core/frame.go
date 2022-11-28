package core

import (
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

func (f Frame) Unary() bool { return len(f.keys.Unique()) == len(f.Arrays) }

func (f Frame) SplitByNodeID() map[core.NodeID]Frame {
	frames := make(map[core.NodeID]Frame)
	for i, key := range f.keys {
		nodeID := key.NodeID()
		nf := frames[nodeID]
		nf.keys = append(nf.keys, key)
		nf.Arrays = append(nf.Arrays, f.Arrays[i])
	}
	return frames
}

func (f Frame) SplitByHost(host core.NodeID) (local Frame, remote Frame) {
	for i, key := range f.keys {
		if key.NodeID() == host {
			local.keys = append(local.keys, key)
			local.Arrays = append(local.Arrays, f.Arrays[i])
		} else {
			remote.keys = append(remote.keys, key)
			remote.Arrays = append(remote.Arrays, f.Arrays[i])
		}
	}
	return local, remote
}

func (f Frame) StorageFrame() storage.Frame { return cesium.NewFrame(f.keys.Strings(), f.Arrays) }

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

// StorageWrapper wraps slices of storage.Framer into slices of Framer by
// adding the appropriate host information.
type StorageWrapper struct {
	Host core.NodeID
}

// Wrap a telemetry frame from the storage layer
func (cw *StorageWrapper) Wrap(frame storage.Frame) Frame {
	keys := make(channel.Keys, len(frame.Arrays))
	for i := range frame.Arrays {
		keys[i] = channel.MustParseKey(frame.Key(i))
	}
	return NewFrame(keys, frame.Arrays)
}

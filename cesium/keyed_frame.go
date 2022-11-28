package cesium

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

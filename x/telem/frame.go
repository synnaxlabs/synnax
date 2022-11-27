package telem

import "github.com/samber/lo"

type Frame struct {
	Arrays []Array
}

func (f Frame) Even() bool {
	for i := 1; i < len(f.Arrays); i++ {
		if f.Arrays[i].Len() != f.Arrays[0].Len() {
			return false
		}
		if f.Arrays[i].Range != f.Arrays[0].Range {
			return false
		}
	}
	return true
}

func (f Frame) Keyed() bool {
	for _, a := range f.Arrays {
		if !a.Keyed() {
			return false
		}
	}
	return true
}

func (f Frame) keys() []string {
	keys := make([]string, len(f.Arrays))
	for i, a := range f.Arrays {
		keys[i] = a.Key
	}
	return keys
}

func (f Frame) Keys() []string {
	return lo.Uniq(f.keys())
}

func (f Frame) NumKeys() int { return len(f.Keys()) }

func (f Frame) Unique() bool { return len(f.Keys()) == len(f.Arrays) }

func (f Frame) Len() int64 {
	f.assertEven("Len")
	return f.Arrays[0].Len()
}

func (f Frame) assertEven(method string) {
	if !f.Even() {
		panic("[telem] - cannot call " + method + " on uneven frame")
	}
}

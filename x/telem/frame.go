package telem

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

func (f Frame) Len() int64 {
	f.assertEven("Len")
	return f.Arrays[0].Len()
}

func (f Frame) assertEven(method string) {
	if !f.Even() {
		panic("[telem] - cannot call " + method + " on uneven frame")
	}
}
